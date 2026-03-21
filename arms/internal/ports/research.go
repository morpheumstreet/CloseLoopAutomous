package ports

import (
	"context"

	"github.com/closeloopautomous/arms/internal/domain"
)

// ResearchPort and IdeationPort model external AI calls kept behind adapters
// so the application core stays free of transport details.

// ResearchPort receives the full domain.Product so implementations can use
// Description, ProgramDocument, RepoURL/RepoBranch, SettingsJSON, etc. in prompts.
type ResearchPort interface {
	RunResearch(ctx context.Context, product domain.Product) (summary string, err error)
}

// IdeationPort receives the product and the stored research summary; implementations
// should treat ProgramDocument and related fields as first-class context alongside researchSummary.
type IdeationPort interface {
	GenerateIdeas(ctx context.Context, product domain.Product, researchSummary string) ([]domain.IdeaDraft, error)
}
