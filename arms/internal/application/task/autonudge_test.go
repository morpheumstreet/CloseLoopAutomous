package task

import (
	"context"
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

func TestAutoNudgeStallIfDue(t *testing.T) {
	ctx := context.Background()
	t0 := time.Unix(1700000000, 0)
	clock := timeadapter.Fixed{T: t0}
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
		Gateway:     gateway,
		Budget:      &budget.Static{Cap: 100, Costs: costs},
		Checkpt:     checkpoints,
		Clock:       clock,
		IDs:         ids,
		Ship:        shipping.PullRequestNoop{},
		Events:      hub,
		Gate:        NewProductGate(),
		AgentHealth: agentHealth,
		AutoStallNudge: AutoStallNudgeSettings{
			Enabled:        true,
			StaleThreshold: 5 * time.Minute,
			Cooldown:       time.Hour,
			MaxPerDay:      10,
		},
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
	_ = svc.ApprovePlan(ctx, tt.ID, "")
	_ = svc.SetKanbanStatus(ctx, tt.ID, domain.StatusAssigned, "")
	if err := svc.Dispatch(ctx, tt.ID, 1); err != nil {
		t.Fatal(err)
	}
	tt, err = tasks.ByID(ctx, tt.ID)
	if err != nil {
		t.Fatal(err)
	}
	// No agent health row → stalled
	r, err := svc.AutoNudgeStallIfDue(ctx, tt)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Nudged || r.StallReason != "no_heartbeat" {
		t.Fatalf("first nudge: %+v", r)
	}
	// Cooldown: same clock → skip
	r2, err := svc.AutoNudgeStallIfDue(ctx, tt)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Nudged || r2.SkipReason != "cooldown" {
		t.Fatalf("want cooldown skip, got %+v", r2)
	}

	svc.AutoStallNudge.Enabled = false
	r3, err := svc.AutoNudgeStallIfDue(ctx, tt)
	if err != nil {
		t.Fatal(err)
	}
	if r3.Nudged || r3.SkipReason != "disabled" {
		t.Fatalf("want disabled, got %+v", r3)
	}
}
