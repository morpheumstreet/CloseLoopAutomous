package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// IdeationLLM implements ports.IdeationPort via an OpenAI-compatible chat API.
type IdeationLLM struct {
	Client  *ChatClient
	Model   string
	Timeout time.Duration
}

const ideationSystemPrompt = `You are a senior product manager generating feature candidates for a software product.
You must respond with a single JSON value only: a JSON array of idea objects. No markdown fences, no commentary.

Each idea object must use these keys (use JSON null only when truly unknown; prefer strings and numbers):
- title (string, required)
- description (string, required)
- impact (number 0–1)
- feasibility (number 0–1)
- reasoning (string)
- category (string: one of feature, improvement, ux, bugfix, research, infra, other)
- research_backing (string: how prior research supports this)
- complexity (string: S, M, L, or XL)
- estimated_effort_hours (number)
- competitive_analysis (string)
- target_user_segment (string)
- revenue_potential (string)
- technical_approach (string)
- risks (string)
- tags (array of strings)
- source_research (string: short quote or paraphrase from the research input)

Produce between 3 and 8 distinct ideas unless the research input is extremely thin — then 1–2 is acceptable.`

type ideaDraftJSON struct {
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	Impact               float64  `json:"impact"`
	Feasibility          float64  `json:"feasibility"`
	Reasoning            string   `json:"reasoning"`
	Category             string   `json:"category"`
	ResearchBacking      string   `json:"research_backing"`
	Complexity           string   `json:"complexity"`
	EstimatedEffortHours float64  `json:"estimated_effort_hours"`
	CompetitiveAnalysis  string   `json:"competitive_analysis"`
	TargetUserSegment    string   `json:"target_user_segment"`
	RevenuePotential     string   `json:"revenue_potential"`
	TechnicalApproach    string   `json:"technical_approach"`
	Risks                string   `json:"risks"`
	Tags                 []string `json:"tags"`
	SourceResearch       string   `json:"source_research"`
}

// GenerateIdeas calls the LLM with product context and the research summary, then parses JSON drafts.
func (i *IdeationLLM) GenerateIdeas(ctx context.Context, product domain.Product, researchSummary string) ([]domain.IdeaDraft, error) {
	if i == nil || i.Client == nil {
		return nil, fmt.Errorf("ideation llm: nil client")
	}
	if strings.TrimSpace(i.Model) == "" {
		return nil, fmt.Errorf("ideation llm: model not configured")
	}
	timeout := i.Timeout
	if timeout <= 0 {
		timeout = 180 * time.Second
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
	rs := strings.TrimSpace(researchSummary)
	if rs == "" {
		rs = "(No prior research summary stored — infer carefully from product context only.)"
	}
	fmt.Fprintf(&ub, "\n## Research summary to build on\n%s\n", rs)

	raw, err := i.Client.ChatCompletion(cctx, i.Model, ideationSystemPrompt, ub.String(), 0.55, 8192)
	if err != nil {
		return nil, err
	}
	return parseIdeationJSON(raw)
}

func parseIdeationJSON(raw string) ([]domain.IdeaDraft, error) {
	s := stripMarkdownCodeFence(strings.TrimSpace(raw))
	var rows []ideaDraftJSON
	if err := json.Unmarshal([]byte(s), &rows); err != nil {
		return nil, fmt.Errorf("ideation llm: parse json array: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("ideation llm: model returned zero ideas")
	}
	out := make([]domain.IdeaDraft, 0, len(rows))
	for _, r := range rows {
		title := strings.TrimSpace(r.Title)
		desc := strings.TrimSpace(r.Description)
		if title == "" && desc == "" {
			continue
		}
		if title == "" {
			title = firstLine(desc)
		}
		out = append(out, domain.IdeaDraft{
			Title:                title,
			Description:          desc,
			Impact:               r.Impact,
			Feasibility:          r.Feasibility,
			Reasoning:            strings.TrimSpace(r.Reasoning),
			Category:             strings.TrimSpace(r.Category),
			ResearchBacking:      strings.TrimSpace(r.ResearchBacking),
			Complexity:           strings.TrimSpace(r.Complexity),
			EstimatedEffortHours: r.EstimatedEffortHours,
			CompetitiveAnalysis:  strings.TrimSpace(r.CompetitiveAnalysis),
			TargetUserSegment:    strings.TrimSpace(r.TargetUserSegment),
			RevenuePotential:     strings.TrimSpace(r.RevenuePotential),
			TechnicalApproach:    strings.TrimSpace(r.TechnicalApproach),
			Risks:                strings.TrimSpace(r.Risks),
			Tags:                 append([]string(nil), r.Tags...),
			SourceResearch:       strings.TrimSpace(r.SourceResearch),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ideation llm: no usable ideas after parsing")
	}
	return out, nil
}

func stripMarkdownCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToLower(s), "json") {
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = strings.TrimSpace(s[idx+1:])
		}
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}

var _ ports.IdeationPort = (*IdeationLLM)(nil)
