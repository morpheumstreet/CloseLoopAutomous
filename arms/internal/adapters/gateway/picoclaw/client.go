// Package picoclaw implements dispatch to a [PicoClaw] Pico Protocol WebSocket endpoint
// (JSON messages: message.send with session_id and payload.content), as used by the
// pico_client channel when dialing a remote server. See upstream:
// https://github.com/sipeed/picoclaw/blob/main/pkg/channels/pico/protocol.go
//
// [PicoClaw]: https://github.com/sipeed/picoclaw
package picoclaw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

const (
	typeMessageSend   = "message.send"
	typePing          = "ping"
	typePong          = "pong"
	typeError         = "error"
	typeMessageCreate = "message.create"
)

// wireMsg matches PicoClaw pkg/channels/pico.PicoMessage.
type wireMsg struct {
	Type      string         `json:"type"`
	ID        string         `json:"id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Timestamp int64          `json:"timestamp,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// Options configure the Pico Protocol WebSocket client.
type Options struct {
	URL    string
	Token  string // optional; sent as Authorization: Bearer
	Timeout time.Duration
	// KnowledgeForDispatch appends ranked snippets to dispatch bodies when non-nil (same hook as OpenClaw).
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
}

// Client maintains one WebSocket and a background reader (handles JSON ping / drains server pushes).
type Client struct {
	opts          Options
	mu            sync.Mutex
	conn          *websocket.Conn
	readCancel    context.CancelFunc
	readDone      sync.WaitGroup
}

// New constructs a Client. Empty URL yields an error on first dispatch.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &Client{opts: opts}
}

// Close shuts down the reader and connection.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dropConnLocked()
}

func (c *Client) dropConnLocked() error {
	if c.readCancel != nil {
		c.readCancel()
		c.readCancel = nil
	}
	c.readDone.Wait()
	var err error
	if c.conn != nil {
		err = c.conn.Close(websocket.StatusNormalClosure, "arms close")
		c.conn = nil
	}
	return err
}

func (c *Client) callContext(parent context.Context) (context.Context, context.CancelFunc) {
	if c.opts.Timeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, c.opts.Timeout)
}

func (c *Client) ensureConnLocked(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}
	u := strings.TrimSpace(c.opts.URL)
	if u == "" {
		return errors.New("picoclaw: gateway URL is required")
	}
	hdr := http.Header{}
	if tok := strings.TrimSpace(c.opts.Token); tok != "" {
		hdr.Set("Authorization", "Bearer "+tok)
	}
	conn, _, err := websocket.Dial(ctx, u, &websocket.DialOptions{HTTPHeader: hdr})
	if err != nil {
		return fmt.Errorf("picoclaw dial: %w", err)
	}
	c.conn = conn
	rctx, cancel := context.WithCancel(context.Background())
	c.readCancel = cancel
	c.readDone.Add(1)
	go func() {
		defer c.readDone.Done()
		c.pumpReads(rctx, conn)
	}()
	return nil
}

func (c *Client) pumpReads(ctx context.Context, conn *websocket.Conn) {
	for {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg wireMsg
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		switch msg.Type {
		case typePing:
			pong := wireMsg{Type: typePong, ID: msg.ID, Timestamp: time.Now().UnixMilli()}
			b, _ := json.Marshal(pong)
			_ = conn.Write(context.Background(), websocket.MessageText, b)
		case typeError, typeMessageCreate:
			// ignore; completion still uses ARMS webhooks / operator flow
		default:
		}
	}
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

// DispatchTaskWithSession sends message.send with payload.content = ARMS task markdown.
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionID string) (string, error) {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return "", errors.New("picoclaw: session_id (execution agent session_key) is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	if err := c.ensureConnLocked(callCtx); err != nil {
		return "", err
	}
	kb := c.knowledgeMarkdown(callCtx, task.ProductID, knowledgeQueryFromTask(task))
	body := openclaw.TaskDispatchMarkdown(task, kb)
	id := uuid.NewString()
	msg := wireMsg{
		Type:      typeMessageSend,
		ID:        id,
		SessionID: sid,
		Timestamp: time.Now().UnixMilli(),
		Payload:   map[string]any{"content": body},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	if err := c.conn.Write(callCtx, websocket.MessageText, b); err != nil {
		_ = c.dropConnLocked()
		return "", fmt.Errorf("picoclaw write: %w", err)
	}
	return id, nil
}

// DispatchSubtaskWithSession sends message.send for a convoy subtask.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionID string) (string, error) {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return "", errors.New("picoclaw: session_id (execution agent session_key) is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	if err := c.ensureConnLocked(callCtx); err != nil {
		return "", err
	}
	kb := c.knowledgeMarkdown(callCtx, parent.ProductID, knowledgeQueryFromSubtask(parent, sub))
	body := openclaw.SubtaskDispatchMarkdown(parent.ID, sub, kb)
	id := uuid.NewString()
	msg := wireMsg{
		Type:      typeMessageSend,
		ID:        id,
		SessionID: sid,
		Timestamp: time.Now().UnixMilli(),
		Payload:   map[string]any{"content": body},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	if err := c.conn.Write(callCtx, websocket.MessageText, b); err != nil {
		_ = c.dropConnLocked()
		return "", fmt.Errorf("picoclaw write: %w", err)
	}
	return id, nil
}
