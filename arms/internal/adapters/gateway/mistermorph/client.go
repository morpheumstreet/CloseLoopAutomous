// Package mistermorph dispatches tasks to a [MisterMorph] daemon/runtime HTTP API (same contract as
// `mistermorph submit --server-url … --wait`): POST /tasks, then poll GET /tasks/{id} until a terminal status.
//
// CloseLoop gateway_endpoints mapping: gateway_url = runtime base URL; gateway_token = Bearer token
// (server.auth_token); device_id = optional model override; execution agent session_key = optional topic_id
// (JSON topic_id on submit).
//
// [MisterMorph]: https://github.com/quailyquaily/mistermorph
package mistermorph

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

// Options configure the MisterMorph runtime HTTP client.
type Options struct {
	// BaseURL is the runtime origin, e.g. https://agent.example.com:8787 (POST …/tasks).
	BaseURL string
	// Token is the runtime Bearer token (server.auth_token on the MisterMorph side).
	Token string
	// ModelOverride is sent as JSON "model" when non-empty (from gateway endpoint device_id).
	ModelOverride string
	Timeout       time.Duration
	// KnowledgeForDispatch appends ranked snippets to dispatch bodies when non-nil (same hook as OpenClaw).
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
	HTTPClient           *http.Client
	// PollInterval is the delay between GET /tasks/{id} polls when waiting for completion; default 1s.
	PollInterval time.Duration
}

// Client speaks the MisterMorph daemonruntime task HTTP API.
type Client struct {
	opts Options
}

// New constructs a Client.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = time.Second
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

func joinTasksURL(base string) string {
	s := strings.TrimRight(strings.TrimSpace(base), "/")
	if s == "" {
		return ""
	}
	return s + "/tasks"
}

func joinTaskByIDURL(base, id string) string {
	s := strings.TrimRight(strings.TrimSpace(base), "/")
	if s == "" || strings.TrimSpace(id) == "" {
		return ""
	}
	return s + "/tasks/" + strings.TrimPrefix(strings.TrimSpace(id), "/")
}

type submitTaskRequest struct {
	Task       string `json:"task"`
	Model      string `json:"model,omitempty"`
	Timeout    string `json:"timeout,omitempty"`
	TopicID    string `json:"topic_id,omitempty"`
	TopicTitle string `json:"topic_title,omitempty"`
}

type submitTaskResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type taskInfo struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Result any    `json:"result,omitempty"`
}

func (c *Client) httpClient() *http.Client {
	if c.opts.HTTPClient != nil {
		return c.opts.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) postSubmit(ctx context.Context, req submitTaskRequest) (*submitTaskResponse, error) {
	endpoint := joinTasksURL(c.opts.BaseURL)
	if endpoint == "" {
		return nil, errors.New("mistermorph: base URL is required")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	tok := strings.TrimSpace(c.opts.Token)
	if tok == "" {
		return nil, errors.New("mistermorph: gateway_token (runtime auth) is required")
	}
	httpReq.Header.Set("Authorization", "Bearer "+tok)

	res, err := c.httpClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mistermorph: submit: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("mistermorph: read body: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("mistermorph: submit HTTP %s: %s", res.Status, bytesTrimPreview(raw))
	}
	var out submitTaskResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("mistermorph: invalid submit JSON: %w", err)
	}
	if strings.TrimSpace(out.ID) == "" {
		return nil, errors.New("mistermorph: submit response missing id")
	}
	return &out, nil
}

func (c *Client) getTask(ctx context.Context, id string) (*taskInfo, error) {
	endpoint := joinTaskByIDURL(c.opts.BaseURL, id)
	if endpoint == "" {
		return nil, errors.New("mistermorph: task URL build failed")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	tok := strings.TrimSpace(c.opts.Token)
	httpReq.Header.Set("Authorization", "Bearer "+tok)

	res, err := c.httpClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mistermorph: get task: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("mistermorph: read body: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("mistermorph: get task HTTP %s: %s", res.Status, bytesTrimPreview(raw))
	}
	var info taskInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("mistermorph: invalid task JSON: %w", err)
	}
	return &info, nil
}

func bytesTrimPreview(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 512 {
		return s[:512] + "…"
	}
	return s
}

func terminalMisterMorphStatus(s string) (doneOK bool, failErr error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "queued", "running", "pending":
		return false, nil
	case "done":
		return true, nil
	case "failed", "canceled":
		return false, fmt.Errorf("mistermorph: task %s", strings.TrimSpace(s))
	default:
		return false, fmt.Errorf("mistermorph: unknown task status %q", s)
	}
}

func (c *Client) waitTerminal(ctx context.Context, id string) error {
	nextWait := time.Duration(0)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("mistermorph: %w", ctx.Err())
		case <-time.After(nextWait):
		}
		info, err := c.getTask(ctx, id)
		if err != nil {
			return err
		}
		doneOK, failErr := terminalMisterMorphStatus(info.Status)
		if failErr != nil {
			if errStr := strings.TrimSpace(info.Error); errStr != "" {
				return fmt.Errorf("%w: %s", failErr, errStr)
			}
			return failErr
		}
		if doneOK {
			return nil
		}
		nextWait = c.opts.PollInterval
	}
}

// DispatchTaskWithSession submits the task and polls until done (or failure / context deadline).
// sessionKey, when non-empty, is sent as topic_id (MisterMorph console/runtime topic routing).
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	kb := c.knowledgeMarkdown(callCtx, task.ProductID, openclaw.KnowledgeQueryFromTask(task))
	text := strings.TrimSpace(openclaw.TaskDispatchMarkdown(task, kb))
	if text == "" {
		return "", errors.New("mistermorph: empty dispatch body")
	}

	req := submitTaskRequest{Task: text}
	if m := strings.TrimSpace(c.opts.ModelOverride); m != "" {
		req.Model = m
	}
	if c.opts.Timeout > 0 {
		req.Timeout = c.opts.Timeout.String()
	}
	if tid := strings.TrimSpace(sessionKey); tid != "" {
		req.TopicID = tid
	}

	submitResp, err := c.postSubmit(callCtx, req)
	if err != nil {
		return "", err
	}
	if err := c.waitTerminal(callCtx, submitResp.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("mistermorph:%s", submitResp.ID), nil
}

// DispatchSubtaskWithSession submits a convoy subtask body and waits for terminal status.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	kb := c.knowledgeMarkdown(callCtx, parent.ProductID, openclaw.KnowledgeQueryFromSubtask(parent, sub))
	text := strings.TrimSpace(openclaw.SubtaskDispatchMarkdown(parent.ID, sub, kb))
	if text == "" {
		return "", errors.New("mistermorph: empty subtask dispatch body")
	}

	req := submitTaskRequest{Task: text}
	if m := strings.TrimSpace(c.opts.ModelOverride); m != "" {
		req.Model = m
	}
	if c.opts.Timeout > 0 {
		req.Timeout = c.opts.Timeout.String()
	}
	if tid := strings.TrimSpace(sessionKey); tid != "" {
		req.TopicID = tid
	}

	submitResp, err := c.postSubmit(callCtx, req)
	if err != nil {
		return "", err
	}
	if err := c.waitTerminal(callCtx, submitResp.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("mistermorph:%s:%s", parent.ID, submitResp.ID), nil
}
