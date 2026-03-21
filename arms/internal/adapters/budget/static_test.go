package budget

import (
	"context"
	"testing"

	"github.com/closeloopautomous/arms/internal/adapters/memory"
	"github.com/closeloopautomous/arms/internal/domain"
)

func TestStaticBudget(t *testing.T) {
	ctx := context.Background()
	costs := memory.NewCostStore()
	b := &Static{Cap: 10, Costs: costs}
	pid := domain.ProductID("p1")

	if err := b.AssertWithinBudget(ctx, pid, 9); err != nil {
		t.Fatal(err)
	}
	_ = costs.Append(ctx, domain.CostEvent{ID: "1", ProductID: pid, Amount: 6})
	if err := b.AssertWithinBudget(ctx, pid, 5); err == nil {
		t.Fatal("expected budget exceeded")
	}
	if err := b.AssertWithinBudget(ctx, pid, 4); err != nil {
		t.Fatalf("6+4 within cap: %v", err)
	}
}
