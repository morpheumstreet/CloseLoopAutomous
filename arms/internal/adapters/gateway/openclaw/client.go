package openclaw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// Options configure the WebSocket gateway client for OpenClaw-class protocols
// (connect.challenge / connect, chat.send). Drivers zeroclaw_ws, clawlet_ws, and ironclaw_ws use thin wrappers around this type.
// Stock NullClaw uses HTTP /a2a instead; see package nullclaw (Client) and driver nullclaw_a2a.
type Options struct {
	URL      string
	Token    string
	DeviceID string // optional extra header (not the Ed25519 device block MC uses)
	// SessionKey is passed to chat.send as sessionKey (e.g. agent:main:mission-control-builder).
	// Set ARMS_OPENCLAW_SESSION_KEY to match your gateway agent session.
	SessionKey string
	Timeout    time.Duration // per Dispatch* (handshake + RPC)
	MinProto   int           // default 3
	MaxProto   int           // default 3
	// KnowledgeForDispatch appends ranked snippets to dispatch markdown when non-nil (#90).
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
}

// Client speaks native OpenClaw gateway JSON over WebSocket.
type Client struct {
	opts          Options
	mu            sync.Mutex
	conn          *websocket.Conn
	authenticated bool
}

// New constructs a Client. Empty SessionKey yields a clear error on Dispatch when URL is set.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.MinProto <= 0 {
		opts.MinProto = 3
	}
	if opts.MaxProto <= 0 {
		opts.MaxProto = 3
	}
	return &Client{opts: opts}
}

var _ ports.AgentGateway = (*Client)(nil)

// Close drops the cached connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dropConnLocked()
}

func (c *Client) dropConnLocked() error {
	c.authenticated = false
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close(websocket.StatusNormalClosure, "arms close")
	c.conn = nil
	return err
}

func mergeURLToken(rawURL, token string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token) != "" {
		q := u.Query()
		q.Set("token", token)
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func (c *Client) dialLocked(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}
	wsURL, err := mergeURLToken(c.opts.URL, c.opts.Token)
	if err != nil {
		return fmt.Errorf("openclaw url: %w", err)
	}
	hdr := http.Header{}
	if tok := strings.TrimSpace(c.opts.Token); tok != "" {
		hdr.Set("Authorization", "Bearer "+tok)
	}
	if id := strings.TrimSpace(c.opts.DeviceID); id != "" {
		hdr.Set("X-Arms-Device-Id", id)
	}
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: hdr})
	if err != nil {
		return fmt.Errorf("openclaw dial: %w", err)
	}
	c.conn = conn
	c.authenticated = false
	return nil
}

// ensureAuthedLocked performs connect.challenge → connect RPC (Mission Control flow).
func (c *Client) ensureAuthedLocked(ctx context.Context) error {
	if c.authenticated && c.conn != nil {
		return nil
	}
	_ = c.dropConnLocked()
	if err := c.dialLocked(ctx); err != nil {
		return err
	}

	for {
		fr, raw, err := readJSONFrame(ctx, c.conn)
		if err != nil {
			_ = c.dropConnLocked()
			return err
		}
		if fr.Type == "event" && fr.Event == "connect.challenge" {
			if err := c.answerChallengeLocked(ctx, raw); err != nil {
				_ = c.dropConnLocked()
				return err
			}
			c.authenticated = true
			return nil
		}
	}
}

func (c *Client) answerChallengeLocked(ctx context.Context, challengeRaw []byte) error {
	var wrap struct {
		Payload struct {
			Nonce string `json:"nonce"`
		} `json:"payload"`
	}
	_ = json.Unmarshal(challengeRaw, &wrap)
	_ = wrap.Payload.Nonce

	reqID := uuid.NewString()
	params := map[string]any{
		"minProtocol": c.opts.MinProto,
		"maxProtocol": c.opts.MaxProto,
		"client": map[string]any{
			"id":       "arms",
			"version":  "0.1.0",
			"platform": "go",
			"mode":     "orchestrator",
		},
		"auth":   map[string]any{"token": c.opts.Token},
		"role":   "operator",
		"scopes": []string{"operator.admin"},
	}
	msg := map[string]any{
		"type":   "req",
		"id":     reqID,
		"method": "connect",
		"params": params,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := c.conn.Write(ctx, websocket.MessageText, b); err != nil {
		return fmt.Errorf("openclaw connect write: %w", err)
	}
	return c.waitResLocked(ctx, reqID)
}

// readJSONFrame skips non-JSON frames until a valid object is received or ctx expires.
func readJSONFrame(ctx context.Context, conn *websocket.Conn) (frame, []byte, error) {
	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			return frame{}, nil, fmt.Errorf("openclaw read: %w", err)
		}
		var fr frame
		if json.Unmarshal(b, &fr) != nil || fr.Type == "" {
			continue
		}
		return fr, b, nil
	}
}

type frame struct {
	Type    string          `json:"type"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Event   string          `json:"event"`
	OK      *bool           `json:"ok"`
	Error   *rpcErrBody     `json:"error"`
	Payload json.RawMessage `json:"payload"`
}

type rpcErrBody struct {
	Message string `json:"message"`
}

func idKey(raw json.RawMessage) string {
	raw = bytesTrimSpaceJSON(raw)
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n int64
	if json.Unmarshal(raw, &n) == nil {
		return fmt.Sprintf("%d", n)
	}
	return string(raw)
}

func bytesTrimSpaceJSON(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func (c *Client) waitResLocked(ctx context.Context, wantID string) error {
	for {
		fr, _, err := readJSONFrame(ctx, c.conn)
		if err != nil {
			return err
		}
		if fr.Type != "res" {
			continue
		}
		if idKey(fr.ID) != wantID {
			continue
		}
		if fr.OK != nil && !*fr.OK {
			msg := "connect failed"
			if fr.Error != nil && strings.TrimSpace(fr.Error.Message) != "" {
				msg = fr.Error.Message
			}
			return errors.New(msg)
		}
		return nil
	}
}

func (c *Client) rpcLocked(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	reqID := uuid.NewString()
	msg := map[string]any{"type": "req", "id": reqID, "method": method, "params": params}
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	if err := c.conn.Write(ctx, websocket.MessageText, b); err != nil {
		return nil, fmt.Errorf("openclaw write: %w", err)
	}
	for {
		fr, _, err := readJSONFrame(ctx, c.conn)
		if err != nil {
			return nil, err
		}
		if fr.Type != "res" || idKey(fr.ID) != reqID {
			continue
		}
		if fr.OK != nil && !*fr.OK {
			msg := "gateway error"
			if fr.Error != nil && strings.TrimSpace(fr.Error.Message) != "" {
				msg = fr.Error.Message
			}
			return nil, errors.New(msg)
		}
		return fr.Payload, nil
	}
}

func (c *Client) DispatchTask(ctx context.Context, task domain.Task) (string, error) {
	return c.DispatchTaskWithSession(ctx, task, strings.TrimSpace(c.opts.SessionKey))
}

// DispatchTaskWithSession sends chat.send with the given sessionKey (per logical agent on the gateway).
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	sk := strings.TrimSpace(sessionKey)
	if sk == "" {
		return "", errors.New("openclaw: sessionKey required for chat.send")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	if err := c.ensureAuthedLocked(callCtx); err != nil {
		return "", err
	}

	kb := c.knowledgeMarkdown(callCtx, task.ProductID, KnowledgeQueryFromTask(task))
	msg := TaskDispatchMarkdown(task, kb)
	params := map[string]any{
		"sessionKey":     sk,
		"message":        msg,
		"idempotencyKey": fmt.Sprintf("arms-dispatch-%s-%d", task.ID, time.Now().UnixNano()),
	}
	payload, err := c.rpcLocked(callCtx, "chat.send", params)
	if err != nil {
		_ = c.dropConnLocked()
		return "", err
	}
	return refFromPayload(payload), nil
}

func (c *Client) DispatchSubtask(ctx context.Context, parent domain.Task, sub domain.Subtask) (string, error) {
	return c.DispatchSubtaskWithSession(ctx, parent, sub, strings.TrimSpace(c.opts.SessionKey))
}

// DispatchSubtaskWithSession sends chat.send for a convoy subtask with the given sessionKey.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	sk := strings.TrimSpace(sessionKey)
	if sk == "" {
		return "", errors.New("openclaw: sessionKey required for chat.send")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	if err := c.ensureAuthedLocked(callCtx); err != nil {
		return "", err
	}

	kb := c.knowledgeMarkdown(callCtx, parent.ProductID, KnowledgeQueryFromSubtask(parent, sub))
	msg := SubtaskDispatchMarkdown(parent.ID, sub, kb)
	params := map[string]any{
		"sessionKey":     sk,
		"message":        msg,
		"idempotencyKey": fmt.Sprintf("arms-subtask-%s-%s-%d", parent.ID, sub.ID, time.Now().UnixNano()),
	}
	payload, err := c.rpcLocked(callCtx, "chat.send", params)
	if err != nil {
		_ = c.dropConnLocked()
		return "", err
	}
	return refFromPayload(payload), nil
}

func (c *Client) callContext(parent context.Context) (context.Context, context.CancelFunc) {
	if c.opts.Timeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, c.opts.Timeout)
}

func (c *Client) knowledgeMarkdown(ctx context.Context, pid domain.ProductID, q string) string {
	if c.opts.KnowledgeForDispatch == nil {
		return ""
	}
	s, err := c.opts.KnowledgeForDispatch(ctx, pid, q)
	if err != nil || strings.TrimSpace(s) == "" {
		return ""
	}
	return s
}

func refFromPayload(raw json.RawMessage) string {
	raw = bytesTrimSpaceJSON(raw)
	if len(raw) == 0 {
		return "sent"
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return string(raw)
	}
	for _, k := range []string{"id", "messageId", "message_id", "runId", "run_id", "sessionId", "session_id"} {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" {
				return s
			}
		}
	}
	return string(raw)
}
