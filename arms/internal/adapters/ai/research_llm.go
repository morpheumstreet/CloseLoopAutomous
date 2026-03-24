package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// ResearchLLM implements ports.ResearchPort via an OpenAI-compatible chat API.
type ResearchLLM struct {
	Client  *ChatClient
	Model   string
	Timeout time.Duration
}

const researchSystemPrompt = `You are a product research analyst for a software team. Produce a concise, actionable research brief in Markdown.
Focus on market/context, user problems, constraints from the materials provided, and open questions — not implementation tasks.
Stay factual; label speculation clearly. Aim for roughly 400–1200 words unless the product context is very thin.`

// RunResearch gathers context from the product record and returns a markdown summary.
func (r *ResearchLLM) RunResearch(ctx context.Context, product domain.Product) (string, error) {
	if r == nil || r.Client == nil {
		return "", fmt.Errorf("research llm: nil client")
	}
	if strings.TrimSpace(r.Model) == "" {
		return "", fmt.Errorf("research llm: model not configured")
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var ub strings.Builder
	fmt.Fprintf(&ub, "Product name: %s\n", strings.TrimSpace(product.Name))
	if s := strings.TrimSpace(product.MissionStatement); s != "" {
		fmt.Fprintf(&ub, "Mission: %s\n", s)
	}
	if s := strings.TrimSpace(product.VisionStatement); s != "" {
		fmt.Fprintf(&ub, "Vision: %s\n", s)
	}
	if ctx := ProductContextSnippet(product); ctx != "" {
		fmt.Fprintf(&ub, "\n## Product context\n%s\n", ctx)
	}
	if s := strings.TrimSpace(product.SettingsJSON); s != "" {
		fmt.Fprintf(&ub, "\n## Settings JSON (opaque, may contain hints)\n%s\n", s)
	}

	user := ub.String()
	if strings.TrimSpace(user) == "" {
		user = "No product context was provided. Produce a generic research template and questions the team should answer."
	}

	return r.Client.ChatCompletion(cctx, r.Model, researchSystemPrompt, user, 0.35, 4096)
}

var _ ports.ResearchPort = (*ResearchLLM)(nil)
