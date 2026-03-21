package cost

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/budget"
	"github.com/closeloopautomous/arms/internal/adapters/memory"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/domain"
)

func TestRecordEnforcesBudget(t *testing.T) {
	ctx := context.Background()
	costs := memory.NewCostStore()
	caps := memory.NewCostCapStore()
	clock := timeadapter.Fixed{T: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)}
	d := 10.0
	_ = caps.Upsert(ctx, &domain.ProductCostCaps{ProductID: "p1", DailyCap: &d})
	budgetPolicy := &budget.Composite{
		Costs:             costs,
		Caps:              caps,
		Clock:             clock,
		DefaultCumulative: 1000,
	}
	s := &Service{
		Costs:  costs,
		Caps:   caps,
		Budget: budgetPolicy,
		Clock:  clock,
		IDs:    &stubIDs{},
	}
	pid := domain.ProductID("p1")
	tid := domain.TaskID("t1")
	if err := s.Record(ctx, pid, tid, 6, "", "", ""); err != nil {
		t.Fatal(err)
	}
	err := s.Record(ctx, pid, tid, 5, "", "", "")
	if err == nil {
		t.Fatal("expected budget exceeded on second record")
	}
	if !errors.Is(err, domain.ErrBudgetExceeded) {
		t.Fatalf("want ErrBudgetExceeded, got %v", err)
	}
}

type stubIDs struct{ n int }

func (s *stubIDs) NewProductID() domain.ProductID { s.n++; return domain.ProductID("p") }
func (s *stubIDs) NewIdeaID() domain.IdeaID       { s.n++; return domain.IdeaID("i") }
func (s *stubIDs) NewTaskID() domain.TaskID       { s.n++; return domain.TaskID("t") }
func (s *stubIDs) NewConvoyID() domain.ConvoyID   { s.n++; return domain.ConvoyID("c") }
func (s *stubIDs) NewSubtaskID() domain.SubtaskID { s.n++; return domain.SubtaskID("st") }
func (s *stubIDs) NewCostEventID() string         { s.n++; return "ce" }
