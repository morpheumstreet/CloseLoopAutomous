package task

import (
	"context"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/budget"
	gw "github.com/closeloopautomous/arms/internal/adapters/gateway"
	"github.com/closeloopautomous/arms/internal/adapters/memory"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/domain"
)

func TestAutoStallReassignPicksAlternateAgent(t *testing.T) {
	ctx := context.Background()
	t0 := time.Unix(1700000000, 0).UTC()
	oldHB := t0.Add(-10 * time.Minute)
	clock := timeadapter.Fixed{T: t0}

	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.Stub{}
	agentHealth := memory.NewAgentHealthStore()
	execAgents := memory.NewExecutionAgentStore()

	pid := domain.ProductID("p1")
	_ = products.Save(ctx, &domain.Product{
		ID: pid, Name: "p", WorkspaceID: "w", UpdatedAt: t0,
	})
	_ = execAgents.Save(ctx, &domain.ExecutionAgent{ID: "ag1", DisplayName: "a1", ProductID: pid, CreatedAt: t0.Add(-2 * time.Hour)})
	_ = execAgents.Save(ctx, &domain.ExecutionAgent{ID: "ag2", DisplayName: "a2", ProductID: pid, CreatedAt: t0.Add(-time.Hour)})

	tid := domain.TaskID("t1")
	_ = tasks.Save(ctx, &domain.Task{
		ID:                      tid,
		ProductID:               pid,
		Status:                  domain.StatusInProgress,
		PlanApproved:            true,
		Spec:                    "do",
		CurrentExecutionAgentID: "ag1",
		CreatedAt:               t0,
		UpdatedAt:               t0,
	})
	_ = agentHealth.UpsertHeartbeat(ctx, tid, pid, string(domain.StatusInProgress), `{}`, oldHB)

	svc := &Service{
		Tasks:       tasks,
		Products:    products,
		Gateway:     gateway,
		Budget:      &budget.Static{Cap: 100, Costs: costs},
		Checkpt:     checkpoints,
		Clock:       clock,
		AgentHealth: agentHealth,
		ExecAgents:  execAgents,
		AutoStallNudge: AutoStallNudgeSettings{
			Enabled:        true,
			StaleThreshold: 5 * time.Minute,
			Cooldown:       time.Hour,
			MaxPerDay:      10,
		},
		AutoStallReassign: AutoStallReassignSettings{
			Enabled:   true,
			Cooldown:  time.Hour,
			MaxPerDay: 10,
		},
	}

	tt, err := tasks.ByID(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	r, err := svc.AutoNudgeStallIfDue(ctx, tt)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Reassigned || r.ReassignTo != "ag2" || r.Nudged {
		t.Fatalf("want reassign to ag2, got %+v", r)
	}
	tt2, err := tasks.ByID(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	if tt2.CurrentExecutionAgentID != "ag2" {
		t.Fatalf("task binding: got %q", tt2.CurrentExecutionAgentID)
	}
	if gateway.Seq < 1 {
		t.Fatal("expected gateway dispatch")
	}
}

func TestAutoStallReassignSingleAgentFallsBackToNudge(t *testing.T) {
	ctx := context.Background()
	t0 := time.Unix(1700000000, 0).UTC()
	oldHB := t0.Add(-10 * time.Minute)
	clock := timeadapter.Fixed{T: t0}

	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	gateway := &gw.Stub{}
	agentHealth := memory.NewAgentHealthStore()
	execAgents := memory.NewExecutionAgentStore()

	pid := domain.ProductID("p1")
	_ = products.Save(ctx, &domain.Product{
		ID: pid, Name: "p", WorkspaceID: "w", UpdatedAt: t0,
	})
	_ = execAgents.Save(ctx, &domain.ExecutionAgent{ID: "ag1", DisplayName: "only", ProductID: pid, CreatedAt: t0})

	tid := domain.TaskID("t1")
	_ = tasks.Save(ctx, &domain.Task{
		ID:                      tid,
		ProductID:               pid,
		Status:                  domain.StatusInProgress,
		PlanApproved:            true,
		Spec:                    "do",
		CurrentExecutionAgentID: "ag1",
		CreatedAt:               t0,
		UpdatedAt:               t0,
	})
	_ = agentHealth.UpsertHeartbeat(ctx, tid, pid, string(domain.StatusInProgress), `{}`, oldHB)

	svc := &Service{
		Tasks:       tasks,
		Products:    products,
		Gateway:     gateway,
		Budget:      &budget.Static{Cap: 100, Costs: costs},
		Checkpt:     checkpoints,
		Clock:       clock,
		AgentHealth: agentHealth,
		ExecAgents:  execAgents,
		AutoStallNudge: AutoStallNudgeSettings{
			Enabled:        true,
			StaleThreshold: 5 * time.Minute,
			Cooldown:       time.Hour,
			MaxPerDay:      10,
		},
		AutoStallReassign: AutoStallReassignSettings{
			Enabled:   true,
			Cooldown:  time.Hour,
			MaxPerDay: 10,
		},
	}

	tt, _ := tasks.ByID(ctx, tid)
	r, err := svc.AutoNudgeStallIfDue(ctx, tt)
	if err != nil {
		t.Fatal(err)
	}
	if r.Reassigned || !r.Nudged {
		t.Fatalf("want nudge fallback, got %+v", r)
	}
}
