package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	"github.com/closeloopautomous/arms/internal/ports"
)

type fakePRPublisher struct{ creates int }

func (f *fakePRPublisher) CreatePullRequest(_ context.Context, in ports.CreatePullRequestInput) (ports.CreatePullRequestResult, error) {
	f.creates++
	return ports.CreatePullRequestResult{HTMLURL: "https://github.com/acme/demo/pull/99", Number: 99}, nil
}

type mergeShipRecorder struct {
	completeCalls            int
	completeIfPolicyCalls    int
	lastTaskOnComplete       domain.TaskID
	tasks                    ports.TaskRepository
	wantPRNumberBeforeMerge int
}

func (m *mergeShipRecorder) Complete(ctx context.Context, taskID domain.TaskID, _ bool) error {
	m.completeCalls++
	m.lastTaskOnComplete = taskID
	if m.tasks != nil && m.wantPRNumberBeforeMerge > 0 {
		t, err := m.tasks.ByID(ctx, taskID)
		if err != nil {
			return err
		}
		if t.PullRequestNumber != m.wantPRNumberBeforeMerge {
			return fmt.Errorf("merge wanted pr#%d got %d", m.wantPRNumberBeforeMerge, t.PullRequestNumber)
		}
	}
	return nil
}

func (m *mergeShipRecorder) CompleteIfPolicyAllowsAuto(ctx context.Context, taskID domain.TaskID) error {
	m.completeIfPolicyCalls++
	return m.Complete(ctx, taskID, false)
}

type flakyPRPublisher struct {
	failRemaining int
	calls         int
}

func (f *flakyPRPublisher) CreatePullRequest(_ context.Context, _ ports.CreatePullRequestInput) (ports.CreatePullRequestResult, error) {
	f.calls++
	if f.failRemaining > 0 {
		f.failRemaining--
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: transient failure", domain.ErrShipping)
	}
	return ports.CreatePullRequestResult{HTMLURL: "https://github.com/acme/demo/pull/7", Number: 7}, nil
}

type unauthorizedPRPublisher struct{ calls int }

func (u *unauthorizedPRPublisher) CreatePullRequest(_ context.Context, _ ports.CreatePullRequestInput) (ports.CreatePullRequestResult, error) {
	u.calls++
	return ports.CreatePullRequestResult{}, errors.Join(
		domain.ErrShippingNonRetryable,
		fmt.Errorf("%w: unauthorized (check token scopes: repo)", domain.ErrShipping),
	)
}

func TestKanbanDispatchAndCheckpoint(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}

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
	ideaAfter, _ := ideas.ByID(ctx, list[0].ID)
	if ideaAfter.Status != domain.IdeaStatusBuilding || ideaAfter.LinkedTaskID != tt.ID {
		t.Fatalf("idea workflow after task create: status=%q task_id=%q want building + %q", ideaAfter.Status, ideaAfter.LinkedTaskID, tt.ID)
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
	gateway := &gw.SimulationMockClaw{}

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
	gateway := &gw.SimulationMockClaw{}

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
	gateway := &gw.SimulationMockClaw{}
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

func TestPostExecFullAutoOpensPROnCompleteWhenHeadSet(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}
	prPub := &fakePRPublisher{}
	mergeRec := &mergeShipRecorder{tasks: tasks, wantPRNumberBeforeMerge: 99}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, err := prodSvc.Register(ctx, product.RegistrationInput{
		Name:           "p",
		WorkspaceID:    "w",
		RepoURL:        "https://github.com/acme/demo",
		AutomationTier: "full_auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	auto := &autopilot.Service{
		Products: products, Ideas: ideas,
		Research: aiStub{}, Ideation: ideationOneIdea{},
		Clock: clock, Identities: ids,
	}
	svc := &Service{
		Tasks:     tasks, Products: products, Ideas: ideas,
		Gateway:   gateway,
		Budget:    &budget.Static{Cap: 100, Costs: costs},
		Checkpt:   checkpoints,
		Clock:     clock,
		IDs:       ids,
		Ship:      prPub,
		MergeShip: mergeRec,
	}
	_ = auto.RunResearch(ctx, p.ID)
	_ = auto.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	_ = auto.SubmitSwipe(ctx, list[0].ID, domain.DecisionYes)
	tt, err := svc.CreateFromApprovedIdea(ctx, list[0].ID, "spec")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.ApprovePlan(ctx, tt.ID, "")
	_ = svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, "")
	_ = svc.Dispatch(ctx, tt.ID, 1)
	tt, _ = tasks.ByID(ctx, tt.ID)
	tt.PullRequestHeadBranch = "feat/autopr"
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}
	if err := svc.Complete(ctx, tt.ID); err != nil {
		t.Fatal(err)
	}
	if prPub.creates != 1 {
		t.Fatalf("want 1 PR create, got %d", prPub.creates)
	}
	if mergeRec.completeCalls != 1 {
		t.Fatalf("want 1 merge complete, got %d", mergeRec.completeCalls)
	}
	tt, _ = tasks.ByID(ctx, tt.ID)
	if tt.PullRequestNumber != 99 || tt.Status != domain.StatusDone {
		t.Fatalf("task after complete: %+v", tt)
	}
}

func TestPostExecSemiAutoOpensPRBeforePolicyMerge(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}
	prPub := &fakePRPublisher{}
	mergeRec := &mergeShipRecorder{tasks: tasks, wantPRNumberBeforeMerge: 99}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, err := prodSvc.Register(ctx, product.RegistrationInput{
		Name:           "p",
		WorkspaceID:    "w",
		RepoURL:        "https://github.com/acme/demo",
		AutomationTier: "semi_auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	auto := &autopilot.Service{
		Products: products, Ideas: ideas,
		Research: aiStub{}, Ideation: ideationOneIdea{},
		Clock: clock, Identities: ids,
	}
	svc := &Service{
		Tasks:     tasks, Products: products, Ideas: ideas,
		Gateway:   gateway,
		Budget:    &budget.Static{Cap: 100, Costs: costs},
		Checkpt:   checkpoints,
		Clock:     clock,
		IDs:       ids,
		Ship:      prPub,
		MergeShip: mergeRec,
	}
	_ = auto.RunResearch(ctx, p.ID)
	_ = auto.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	_ = auto.SubmitSwipe(ctx, list[0].ID, domain.DecisionYes)
	tt, err := svc.CreateFromApprovedIdea(ctx, list[0].ID, "spec")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.ApprovePlan(ctx, tt.ID, "")
	_ = svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, "")
	_ = svc.Dispatch(ctx, tt.ID, 1)
	tt, _ = tasks.ByID(ctx, tt.ID)
	tt.PullRequestHeadBranch = "feat/semi"
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}
	if err := svc.Complete(ctx, tt.ID); err != nil {
		t.Fatal(err)
	}
	if prPub.creates != 1 {
		t.Fatalf("want 1 PR create, got %d", prPub.creates)
	}
	if mergeRec.completeIfPolicyCalls != 1 {
		t.Fatalf("want 1 CompleteIfPolicyAllowsAuto, got %d", mergeRec.completeIfPolicyCalls)
	}
}

func TestPostExecSupervisedDoesNotAutoMerge(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}
	prPub := &fakePRPublisher{}
	mergeRec := &mergeShipRecorder{}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, err := prodSvc.Register(ctx, product.RegistrationInput{
		Name:           "p",
		WorkspaceID:    "w",
		RepoURL:        "https://github.com/acme/demo",
		AutomationTier: "supervised",
	})
	if err != nil {
		t.Fatal(err)
	}
	auto := &autopilot.Service{
		Products: products, Ideas: ideas,
		Research: aiStub{}, Ideation: ideationOneIdea{},
		Clock: clock, Identities: ids,
	}
	svc := &Service{
		Tasks:     tasks, Products: products, Ideas: ideas,
		Gateway:   gateway,
		Budget:    &budget.Static{Cap: 100, Costs: costs},
		Checkpt:   checkpoints,
		Clock:     clock,
		IDs:       ids,
		Ship:      prPub,
		MergeShip: mergeRec,
	}
	_ = auto.RunResearch(ctx, p.ID)
	_ = auto.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	_ = auto.SubmitSwipe(ctx, list[0].ID, domain.DecisionYes)
	tt, err := svc.CreateFromApprovedIdea(ctx, list[0].ID, "spec")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.ApprovePlan(ctx, tt.ID, "")
	_ = svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, "")
	_ = svc.Dispatch(ctx, tt.ID, 1)
	tt, _ = tasks.ByID(ctx, tt.ID)
	tt.PullRequestHeadBranch = "feat/sup"
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}
	if err := svc.Complete(ctx, tt.ID); err != nil {
		t.Fatal(err)
	}
	if mergeRec.completeCalls != 0 || mergeRec.completeIfPolicyCalls != 0 {
		t.Fatalf("supervised should not invoke merge ship, got complete=%d policy=%d",
			mergeRec.completeCalls, mergeRec.completeIfPolicyCalls)
	}
	if prPub.creates != 0 {
		t.Fatalf("supervised should not auto-open PR on complete, creates=%d", prPub.creates)
	}
}

func TestPostExecSkipsPROpenWhenURLAlreadySet(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}
	prPub := &fakePRPublisher{}
	mergeRec := &mergeShipRecorder{tasks: tasks, wantPRNumberBeforeMerge: 42}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, err := prodSvc.Register(ctx, product.RegistrationInput{
		Name:           "p",
		WorkspaceID:    "w",
		RepoURL:        "https://github.com/acme/demo",
		AutomationTier: "full_auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	auto := &autopilot.Service{
		Products: products, Ideas: ideas,
		Research: aiStub{}, Ideation: ideationOneIdea{},
		Clock: clock, Identities: ids,
	}
	svc := &Service{
		Tasks:     tasks, Products: products, Ideas: ideas,
		Gateway:   gateway,
		Budget:    &budget.Static{Cap: 100, Costs: costs},
		Checkpt:   checkpoints,
		Clock:     clock,
		IDs:       ids,
		Ship:      prPub,
		MergeShip: mergeRec,
	}
	_ = auto.RunResearch(ctx, p.ID)
	_ = auto.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	_ = auto.SubmitSwipe(ctx, list[0].ID, domain.DecisionYes)
	tt, err := svc.CreateFromApprovedIdea(ctx, list[0].ID, "spec")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.ApprovePlan(ctx, tt.ID, "")
	_ = svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, "")
	_ = svc.Dispatch(ctx, tt.ID, 1)
	tt, _ = tasks.ByID(ctx, tt.ID)
	tt.PullRequestHeadBranch = "feat/existing"
	tt.PullRequestURL = "https://github.com/acme/demo/pull/42"
	tt.PullRequestNumber = 42
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}
	if err := svc.Complete(ctx, tt.ID); err != nil {
		t.Fatal(err)
	}
	if prPub.creates != 0 {
		t.Fatalf("want 0 PR create when URL set, got %d", prPub.creates)
	}
	if mergeRec.completeCalls != 1 {
		t.Fatalf("want 1 merge complete, got %d", mergeRec.completeCalls)
	}
}

func TestAutoPROnConvoyActiveToReviewFullAuto(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}
	prPub := &fakePRPublisher{}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, err := prodSvc.Register(ctx, product.RegistrationInput{
		Name:           "p",
		WorkspaceID:    "w",
		RepoURL:        "https://github.com/acme/demo",
		AutomationTier: "full_auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	auto := &autopilot.Service{
		Products: products, Ideas: ideas,
		Research: aiStub{}, Ideation: ideationOneIdea{},
		Clock: clock, Identities: ids,
	}
	svc := &Service{
		Tasks:   tasks, Products: products, Ideas: ideas,
		Gateway: gateway,
		Budget:  &budget.Static{Cap: 100, Costs: costs},
		Checkpt: checkpoints,
		Clock:   clock,
		IDs:     ids,
		Ship:    prPub,
	}
	_ = auto.RunResearch(ctx, p.ID)
	_ = auto.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	_ = auto.SubmitSwipe(ctx, list[0].ID, domain.DecisionYes)
	tt, err := svc.CreateFromApprovedIdea(ctx, list[0].ID, "spec")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.ApprovePlan(ctx, tt.ID, "")
	_ = svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, "")
	_ = svc.Dispatch(ctx, tt.ID, 1)
	tt, _ = tasks.ByID(ctx, tt.ID)
	tt.Status = domain.StatusConvoyActive
	tt.PullRequestHeadBranch = "feat/convoy"
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetKanbanStatus(ctx, tt.ID, domain.StatusReview, ""); err != nil {
		t.Fatal(err)
	}
	if prPub.creates != 1 {
		t.Fatalf("want 1 PR from convoy->review, got %d", prPub.creates)
	}
}

func TestCreatePullRequestWithRetry(t *testing.T) {
	ctx := context.Background()
	in := ports.CreatePullRequestInput{Owner: "o", Repo: "r", HeadBranch: "h", BaseBranch: "main", Title: "t", TaskID: "tsk-1"}
	f := &flakyPRPublisher{failRemaining: 2}
	cre, err := createPullRequestWithRetry(ctx, f, in)
	if err != nil {
		t.Fatal(err)
	}
	if cre.Number != 7 || f.calls != 3 {
		t.Fatalf("got calls=%d cre=%+v", f.calls, cre)
	}
}

func TestCreatePullRequestWithRetryUnauthorizedNoRetry(t *testing.T) {
	ctx := context.Background()
	in := ports.CreatePullRequestInput{Owner: "o", Repo: "r", HeadBranch: "h", BaseBranch: "main", Title: "t"}
	u := &unauthorizedPRPublisher{}
	_, err := createPullRequestWithRetry(ctx, u, in)
	if err == nil {
		t.Fatal("want error")
	}
	if u.calls != 1 {
		t.Fatalf("want single attempt, got %d", u.calls)
	}
}

func TestOpenPullRequestNilShip(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}
	p, err := (&product.Service{Products: products, Clock: clock, IDs: ids}).Register(ctx, product.RegistrationInput{
		Name: "p", WorkspaceID: "w", RepoURL: "https://github.com/acme/demo",
	})
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
		Gateway: gateway, Budget: &budget.Static{Cap: 100, Costs: costs},
		Checkpt: checkpoints, Clock: clock, IDs: ids, Ship: nil,
	}
	_, _, err = svc.OpenPullRequest(ctx, tt.ID, "feat/x", "", "")
	if err == nil || !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestCreateFromSpecWithNewIdea(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}
	p, err := (&product.Service{Products: products, Clock: clock, IDs: ids}).Register(ctx, product.RegistrationInput{
		Name: "p", WorkspaceID: "w",
	})
	if err != nil {
		t.Fatal(err)
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
	spec := "Dark mode toggle\n\nImplement system preference sync."
	tt, err := svc.CreateFromSpecWithNewIdea(ctx, p.ID, spec, "", "research_discovery")
	if err != nil {
		t.Fatal(err)
	}
	if tt.Status != domain.StatusPlanning || tt.Spec != spec {
		t.Fatalf("task: %+v", tt)
	}
	gotIdea, err := ideas.ByID(ctx, tt.IdeaID)
	if err != nil {
		t.Fatal(err)
	}
	if gotIdea.Title != "Dark mode toggle" || !strings.Contains(gotIdea.Description, "Implement system") {
		t.Fatalf("idea title/desc: %+v", gotIdea)
	}
	if gotIdea.Category != "research_discovery" {
		t.Fatalf("idea category: got %q want research_discovery", gotIdea.Category)
	}
	if !gotIdea.Decided || gotIdea.Decision != domain.DecisionYes || gotIdea.Status != domain.IdeaStatusBuilding {
		t.Fatalf("idea workflow: %+v", gotIdea)
	}
}

func TestCreateFromSpecWithNewIdea_preferredID(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}
	p, err := (&product.Service{Products: products, Clock: clock, IDs: ids}).Register(ctx, product.RegistrationInput{
		Name: "p", WorkspaceID: "w",
	})
	if err != nil {
		t.Fatal(err)
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
	const wantID = "bot-com-github-https-1"
	tt, err := svc.CreateFromSpecWithNewIdea(ctx, p.ID, "Title\n\nBody", wantID, "")
	if err != nil {
		t.Fatal(err)
	}
	if string(tt.IdeaID) != wantID {
		t.Fatalf("idea id: got %q want %q", tt.IdeaID, wantID)
	}
}

func TestCreateFromSpecWithNewIdea_preferredIDConflict(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.SimulationMockClaw{}
	p, err := (&product.Service{Products: products, Clock: clock, IDs: ids}).Register(ctx, product.RegistrationInput{
		Name: "p", WorkspaceID: "w",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := clock.Now()
	if err := ideas.Save(ctx, &domain.Idea{
		ID: "taken", ProductID: p.ID, Title: "x", Decided: true, Decision: domain.DecisionYes,
		CreatedAt: now, UpdatedAt: now, Source: "manual", Category: "feature",
	}); err != nil {
		t.Fatal(err)
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
	_, err = svc.CreateFromSpecWithNewIdea(ctx, p.ID, "Spec", "taken", "")
	if err == nil || !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}
