package budget

import (
	"context"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Static enforces a single cap per product using recorded costs.
type Static struct {
	Cap   float64
	Costs ports.CostRepository
}

func (b *Static) AssertWithinBudget(ctx context.Context, productID domain.ProductID, additional float64) error {
	sum, err := b.Costs.SumByProduct(ctx, productID)
	if err != nil {
		return err
	}
	if sum+additional > b.Cap {
		return domain.ErrBudgetExceeded
	}
	return nil
}

var _ ports.BudgetPolicy = (*Static)(nil)
