package budget

import (
	"context"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Composite enforces per-product caps from [ports.CostCapRepository] plus a default cumulative ceiling when no row or null cumulative_cap.
type Composite struct {
	Costs             ports.CostRepository
	Caps              ports.CostCapRepository
	Clock             ports.Clock
	DefaultCumulative float64
}

var _ ports.BudgetPolicy = (*Composite)(nil)

func (b *Composite) AssertWithinBudget(ctx context.Context, productID domain.ProductID, additional float64) error {
	caps, err := b.Caps.Get(ctx, productID)
	if err != nil && err != domain.ErrNotFound {
		return err
	}
	now := b.Clock.Now().UTC()
	if caps != nil {
		if caps.DailyCap != nil {
			start := startOfDayUTC(now)
			sum, err := b.Costs.SumByProductSince(ctx, productID, start)
			if err != nil {
				return err
			}
			if sum+additional > *caps.DailyCap {
				return domain.ErrBudgetExceeded
			}
		}
		if caps.MonthlyCap != nil {
			start := startOfMonthUTC(now)
			sum, err := b.Costs.SumByProductSince(ctx, productID, start)
			if err != nil {
				return err
			}
			if sum+additional > *caps.MonthlyCap {
				return domain.ErrBudgetExceeded
			}
		}
		if caps.CumulativeCap != nil {
			sum, err := b.Costs.SumByProduct(ctx, productID)
			if err != nil {
				return err
			}
			if sum+additional > *caps.CumulativeCap {
				return domain.ErrBudgetExceeded
			}
		}
		return nil
	}
	if b.DefaultCumulative > 0 {
		sum, err := b.Costs.SumByProduct(ctx, productID)
		if err != nil {
			return err
		}
		if sum+additional > b.DefaultCumulative {
			return domain.ErrBudgetExceeded
		}
	}
	return nil
}

func startOfDayUTC(t time.Time) time.Time {
	t = t.UTC()
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func startOfMonthUTC(t time.Time) time.Time {
	t = t.UTC()
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
}
