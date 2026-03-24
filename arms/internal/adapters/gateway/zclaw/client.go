// Package zclaw dispatches tasks to a [zclaw] web relay over HTTP: POST /api/chat with JSON {"message":"…"}.
// Upstream: https://github.com/tnm/zclaw (scripts/web_relay.py). Optional auth: header X-Zclaw-Key when the relay sets ZCLAW_WEB_API_KEY.
//
// [zclaw]: https://github.com/tnm/zclaw
package zclaw

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
	"unicode/utf8"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// maxMessageRunes matches scripts/web_relay.py MAX_CHAT_MESSAGE_LEN (4096).
const maxMessageRunes = 4096

// Options configure the zclaw web-relay HTTP client.
type Options struct {
	// BaseURL is the relay origin, e.g. http://127.0.0.1:8787. POST targets …/api/chat.
	BaseURL string
	// Token is sent as X-Zclaw-Key when non-empty (relay api_key).
	Token   string
	Timeout time.Duration
	// KnowledgeForDispatch appends ranked snippets to dispatch bodies when non-nil (same hook as OpenClaw).
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
	HTTPClient           *http.Client
}

// Client speaks zclaw web relay /api/chat.
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

// Close is a no-op (HTTP); matches pooling alongside WebSocket clients.
func (*Client) Close() error { return nil }

func joinChatURL(raw string) string {
	s := strings.TrimRight(strings.TrimSpace(raw), "/")
	if s == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(s), "/api/chat") {
		return s
	}
	return s + "/api/chat"
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

func truncateToMaxRunes(s string) string {
	if utf8.RuneCountInString(s) <= maxMessageRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxMessageRunes])
}

type chatOKResponse struct {
	Reply        string `json:"reply"`
	BridgeTarget string `json:"bridge_target"`
	ElapsedMS    int    `json:"elapsed_ms"`
}

type chatErrBody struct {
	Error string `json:"error"`
}

func (c *Client) postChat(ctx context.Context, message string) (externalRef string, err error) {
	endpoint := joinChatURL(c.opts.BaseURL)
	if endpoint == "" {
		return "", errors.New("zclaw: base URL is required")
	}
	body, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := strings.TrimSpace(c.opts.Token); tok != "" {
		req.Header.Set("X-Zclaw-Key", tok)
	}
	hc := c.opts.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("zclaw: %w", err)
	}
	defer res.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("zclaw: read body: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var eb chatErrBody
		if json.Unmarshal(respBody, &eb) == nil && strings.TrimSpace(eb.Error) != "" {
			return "", fmt.Errorf("zclaw: HTTP %s: %s", res.Status, eb.Error)
		}
		return "", fmt.Errorf("zclaw: HTTP %s: %s", res.Status, bytesTrimPreview(respBody))
	}
	var ok chatOKResponse
	if err := json.Unmarshal(respBody, &ok); err != nil {
		return "", fmt.Errorf("zclaw: invalid JSON: %w", err)
	}
	ref := fmt.Sprintf("zclaw:%dms", ok.ElapsedMS)
	if strings.TrimSpace(ok.BridgeTarget) != "" {
		ref += ":" + strings.TrimSpace(ok.BridgeTarget)
	}
	return ref, nil
}

func bytesTrimPreview(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 512 {
		return s[:512] + "…"
	}
	return s
}

// DispatchTaskWithSession posts the task dispatch body to the relay. sessionKey must be non-empty (arms routing); it is not sent on the wire (single serial bridge per relay).
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	if strings.TrimSpace(sessionKey) == "" {
		return "", errors.New("zclaw: session_key is required (use any non-empty label, e.g. default)")
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	kb := c.knowledgeMarkdown(callCtx, task.ProductID, openclaw.KnowledgeQueryFromTask(task))
	text := truncateToMaxRunes(openclaw.TaskDispatchMarkdown(task, kb))
	return c.postChat(callCtx, text)
}

// DispatchSubtaskWithSession posts a convoy subtask dispatch to the relay.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	if strings.TrimSpace(sessionKey) == "" {
		return "", errors.New("zclaw: session_key is required (use any non-empty label, e.g. default)")
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	kb := c.knowledgeMarkdown(callCtx, parent.ProductID, openclaw.KnowledgeQueryFromSubtask(parent, sub))
	text := truncateToMaxRunes(openclaw.SubtaskDispatchMarkdown(parent.ID, sub, kb))
	return c.postChat(callCtx, text)
}
