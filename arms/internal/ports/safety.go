package ports

import (
	"context"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

// BudgetPolicy enforces cost caps before dispatching work.
type BudgetPolicy interface {
	AssertWithinBudget(ctx context.Context, productID domain.ProductID, additional float64) error
}

// Clock abstracts time for tests.
type Clock interface {
	Now() time.Time
}
