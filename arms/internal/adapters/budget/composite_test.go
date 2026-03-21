package budget

import (
	"context"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/memory"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/domain"
)

func TestCompositeDefaultCumulative(t *testing.T) {
	ctx := context.Background()
	costs := memory.NewCostStore()
	caps := memory.NewCostCapStore()
	clock := timeadapter.Fixed{T: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)}
	b := &Composite{Costs: costs, Caps: caps, Clock: clock, DefaultCumulative: 50}
	pid := domain.ProductID("p1")
	_ = costs.Append(ctx, domain.CostEvent{ID: "1", ProductID: pid, TaskID: "t1", Amount: 40, At: clock.T})
	if err := b.AssertWithinBudget(ctx, pid, 15); err == nil {
		t.Fatal("expected budget exceeded")
	}
	if err := b.AssertWithinBudget(ctx, pid, 10); err != nil {
		t.Fatal(err)
	}
}

func TestCompositeDailyCap(t *testing.T) {
	ctx := context.Background()
	costs := memory.NewCostStore()
	caps := memory.NewCostCapStore()
	clock := timeadapter.Fixed{T: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)}
	d := 25.0
	_ = caps.Upsert(ctx, &domain.ProductCostCaps{ProductID: "p1", DailyCap: &d})
	b := &Composite{Costs: costs, Caps: caps, Clock: clock, DefaultCumulative: 1000}
	pid := domain.ProductID("p1")
	_ = costs.Append(ctx, domain.CostEvent{ID: "1", ProductID: pid, TaskID: "t1", Amount: 20, At: clock.T})
	if err := b.AssertWithinBudget(ctx, pid, 10); err == nil {
		t.Fatal("expected daily cap exceeded")
	}
	if err := b.AssertWithinBudget(ctx, pid, 5); err != nil {
		t.Fatal(err)
	}
}
