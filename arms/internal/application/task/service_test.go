package task

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/budget"
	gw "github.com/closeloopautomous/arms/internal/adapters/gateway"
	"github.com/closeloopautomous/arms/internal/adapters/identity"
	"github.com/closeloopautomous/arms/internal/adapters/memory"
	"github.com/closeloopautomous/arms/internal/adapters/shipping"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/application/autopilot"
	"github.com/closeloopautomous/arms/internal/application/livefeed"
	"github.com/closeloopautomous/arms/internal/application/product"
	"github.com/closeloopautomous/arms/internal/domain"
)

func TestKanbanDispatchAndCheckpoint(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.Stub{}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	auto := &autopilot.Service{
		Products: products, Ideas: ideas,
		Research: aiStub{}, Ideation: ideationOneIdea{},
		Clock: clock, Identities: ids,
	}
	svc := &Service{
		Tasks: tasks, Products: products, Ideas: ideas,
		Gateway: gateway,
		Budget:  &budget.Static{Cap: 100, Costs: costs},
		Checkpt: checkpoints,
		Clock:   clock,
		IDs:     ids,
		Ship:    shipping.PullRequestNoop{},
	}

	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	_ = auto.RunResearch(ctx, p.ID)
	_ = auto.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	_ = auto.SubmitSwipe(ctx, list[0].ID, domain.DecisionYes)

	tt, err := svc.CreateFromApprovedIdea(ctx, list[0].ID, "spec")
	if err != nil {
		t.Fatal(err)
	}
	if tt.Status != domain.StatusPlanning || tt.PlanApproved {
		t.Fatalf("create: %+v", tt)
	}
	if err := svc.ApprovePlan(ctx, tt.ID, ""); err != nil {
		t.Fatal(err)
	}
	tt, _ = tasks.ByID(ctx, tt.ID)
	if tt.Status != domain.StatusInbox || !tt.PlanApproved {
		t.Fatalf("after approve: %+v", tt)
	}
	if err := svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.Dispatch(ctx, tt.ID, 1); err != nil {
		t.Fatal(err)
	}
	tt, _ = tasks.ByID(ctx, tt.ID)
	if tt.Status != domain.StatusInProgress || tt.ExternalRef == "" {
		t.Fatalf("after dispatch: %+v", tt)
	}
	if err := svc.RecordCheckpoint(ctx, tt.ID, "ckpt-1"); err != nil {
		t.Fatal(err)
	}
	tt, _ = tasks.ByID(ctx, tt.ID)
	if tt.Status != domain.StatusInProgress || tt.Checkpoint != "ckpt-1" {
		t.Fatalf("after checkpoint: %+v", tt)
	}
}

func TestReturnToPlanningFromInboxAndAssigned(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.Stub{}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	auto := &autopilot.Service{
		Products: products, Ideas: ideas,
		Research: aiStub{}, Ideation: ideationOneIdea{},
		Clock: clock, Identities: ids,
	}
	svc := &Service{
		Tasks: tasks, Products: products, Ideas: ideas,
		Gateway: gateway,
		Budget:  &budget.Static{Cap: 100, Costs: costs},
		Checkpt: checkpoints,
		Clock:   clock,
		IDs:     ids,
		Ship:    shipping.PullRequestNoop{},
	}

	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	_ = auto.RunResearch(ctx, p.ID)
	_ = auto.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	_ = auto.SubmitSwipe(ctx, list[0].ID, domain.DecisionYes)

	tt, err := svc.CreateFromApprovedIdea(ctx, list[0].ID, "spec")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ApprovePlan(ctx, tt.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReturnToPlanning(ctx, tt.ID, "needs more detail"); err != nil {
		t.Fatal(err)
	}
	tt, _ = tasks.ByID(ctx, tt.ID)
	if tt.Status != domain.StatusPlanning || tt.PlanApproved || tt.StatusReason != "needs more detail" {
		t.Fatalf("after reject from inbox: %+v", tt)
	}
	if err := svc.ApprovePlan(ctx, tt.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReturnToPlanning(ctx, tt.ID, "wrong assignee"); err != nil {
		t.Fatal(err)
	}
	tt, _ = tasks.ByID(ctx, tt.ID)
	if tt.Status != domain.StatusPlanning || tt.PlanApproved || tt.StatusReason != "wrong assignee" {
		t.Fatalf("after reject from assigned: %+v", tt)
	}
}

func TestReturnToPlanningBlockedAfterDispatch(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.Stub{}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	auto := &autopilot.Service{
		Products: products, Ideas: ideas,
		Research: aiStub{}, Ideation: ideationOneIdea{},
		Clock: clock, Identities: ids,
	}
	svc := &Service{
		Tasks: tasks, Products: products, Ideas: ideas,
		Gateway: gateway,
		Budget:  &budget.Static{Cap: 100, Costs: costs},
		Checkpt: checkpoints,
		Clock:   clock,
		IDs:     ids,
		Ship:    shipping.PullRequestNoop{},
	}

	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	_ = auto.RunResearch(ctx, p.ID)
	_ = auto.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	_ = auto.SubmitSwipe(ctx, list[0].ID, domain.DecisionYes)
	tt, _ := svc.CreateFromApprovedIdea(ctx, list[0].ID, "spec")
	_ = svc.ApprovePlan(ctx, tt.ID, "")
	_ = svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, "")
	if err := svc.Dispatch(ctx, tt.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReturnToPlanning(ctx, tt.ID, "too late"); err == nil {
		t.Fatal("expected error after dispatch")
	}
}

type aiStub struct{}

func (aiStub) RunResearch(context.Context, domain.Product) (string, error) {
	return "r", nil
}

type ideationOneIdea struct{}

func (ideationOneIdea) GenerateIdeas(context.Context, domain.Product, string) ([]domain.IdeaDraft, error) {
	return []domain.IdeaDraft{{Title: "t"}}, nil
}

func TestNudgeStallAndCompleteWithLiveActivityMemory(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.Stub{}
	hub := livefeed.NewHub()
	agentHealth := memory.NewAgentHealthStore()

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	auto := &autopilot.Service{
		Products: products, Ideas: ideas,
		Research: aiStub{}, Ideation: ideationOneIdea{},
		Clock: clock, Identities: ids,
	}
	svc := &Service{
		Tasks: tasks, Products: products, Ideas: ideas,
		Gateway: gateway,
		Budget:  &budget.Static{Cap: 100, Costs: costs},
		Checkpt: checkpoints,
		Clock:   clock,
		IDs:     ids,
		Ship:    shipping.PullRequestNoop{},
		Events:  hub,
		Gate:    NewProductGate(),
		AgentHealth: agentHealth,
	}

	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	_ = auto.RunResearch(ctx, p.ID)
	_ = auto.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	_ = auto.SubmitSwipe(ctx, list[0].ID, domain.DecisionYes)
	tt, err := svc.CreateFromApprovedIdea(ctx, list[0].ID, "spec")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.NudgeStall(ctx, tt.ID, "x"); err == nil {
		t.Fatal("expected nudge to fail in planning")
	}
	_ = svc.ApprovePlan(ctx, tt.ID, "")
	_ = svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, "")
	if err := svc.Dispatch(ctx, tt.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.NudgeStall(ctx, tt.ID, "ping agent"); err != nil {
		t.Fatal(err)
	}
	tt, _ = tasks.ByID(ctx, tt.ID)
	if tt.StatusReason == "" {
		t.Fatalf("want status_reason nudge prefix, got %q", tt.StatusReason)
	}
	h, _ := agentHealth.ByTask(ctx, tt.ID)
	if h == nil {
		t.Fatal("want agent health row after nudge")
	}
	var detail map[string]any
	if err := json.Unmarshal([]byte(h.DetailJSON), &detail); err != nil {
		t.Fatalf("detail json: %v", err)
	}
	if _, ok := detail["stall_nudges"]; !ok {
		t.Fatalf("want stall_nudges in detail, got %q", h.DetailJSON)
	}
	ch, unsub := hub.Subscribe()
	defer unsub()
	if err := svc.CompleteWithLiveActivity(ctx, tt.ID, "unit"); err != nil {
		t.Fatal(err)
	}
	tt, _ = tasks.ByID(ctx, tt.ID)
	if tt.Status != domain.StatusDone {
		t.Fatalf("want done got %s", tt.Status)
	}
	if !svc.UsesLiveActivityTX() {
		select {
		case b := <-ch:
			var ev struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(b, &ev); err != nil {
				t.Fatalf("hub payload: %v", err)
			}
			if ev.Type != "task_completed" {
				t.Fatalf("want task_completed on hub, got %q", ev.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for task_completed on hub")
		}
	}
}
