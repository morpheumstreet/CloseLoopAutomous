package task

import (
	"context"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/budget"
	gw "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/identity"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/memory"
	timeadapter "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/time"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func testTaskService(t *testing.T, clock timeadapter.Fixed) (*Service, *memory.TaskStore, *memory.ProductStore) {
	t.Helper()
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	svc := &Service{
		Tasks: tasks, Products: products, Ideas: memory.NewIdeaStore(),
		Gateway: &gw.SimulationMockClaw{}, Budget: &budget.Static{Cap: 100, Costs: memory.NewCostStore()},
		Checkpt: memory.NewCheckpointStore(), Clock: clock, IDs: &identity.Sequential{},
	}
	return svc, tasks, products
}

func TestApplyCIWebhookOutcomeTestingToReview(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	svc, tasks, products := testTaskService(t, clock)
	now := clock.Now()
	_ = products.Save(ctx, &domain.Product{
		ID: "p1", Name: "n", Stage: domain.StageResearch, WorkspaceID: "w",
		UpdatedAt: now, AutomationTier: domain.TierFullAuto,
	})
	_ = tasks.Save(ctx, &domain.Task{
		ID: "t1", ProductID: "p1", IdeaID: "i1", Spec: "s",
		Status: domain.StatusTesting, PlanApproved: true, CreatedAt: now, UpdatedAt: now,
	})
	if err := svc.ApplyCIWebhookOutcome(ctx, "t1", "review", "", "ci_test"); err != nil {
		t.Fatal(err)
	}
	tt, err := tasks.ByID(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if tt.Status != domain.StatusReview {
		t.Fatalf("want review got %s", tt.Status)
	}
}

func TestApplyCIWebhookOutcomeFailed(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	svc, tasks, products := testTaskService(t, clock)
	now := clock.Now()
	_ = products.Save(ctx, &domain.Product{
		ID: "p1", Name: "n", Stage: domain.StageResearch, WorkspaceID: "w",
		UpdatedAt: now, AutomationTier: domain.TierSupervised,
	})
	_ = tasks.Save(ctx, &domain.Task{
		ID: "t1", ProductID: "p1", IdeaID: "i1", Spec: "s",
		Status: domain.StatusTesting, PlanApproved: true, CreatedAt: now, UpdatedAt: now,
	})
	if err := svc.ApplyCIWebhookOutcome(ctx, "t1", "failed", "job xyz", "ci_test"); err != nil {
		t.Fatal(err)
	}
	tt, err := tasks.ByID(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if tt.Status != domain.StatusFailed {
		t.Fatalf("want failed got %s", tt.Status)
	}
	if tt.StatusReason != "job xyz" {
		t.Fatalf("reason %q", tt.StatusReason)
	}
}

func TestApplyCIWebhookOutcomeDone(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	svc, tasks, products := testTaskService(t, clock)
	now := clock.Now()
	_ = products.Save(ctx, &domain.Product{
		ID: "p1", Name: "n", Stage: domain.StageResearch, WorkspaceID: "w",
		UpdatedAt: now, AutomationTier: domain.TierSemiAuto,
	})
	_ = tasks.Save(ctx, &domain.Task{
		ID: "t1", ProductID: "p1", IdeaID: "i1", Spec: "s",
		Status: domain.StatusReview, PlanApproved: true, CreatedAt: now, UpdatedAt: now,
	})
	if err := svc.ApplyCIWebhookOutcome(ctx, "t1", "done", "", "ci_test"); err != nil {
		t.Fatal(err)
	}
	tt, err := tasks.ByID(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if tt.Status != domain.StatusDone {
		t.Fatalf("want done got %s", tt.Status)
	}
}

func TestApplyCIWebhookOutcomeInvalidTarget(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	svc, _, _ := testTaskService(t, clock)
	err := svc.ApplyCIWebhookOutcome(ctx, "t1", "in_progress", "", "ci_test")
	if err == nil {
		t.Fatal("want error")
	}
}

func TestApplyCIWebhookOutcomeMissingNext(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	svc, _, _ := testTaskService(t, clock)
	err := svc.ApplyCIWebhookOutcome(ctx, "t1", "  ", "", "ci_test")
	if err == nil {
		t.Fatal("want error")
	}
}
