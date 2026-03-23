// Package metaclaw dispatches tasks to a [MetaClaw]-style OpenAI-compatible proxy: POST …/v1/chat/completions.
// MetaClaw sits in front of agent runtimes and exposes standard OpenAI HTTP APIs; arms sends one user message
// built from the same markdown dispatch helpers as other gateways.
//
// CloseLoop gateway_endpoints mapping: gateway_url = MetaClaw HTTP origin (e.g. https://127.0.0.1:8765; /v1/chat/completions is appended when missing);
// gateway_token = optional Bearer API key; device_id = optional JSON model override (OpenAI "model" field); execution agent session_key must be non-empty
// (arms validation) and is sent as OpenAI "user" for traceability when set.
//
// Wire contract: OpenAI-compatible POST /v1/chat/completions (same surface many “claw” proxies expose).
package metaclaw

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

	"github.com/closeloopautomous/arms/internal/adapters/gateway/openclaw"
	"github.com/closeloopautomous/arms/internal/domain"
)

const defaultModel = "gpt-4o-mini"

// Options configure the MetaClaw OpenAI-compatible HTTP client.
type Options struct {
	// BaseURL is the proxy origin, e.g. https://metaclaw.example:8080 (POST …/v1/chat/completions).
	BaseURL string
	// Token is sent as Authorization: Bearer when non-empty.
	Token string
	// ModelOverride is sent as JSON "model" when non-empty (from gateway endpoint device_id).
	ModelOverride string
	Timeout time.Duration
	// KnowledgeForDispatch appends ranked snippets to dispatch bodies when non-nil (same hook as OpenClaw).
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
	HTTPClient           *http.Client
}

// Client speaks OpenAI-compatible POST /v1/chat/completions.
type Client struct {
	opts Options
}

// New constructs a Client.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &Client{opts: opts}
}

// Close is a no-op (HTTP); matches pooling alongside WebSocket clients.
func (*Client) Close() error { return nil }

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

func (c *Client) httpClient() *http.Client {
	if c.opts.HTTPClient != nil {
		return c.opts.HTTPClient
	}
	return http.DefaultClient
}

// chatCompletionsURL normalizes gateway_url to a full …/v1/chat/completions endpoint.
func chatCompletionsURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		s = "https://" + s
		lower = strings.ToLower(s)
	}
	s = strings.TrimRight(s, "/")
	lower = strings.ToLower(s)
	if strings.HasSuffix(lower, "/v1/chat/completions") {
		return s
	}
	if strings.HasSuffix(lower, "/v1") {
		return s + "/chat/completions"
	}
	return s + "/v1/chat/completions"
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string         `json:"model"`
	Messages    []chatMessage  `json:"messages"`
	Temperature float64        `json:"temperature,omitempty"`
	User        string         `json:"user,omitempty"`
}

type apiErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

type chatCompletionResponse struct {
	ID    string        `json:"id"`
	Error *apiErrorBody `json:"error,omitempty"`
}

func bytesTrimPreview(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 512 {
		return s[:512] + "…"
	}
	return s
}

func (c *Client) postChatCompletion(ctx context.Context, model, user, userContent string) (string, error) {
	endpoint := chatCompletionsURL(c.opts.BaseURL)
	if endpoint == "" {
		return "", errors.New("metaclaw: gateway_url (base URL) is required")
	}
	m := strings.TrimSpace(model)
	if m == "" {
		m = defaultModel
	}
	reqBody := chatCompletionRequest{
		Model:       m,
		Messages:    []chatMessage{{Role: "user", Content: userContent}},
		Temperature: 0.7,
	}
	if u := strings.TrimSpace(user); u != "" {
		reqBody.User = u
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := strings.TrimSpace(c.opts.Token); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	res, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("metaclaw: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("metaclaw: read body: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var wrap chatCompletionResponse
		if json.Unmarshal(raw, &wrap) == nil && wrap.Error != nil && strings.TrimSpace(wrap.Error.Message) != "" {
			return "", fmt.Errorf("metaclaw: HTTP %s: %s", res.Status, wrap.Error.Message)
		}
		return "", fmt.Errorf("metaclaw: HTTP %s: %s", res.Status, bytesTrimPreview(raw))
	}
	var out chatCompletionResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("metaclaw: invalid JSON: %w", err)
	}
	if out.Error != nil && strings.TrimSpace(out.Error.Message) != "" {
		return "", fmt.Errorf("metaclaw: %s", out.Error.Message)
	}
	if strings.TrimSpace(out.ID) == "" {
		return "", errors.New("metaclaw: response missing id")
	}
	return strings.TrimSpace(out.ID), nil
}

// DispatchTaskWithSession posts a chat completion for the task. sessionKey must be non-empty (arms routing).
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	if strings.TrimSpace(sessionKey) == "" {
		return "", errors.New("metaclaw: session_key is required (use any non-empty label, e.g. default)")
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	kb := c.knowledgeMarkdown(callCtx, task.ProductID, openclaw.KnowledgeQueryFromTask(task))
	text := openclaw.TaskDispatchMarkdown(task, kb)
	model := strings.TrimSpace(c.opts.ModelOverride)
	return c.postChatCompletion(callCtx, model, sessionKey, text)
}

// DispatchSubtaskWithSession posts a chat completion for a convoy subtask.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	if strings.TrimSpace(sessionKey) == "" {
		return "", errors.New("metaclaw: session_key is required (use any non-empty label, e.g. default)")
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	kb := c.knowledgeMarkdown(callCtx, parent.ProductID, openclaw.KnowledgeQueryFromSubtask(parent, sub))
	text := openclaw.SubtaskDispatchMarkdown(parent.ID, sub, kb)
	model := strings.TrimSpace(c.opts.ModelOverride)
	return c.postChatCompletion(callCtx, model, sessionKey, text)
}
