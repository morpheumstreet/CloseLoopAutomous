package ports

import (
	"context"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// RequireActiveProduct returns nil when the product exists and is not soft-deleted.
// Otherwise it returns the same error as [ProductRepository.ByID] (typically [domain.ErrNotFound]).
// Use this from application services to cascade soft-delete semantics without duplicating ByID checks.
func RequireActiveProduct(ctx context.Context, r ProductRepository, id domain.ProductID) error {
	_, err := r.ByID(ctx, id)
	return err
}
