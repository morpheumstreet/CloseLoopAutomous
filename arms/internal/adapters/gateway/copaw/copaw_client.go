// Package copaw dispatches tasks to [CoPaw] via JSON-RPC 2.0 POST to the Console API (…/console/api, method chat.send).
//
// [CoPaw]: https://github.com/agentscope-ai/CoPaw
package copaw

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

const rpcMethodChatSend = "chat.send"

// Options configure the CoPaw HTTP JSON-RPC client.
type Options struct {
	// BaseURL is the CoPaw HTTP origin, e.g. http://127.0.0.1:8088. Requests POST to …/console/api.
	BaseURL string
	Token   string
	// Workspace is the CoPaw workspace id (maps from gateway_endpoints.device_id).
	Workspace string
	Timeout   time.Duration
	// KnowledgeForDispatch appends ranked snippets to the prompt when non-nil (same hook as other gateways).
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
	HTTPClient           *http.Client
}

// Client speaks CoPaw Console JSON-RPC (chat.send).
type Client struct {
	opts Options
}

// New constructs a Client. Empty BaseURL or Workspace yields an error on dispatch.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &Client{opts: opts}
}

// Close is a no-op (HTTP); satisfies pooling like other gateway clients.
func (*Client) Close() error { return nil }

func joinConsoleAPIURL(raw string) string {
	s := strings.TrimRight(strings.TrimSpace(raw), "/")
	if s == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(s), "/console/api") {
		return s
	}
	return s + "/console/api"
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

func (c *Client) postJSONRPC(ctx context.Context, method string, params any) (json.RawMessage, error) {
	endpoint := joinConsoleAPIURL(c.opts.BaseURL)
	if endpoint == "" {
		return nil, errors.New("copaw: base URL is required")
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
		return nil, fmt.Errorf("copaw: %w", err)
	}
	defer res.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("copaw: read body: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("copaw: HTTP %s: %s", res.Status, bytesTrimPreview(respBody))
	}
	var wrap jsonrpcResponse
	if err := json.Unmarshal(respBody, &wrap); err != nil {
		return nil, fmt.Errorf("copaw: invalid JSON: %w", err)
	}
	if wrap.Error != nil {
		msg := strings.TrimSpace(wrap.Error.Message)
		if msg == "" {
			msg = "rpc error"
		}
		return nil, fmt.Errorf("copaw: %s (code %d)", msg, wrap.Error.Code)
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

// DispatchTaskWithSession sends chat.send. sessionKey is the CoPaw chat/session identifier; workspace comes from Options.Workspace (endpoint device_id).
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	ws := strings.TrimSpace(c.opts.Workspace)
	if ws == "" {
		return "", errors.New("copaw: device_id (workspace) is required on the gateway endpoint")
	}
	sk := strings.TrimSpace(sessionKey)
	if sk == "" {
		return "", errors.New("copaw: session_key is required")
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	kb := c.knowledgeMarkdown(callCtx, task.ProductID, openclaw.KnowledgeQueryFromTask(task))
	prompt := openclaw.TaskDispatchMarkdown(task, kb)
	params := map[string]any{
		"workspace":   ws,
		"prompt":      prompt,
		"session_key": sk,
	}
	raw, err := c.postJSONRPC(callCtx, rpcMethodChatSend, params)
	if err != nil {
		return "", err
	}
	return refFromRPCResult(raw), nil
}

// DispatchSubtaskWithSession sends chat.send for a convoy subtask.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	ws := strings.TrimSpace(c.opts.Workspace)
	if ws == "" {
		return "", errors.New("copaw: device_id (workspace) is required on the gateway endpoint")
	}
	sk := strings.TrimSpace(sessionKey)
	if sk == "" {
		return "", errors.New("copaw: session_key is required")
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	kb := c.knowledgeMarkdown(callCtx, parent.ProductID, openclaw.KnowledgeQueryFromSubtask(parent, sub))
	prompt := openclaw.SubtaskDispatchMarkdown(parent.ID, sub, kb)
	params := map[string]any{
		"workspace":   ws,
		"prompt":      prompt,
		"session_key": sk,
	}
	raw, err := c.postJSONRPC(callCtx, rpcMethodChatSend, params)
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
	for _, k := range []string{"id", "taskId", "task_id", "messageId", "message_id", "request_id", "requestId"} {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}
