// Package researchclaw calls a ResearchClaw HTTP API (OpenAPI: /api/health, /api/pipeline/start, /api/pipeline/status, /api/runs/{run_id}).
package researchclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Options tune polling after POST /api/pipeline/start.
type Options struct {
	PollInterval time.Duration
	PollTimeout  time.Duration
	HTTP         *http.Client
}

func normalizeBase(raw string) (string, error) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", fmt.Errorf("researchclaw: base url is empty")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return "", fmt.Errorf("researchclaw: base url must start with http:// or https://")
	}
	u = strings.TrimRight(u, "/")
	return u, nil
}

func joinAPI(base, path string) (string, error) {
	b, err := normalizeBase(base)
	if err != nil {
		return "", err
	}
	p := strings.TrimSpace(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return b + p, nil
}

func authHeader(apiKey string) string {
	t := strings.TrimSpace(apiKey)
	if t == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(t), "bearer ") {
		return t
	}
	return "Bearer " + t
}

// GetJSON performs GET and decodes JSON object into out (map or struct pointer).
func GetJSON(ctx context.Context, hc *http.Client, rawURL, bearer string, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	if h := authHeader(bearer); h != "" {
		req.Header.Set("Authorization", h)
	}
	req.Header.Set("Accept", "application/json")
	res, err := hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return res.StatusCode, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return res.StatusCode, fmt.Errorf("researchclaw: GET %s: %s", rawURL, strings.TrimSpace(string(body)))
	}
	if out != nil && len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return res.StatusCode, fmt.Errorf("researchclaw: decode GET %s: %w", rawURL, err)
		}
	}
	return res.StatusCode, nil
}

// Probe calls GET /api/health and optionally GET /api/version.
func Probe(ctx context.Context, baseURL, apiKey string, hc *http.Client) (health map[string]any, version map[string]any, err error) {
	if hc == nil {
		hc = http.DefaultClient
	}
	u, err := joinAPI(baseURL, "/api/health")
	if err != nil {
		return nil, nil, err
	}
	health = map[string]any{}
	if _, err := GetJSON(ctx, hc, u, apiKey, &health); err != nil {
		return nil, nil, err
	}
	vu, err := joinAPI(baseURL, "/api/version")
	if err != nil {
		return health, nil, nil
	}
	version = map[string]any{}
	if _, err := GetJSON(ctx, hc, vu, apiKey, &version); err != nil {
		return health, nil, nil
	}
	return health, version, nil
}

// RunProductResearch starts the pipeline with a topic derived from the product, polls until idle or run terminal, returns markdown summary.
func RunProductResearch(ctx context.Context, hub *domain.ResearchHub, product domain.Product, opt Options) (string, error) {
	if hub == nil {
		return "", fmt.Errorf("researchclaw: nil hub")
	}
	hc := opt.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	pollEvery := opt.PollInterval
	if pollEvery <= 0 {
		pollEvery = 3 * time.Second
	}
	deadline := opt.PollTimeout
	if deadline <= 0 {
		deadline = 15 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	topic := buildTopic(product)
	startURL, err := joinAPI(hub.BaseURL, "/api/pipeline/start")
	if err != nil {
		return "", err
	}
	body := map[string]any{
		"topic":         topic,
		"auto_approve":  true,
		"config_overrides": nil,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, startURL, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	if h := authHeader(hub.APIKey); h != "" {
		req.Header.Set("Authorization", h)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("researchclaw: pipeline start: %w", err)
	}
	defer res.Body.Close()
	rb, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("researchclaw: POST /api/pipeline/start: %d %s", res.StatusCode, strings.TrimSpace(string(rb)))
	}
	var startResp map[string]any
	if err := json.Unmarshal(rb, &startResp); err != nil {
		return "", fmt.Errorf("researchclaw: decode start response: %w", err)
	}
	runID, _ := startResp["run_id"].(string)
	if strings.TrimSpace(runID) == "" {
		runID, _ = startResp["runId"].(string)
	}
	runID = strings.TrimSpace(runID)

	ticker := time.NewTicker(pollEvery)
	defer ticker.Stop()
	for {
		done, summary, err := pollOnce(ctx, hc, hub, runID)
		if err != nil {
			return "", err
		}
		if done {
			return summary, nil
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("researchclaw: wait pipeline: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func buildTopic(p domain.Product) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s", strings.TrimSpace(p.Name))
	if s := strings.TrimSpace(p.MissionStatement); s != "" {
		fmt.Fprintf(&b, "\n\nMission: %s", s)
	}
	if s := strings.TrimSpace(p.VisionStatement); s != "" {
		fmt.Fprintf(&b, "\n\nVision: %s", s)
	}
	if s := strings.TrimSpace(p.ProgramDocument); s != "" {
		fmt.Fprintf(&b, "\n\nProgram:\n%s", s)
	}
	if s := strings.TrimSpace(p.Description); s != "" {
		fmt.Fprintf(&b, "\n\nDescription: %s", s)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "Research topic"
	}
	return out
}

func pollOnce(ctx context.Context, hc *http.Client, hub *domain.ResearchHub, runID string) (done bool, summary string, err error) {
	statusURL, err := joinAPI(hub.BaseURL, "/api/pipeline/status")
	if err != nil {
		return false, "", err
	}
	st := map[string]any{}
	if _, gerr := GetJSON(ctx, hc, statusURL, hub.APIKey, &st); gerr == nil {
		if pipelineLooksIdle(st) {
			if strings.TrimSpace(runID) != "" {
				if s, ok := fetchRunSummary(ctx, hc, hub, runID); ok {
					return true, s, nil
				}
			}
			return true, formatSummaryFromUnknown(st), nil
		}
	}
	if strings.TrimSpace(runID) != "" {
		run, rerr := getRun(ctx, hc, hub, runID)
		if rerr == nil && runTerminal(run) {
			return true, extractRunMarkdown(run), nil
		}
	}
	return false, "", nil
}

func getRun(ctx context.Context, hc *http.Client, hub *domain.ResearchHub, runID string) (map[string]any, error) {
	escaped := url.PathEscape(runID)
	u, err := joinAPI(hub.BaseURL, "/api/runs/"+escaped)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if _, err := GetJSON(ctx, hc, u, hub.APIKey, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func fetchRunSummary(ctx context.Context, hc *http.Client, hub *domain.ResearchHub, runID string) (string, bool) {
	run, err := getRun(ctx, hc, hub, runID)
	if err != nil || len(run) == 0 {
		return "", false
	}
	return extractRunMarkdown(run), true
}

func pipelineLooksIdle(st map[string]any) bool {
	if st == nil {
		return false
	}
	if v, ok := st["pipeline_running"].(bool); ok && !v {
		return true
	}
	if v, ok := st["running"].(bool); ok && !v {
		return true
	}
	if v, ok := st["active"].(bool); ok && !v {
		return true
	}
	if s, ok := st["status"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "idle", "completed", "stopped", "done", "ok":
			return true
		}
	}
	return false
}

func runTerminal(run map[string]any) bool {
	if s, ok := run["status"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "completed", "complete", "finished", "failed", "error", "stopped", "cancelled", "canceled":
			return true
		}
	}
	if s, ok := run["state"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "completed", "complete", "finished", "failed", "error", "stopped":
			return true
		}
	}
	return false
}

func extractRunMarkdown(run map[string]any) string {
	for _, k := range []string{"summary", "research_summary", "final_report", "markdown", "report", "output", "text"} {
		if v, ok := run[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	// Nested common keys
	if art, ok := run["artifacts"].(map[string]any); ok {
		s := extractRunMarkdown(art)
		if s != "" {
			return s
		}
	}
	return formatSummaryFromUnknown(run)
}

func formatSummaryFromUnknown(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("```\n%v\n```", v)
	}
	return "## ResearchClaw response\n\n```json\n" + string(b) + "\n```\n"
}
