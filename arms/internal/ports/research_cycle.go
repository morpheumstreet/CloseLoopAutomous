package ports

import (
	"context"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

// ResearchCycleRepository stores append-only research run history per product.
type ResearchCycleRepository interface {
	Append(ctx context.Context, id string, productID domain.ProductID, summary string, at time.Time) error
	ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.ResearchCycle, error)
}
