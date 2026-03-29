package ports

import (
	"context"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// ResearchHubRegistry persists ResearchClaw HTTP endpoint profiles (base URL + optional API key).
type ResearchHubRegistry interface {
	List(ctx context.Context, limit int) ([]domain.ResearchHub, error)
	ByID(ctx context.Context, id string) (*domain.ResearchHub, error)
	Save(ctx context.Context, h *domain.ResearchHub) error
	Update(ctx context.Context, h *domain.ResearchHub) error
	Delete(ctx context.Context, id string) error
}

// ResearchSystemSettingsRepository stores global research routing preferences (singleton row).
type ResearchSystemSettingsRepository interface {
	Get(ctx context.Context) (domain.ResearchSystemSettings, error)
	Upsert(ctx context.Context, s domain.ResearchSystemSettings) error
}
