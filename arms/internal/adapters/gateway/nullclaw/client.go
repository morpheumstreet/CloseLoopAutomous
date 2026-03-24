// Client implements dispatch to a stock NullClaw gateway via JSON-RPC 2.0 POST /a2a
// (Google A2A v0.3.0 message/send). See https://github.com/nullclaw/nullclaw docs/en/gateway-api.md
package nullclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Options configure the NullClaw HTTP JSON-RPC client.
type Options struct {
	// BaseURL is the gateway HTTP origin, e.g. http://127.0.0.1:3000. The client POSTs to …/a2a.
	// NullClaw must have a2a.enabled set in config (see upstream gateway-api.md).
	BaseURL string
	Token   string
	Timeout time.Duration
	// KnowledgeForDispatch appends ranked snippets to dispatch bodies when non-nil (same hook as OpenClaw).
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
	// HTTPClient is optional; when nil, http.DefaultClient is used (timeouts still come from callContext).
	HTTPClient *http.Client
}

// Client speaks NullClaw /a2a (message/send).
type Client struct {
	opts Options
}

// New constructs a Client. Empty BaseURL yields an error on dispatch.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &Client{opts: opts}
}

// Close is a no-op (HTTP); satisfies pooling same as WebSocket clients.
func (*Client) Close() error { return nil }

func joinA2AURL(raw string) string {
	s := strings.TrimRight(strings.TrimSpace(raw), "/")
	if s == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(s), "/a2a") {
		return s
	}
	return s + "/a2a"
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

type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonrpcError   `json:"error"`
}

type jsonrpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) postJSONRPC(ctx context.Context, method string, params any) (json.RawMessage, error) {
	endpoint := joinA2AURL(c.opts.BaseURL)
	if endpoint == "" {
		return nil, errors.New("nullclaw: base URL is required")
	}
	id := uuid.NewString()
	body, err := json.Marshal(jsonrpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := strings.TrimSpace(c.opts.Token); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	hc := c.opts.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nullclaw: %w", err)
	}
	defer res.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("nullclaw: read body: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("nullclaw: HTTP %s: %s", res.Status, bytesTrimPreview(respBody))
	}
	var wrap jsonrpcResponse
	if err := json.Unmarshal(respBody, &wrap); err != nil {
		return nil, fmt.Errorf("nullclaw: invalid JSON: %w", err)
	}
	if wrap.Error != nil {
		msg := strings.TrimSpace(wrap.Error.Message)
		if msg == "" {
			msg = "rpc error"
		}
		return nil, fmt.Errorf("nullclaw: %s (code %d)", msg, wrap.Error.Code)
	}
	return wrap.Result, nil
}

func bytesTrimPreview(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 512 {
		return s[:512] + "…"
	}
	return s
}

// DispatchTaskWithSession sends message/send; sessionKey is passed as A2A contextId.
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	sk := strings.TrimSpace(sessionKey)
	if sk == "" {
		return "", errors.New("nullclaw: session_key (A2A contextId) is required")
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	kb := c.knowledgeMarkdown(callCtx, task.ProductID, openclaw.KnowledgeQueryFromTask(task))
	text := openclaw.TaskDispatchMarkdown(task, kb)
	msgID := fmt.Sprintf("arms-task-%s-%d", task.ID, time.Now().UnixNano())
	params := map[string]any{
		"message": map[string]any{
			"messageId": msgID,
			"contextId": sk,
			"role":      "user",
			"parts": []map[string]any{
				{"kind": "text", "text": text},
			},
		},
	}
	raw, err := c.postJSONRPC(callCtx, "message/send", params)
	if err != nil {
		return "", err
	}
	return refFromRPCResult(raw), nil
}

// DispatchSubtaskWithSession sends message/send for a convoy subtask.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	sk := strings.TrimSpace(sessionKey)
	if sk == "" {
		return "", errors.New("nullclaw: session_key (A2A contextId) is required")
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	kb := c.knowledgeMarkdown(callCtx, parent.ProductID, openclaw.KnowledgeQueryFromSubtask(parent, sub))
	text := openclaw.SubtaskDispatchMarkdown(parent.ID, sub, kb)
	msgID := fmt.Sprintf("arms-sub-%s-%s-%d", parent.ID, sub.ID, time.Now().UnixNano())
	params := map[string]any{
		"message": map[string]any{
			"messageId": msgID,
			"contextId": sk,
			"role":      "user",
			"parts": []map[string]any{
				{"kind": "text", "text": text},
			},
		},
	}
	raw, err := c.postJSONRPC(callCtx, "message/send", params)
	if err != nil {
		return "", err
	}
	return refFromRPCResult(raw), nil
}

func refFromRPCResult(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "sent"
	}
	var top map[string]any
	if json.Unmarshal(raw, &top) != nil {
		return string(raw)
	}
	if r := pickTaskRef(top); r != "" {
		return r
	}
	if inner, ok := top["task"].(map[string]any); ok {
		if r := pickTaskRef(inner); r != "" {
			return r
		}
	}
	return string(raw)
}

func pickTaskRef(m map[string]any) string {
	for _, k := range []string{"id", "taskId", "task_id", "messageId", "message_id"} {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}
