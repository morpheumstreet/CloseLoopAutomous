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
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/product"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestApplyAgentWebhookOutcomeAdvanceToTesting(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	gateway := &gw.SimulationMockClaw{}
	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, err := prodSvc.Register(ctx, product.RegistrationInput{
		Name: "p", WorkspaceID: "w", AutomationTier: "full_auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	tt := &domain.Task{
		ID: "t1", ProductID: p.ID, IdeaID: "i1", Spec: "s", Status: domain.StatusInProgress,
		PlanApproved: true, ExternalRef: "ref", CreatedAt: clock.Now(), UpdatedAt: clock.Now(),
	}
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}
	svc := &Service{
		Tasks: tasks, Products: products, Ideas: memory.NewIdeaStore(),
		Gateway: gateway, Budget: &budget.Static{Cap: 100, Costs: memory.NewCostStore()},
		Checkpt: memory.NewCheckpointStore(), Clock: clock, IDs: ids, Ship: &fakePRPublisher{},
		Gate: NewProductGate(),
	}
	if err := svc.ApplyAgentWebhookOutcome(ctx, tt.ID, "testing", "webhook"); err != nil {
		t.Fatal(err)
	}
	tt2, _ := tasks.ByID(ctx, tt.ID)
	if tt2.Status != domain.StatusTesting {
		t.Fatalf("want testing got %s", tt2.Status)
	}
}

func TestApplyAgentWebhookOutcomeSupervisedFallsBackToDone(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	gateway := &gw.SimulationMockClaw{}
	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, err := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	if err != nil {
		t.Fatal(err)
	}
	tt := &domain.Task{
		ID: "t1", ProductID: p.ID, IdeaID: "i1", Spec: "s", Status: domain.StatusInProgress,
		PlanApproved: true, CreatedAt: clock.Now(), UpdatedAt: clock.Now(),
	}
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}
	svc := &Service{
		Tasks: tasks, Products: products, Ideas: memory.NewIdeaStore(),
		Gateway: gateway, Budget: &budget.Static{Cap: 100, Costs: memory.NewCostStore()},
		Checkpt: memory.NewCheckpointStore(), Clock: clock, IDs: ids, Gate: NewProductGate(),
	}
	if err := svc.ApplyAgentWebhookOutcome(ctx, tt.ID, "testing", "webhook"); err != nil {
		t.Fatal(err)
	}
	tt2, _ := tasks.ByID(ctx, tt.ID)
	if tt2.Status != domain.StatusDone {
		t.Fatalf("supervised should complete, got %s", tt2.Status)
	}
}
