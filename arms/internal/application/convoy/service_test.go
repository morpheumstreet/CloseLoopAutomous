package convoy

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/budget"
	gw "github.com/closeloopautomous/arms/internal/adapters/gateway"
	"github.com/closeloopautomous/arms/internal/adapters/identity"
	"github.com/closeloopautomous/arms/internal/adapters/memory"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/application/product"
	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

func TestGetAndListByProduct(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	gateway := &gw.Stub{}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})

	tt := &domain.Task{
		ID: domain.TaskID("task-1"), ProductID: p.ID, IdeaID: domain.IdeaID("idea-1"),
		Spec: "s", Status: domain.StatusAssigned, PlanApproved: true,
		CreatedAt: clock.Now(), UpdatedAt: clock.Now(),
	}
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}

	costs := memory.NewCostStore()
	caps := memory.NewCostCapStore()
	bpol := &budget.Composite{Costs: costs, Caps: caps, Clock: clock, DefaultCumulative: 0}
	svc := &Service{
		Convoys:  convoys,
		Tasks:    tasks,
		Products: products,
		Gateway:  gateway,
		Budget:   bpol,
		Clock:    clock,
		IDs:      ids,
	}
	c, err := svc.Create(ctx, tt.ID, p.ID, []domain.Subtask{{AgentRole: "builder"}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(ctx, c.ID)
	if err != nil || got.ID != c.ID || len(got.Subtasks) != 1 {
		t.Fatalf("Get: %+v err %v", got, err)
	}
	list, err := svc.ListByProduct(ctx, p.ID)
	if err != nil || len(list) != 1 || list[0].ID != c.ID {
		t.Fatalf("ListByProduct: %v err %v", list, err)
	}
}

type recordPub struct {
	mu  sync.Mutex
	evs []ports.LiveActivityEvent
}

func (r *recordPub) Publish(_ context.Context, ev ports.LiveActivityEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.evs = append(r.evs, ev)
	return nil
}

func TestDispatchReady_WaitsForDependencyCompletion(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	gateway := &gw.Stub{}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	tt := &domain.Task{
		ID: domain.TaskID("task-1"), ProductID: p.ID, IdeaID: domain.IdeaID("idea-1"),
		Spec: "s", Status: domain.StatusAssigned, PlanApproved: true,
		CreatedAt: clock.Now(), UpdatedAt: clock.Now(),
	}
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}
	bID := domain.SubtaskID("builder-1")
	testerID := domain.SubtaskID("tester-1")
	costs := memory.NewCostStore()
	caps := memory.NewCostCapStore()
	bpol := &budget.Composite{Costs: costs, Caps: caps, Clock: clock, DefaultCumulative: 0}
	svc := &Service{
		Convoys: convoys, Tasks: tasks, Products: products, Gateway: gateway, Budget: bpol, Clock: clock, IDs: ids,
	}
	c, err := svc.Create(ctx, tt.ID, p.ID, []domain.Subtask{
		{ID: bID, AgentRole: "builder"},
		{ID: testerID, AgentRole: "tester", DependsOn: []domain.SubtaskID{bID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DispatchReady(ctx, c.ID, 0); err != nil {
		t.Fatal(err)
	}
	c1, _ := convoys.ByID(ctx, c.ID)
	if !c1.Subtasks[0].Dispatched || c1.Subtasks[1].Dispatched {
		t.Fatalf("first wave: want only builder dispatched, got %#v", c1.Subtasks)
	}
	if err := svc.DispatchReady(ctx, c.ID, 0); err != nil {
		t.Fatal(err)
	}
	c2, _ := convoys.ByID(ctx, c.ID)
	if c2.Subtasks[1].Dispatched {
		t.Fatal("tester must not dispatch before builder completes")
	}
	if err := svc.CompleteSubtask(ctx, c.ID, bID, tt.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DispatchReady(ctx, c.ID, 0); err != nil {
		t.Fatal(err)
	}
	c3, _ := convoys.ByID(ctx, c.ID)
	if !c3.Subtasks[0].Completed || !c3.Subtasks[1].Dispatched {
		t.Fatalf("after builder done: want tester dispatched, got %#v", c3.Subtasks)
	}
}

func TestDispatchReady_PublishesLiveActivity(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	gateway := &gw.Stub{}
	pub := &recordPub{}

	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	tt := &domain.Task{
		ID: domain.TaskID("task-1"), ProductID: p.ID, IdeaID: domain.IdeaID("idea-1"),
		Spec: "s", Status: domain.StatusAssigned, PlanApproved: true,
		CreatedAt: clock.Now(), UpdatedAt: clock.Now(),
	}
	if err := tasks.Save(ctx, tt); err != nil {
		t.Fatal(err)
	}
	costs2 := memory.NewCostStore()
	caps2 := memory.NewCostCapStore()
	bpol2 := &budget.Composite{Costs: costs2, Caps: caps2, Clock: clock, DefaultCumulative: 0}
	svc := &Service{
		Convoys: convoys, Tasks: tasks, Products: products, Gateway: gateway, Budget: bpol2,
		Clock: clock, IDs: ids, Events: pub,
	}
	c, err := svc.Create(ctx, tt.ID, p.ID, []domain.Subtask{{ID: "s1", AgentRole: "builder"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DispatchReady(ctx, c.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := svc.CompleteSubtask(ctx, c.ID, "s1", tt.ID); err != nil {
		t.Fatal(err)
	}
	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.evs) != 2 || pub.evs[0].Type != "convoy_subtask_dispatched" || pub.evs[1].Type != "convoy_subtask_completed" {
		t.Fatalf("events: %#v", pub.evs)
	}
}
