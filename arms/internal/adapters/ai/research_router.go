package ai

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/researchclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// ResearchRouter sends RunResearch to ResearchClaw when system settings enable it and a default hub is set; otherwise uses Fallback.
type ResearchRouter struct {
	Settings ports.ResearchSystemSettingsRepository
	Hubs     ports.ResearchHubRegistry
	Fallback ports.ResearchPort
	HTTP     *http.Client
	// PollInterval / PollTimeout are applied to ResearchClaw pipeline polling (defaults: 3s / 15m).
	PollInterval time.Duration
	PollTimeout  time.Duration
}

// RunResearch implements ports.ResearchPort.
func (r *ResearchRouter) RunResearch(ctx context.Context, product domain.Product) (string, error) {
	if r == nil || r.Fallback == nil {
		return "", fmt.Errorf("research router: not configured")
	}
	if r.Settings == nil || r.Hubs == nil {
		return r.Fallback.RunResearch(ctx, product)
	}
	st, err := r.Settings.Get(ctx)
	if err != nil {
		return "", err
	}
	if !st.AutoResearchClawEnabled {
		return r.Fallback.RunResearch(ctx, product)
	}
	hid := strings.TrimSpace(st.DefaultResearchHubID)
	if hid == "" {
		slog.Warn("arms research", "msg", "auto_research_claw_enabled but no default_research_hub_id; using LLM/stub fallback")
		return r.Fallback.RunResearch(ctx, product)
	}
	hub, err := r.Hubs.ByID(ctx, hid)
	if err != nil {
		return "", fmt.Errorf("research hub %q: %w", hid, err)
	}
	if strings.TrimSpace(hub.BaseURL) == "" {
		return "", fmt.Errorf("research hub %q has empty base_url", hid)
	}
	hc := r.HTTP
	if hc == nil {
		hc = DefaultHTTPClient()
	}
	summary, err := researchclaw.RunProductResearch(ctx, hub, product, researchclaw.Options{
		HTTP:         hc,
		PollInterval: r.PollInterval,
		PollTimeout:  r.PollTimeout,
	})
	if err != nil {
		return "", err
	}
	return summary, nil
}

var _ ports.ResearchPort = (*ResearchRouter)(nil)
