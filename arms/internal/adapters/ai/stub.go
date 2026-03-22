package ai

import (
	"context"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type ResearchStub struct{}

func (ResearchStub) RunResearch(_ context.Context, product domain.Product) (string, error) {
	base := "summary for " + string(product.ID)
	if ctx := ProductContextSnippet(product); ctx != "" {
		return base + "\n---\n" + ctx, nil
	}
	return base, nil
}

var _ ports.ResearchPort = ResearchStub{}

type IdeationStub struct{}

func (IdeationStub) GenerateIdeas(_ context.Context, product domain.Product, researchSummary string) ([]domain.IdeaDraft, error) {
	reasoning := researchSummary + " / " + string(product.ID)
	if ctx := ProductContextSnippet(product); ctx != "" {
		reasoning += "\n---\n" + ctx
	}
	title := "Sample feature"
	if strings.TrimSpace(product.ProgramDocument) != "" {
		title = "Sample feature (program-informed)"
	}
	return []domain.IdeaDraft{
		{
			Title:                title,
			Description:          "Generated from research",
			Impact:               0.8,
			Feasibility:          0.7,
			Reasoning:            reasoning,
			Category:             "feature",
			Complexity:           "M",
			EstimatedEffortHours: 8,
			Tags:                 []string{"stub", "ideation"},
			TechnicalApproach:    "Iterate in planning task after approval",
			Risks:                "Stub data only until real LLM ideation is wired",
			SourceResearch:       "Uses product research_summary + program hints",
		},
	}, nil
}

var _ ports.IdeationPort = IdeationStub{}
