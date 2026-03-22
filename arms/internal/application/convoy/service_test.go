package convoy

import (
	"context"
	"errors"
	"strings"
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
	c, err := svc.Create(ctx, CreateInput{ParentTaskID: tt.ID, ProductID: p.ID, Subtasks: []domain.Subtask{{AgentRole: "builder"}}})
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
	c, err := svc.Create(ctx, CreateInput{ParentTaskID: tt.ID, ProductID: p.ID, Subtasks: []domain.Subtask{
		{ID: bID, AgentRole: "builder"},
		{ID: testerID, AgentRole: "tester", DependsOn: []domain.SubtaskID{bID}},
	}})
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
	c, err := svc.Create(ctx, CreateInput{ParentTaskID: tt.ID, ProductID: p.ID, Subtasks: []domain.Subtask{{ID: "s1", AgentRole: "builder"}}})
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

func TestCreateConvoyRejectsDependencyCycle(t *testing.T) {
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
	_ = tasks.Save(ctx, tt)
	svc := &Service{Convoys: convoys, Tasks: tasks, Products: products, Gateway: gateway, Clock: clock, IDs: ids}
	a := domain.SubtaskID("a")
	b := domain.SubtaskID("b")
	_, err := svc.Create(ctx, CreateInput{ParentTaskID: tt.ID, ProductID: p.ID, Subtasks: []domain.Subtask{
		{ID: a, AgentRole: "x", DependsOn: []domain.SubtaskID{b}},
		{ID: b, AgentRole: "y", DependsOn: []domain.SubtaskID{a}},
	}})
	if err == nil {
		t.Fatal("want validation error")
	}
}

// subtaskFailGW fails DispatchSubtask per subtask id for a fixed number of calls, then delegates to Stub.
type subtaskFailGW struct {
	inner *gw.Stub
	left  map[domain.SubtaskID]int
}

func (f *subtaskFailGW) DispatchTask(ctx context.Context, task domain.Task) (string, error) {
	return f.inner.DispatchTask(ctx, task)
}

func (f *subtaskFailGW) DispatchSubtask(ctx context.Context, parent domain.Task, sub domain.Subtask) (string, error) {
	if f.left != nil {
		if n, ok := f.left[sub.ID]; ok && n > 0 {
			f.left[sub.ID] = n - 1
			return "", domain.ErrGateway
		}
	}
	return f.inner.DispatchSubtask(ctx, parent, sub)
}

func TestDispatchReady_RetriesGatewayThenSucceeds(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	gateway := &subtaskFailGW{inner: &gw.Stub{}, left: map[domain.SubtaskID]int{"s1": 2}}

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
	bpol := &budget.Composite{Costs: memory.NewCostStore(), Caps: memory.NewCostCapStore(), Clock: clock, DefaultCumulative: 0}
	svc := &Service{
		Convoys: convoys, Tasks: tasks, Products: products, Gateway: gateway, Budget: bpol,
		Clock: clock, IDs: ids, MaxSubtaskDispatchAttempts: 5,
	}
	c, err := svc.Create(ctx, CreateInput{ParentTaskID: tt.ID, ProductID: p.ID, Subtasks: []domain.Subtask{{ID: "s1", AgentRole: "builder"}}})
	if err != nil {
		t.Fatal(err)
	}
	for wave := 0; wave < 3; wave++ {
		if err := svc.DispatchReady(ctx, c.ID, 0); err != nil {
			t.Fatalf("wave %d: %v", wave, err)
		}
	}
	cf, _ := convoys.ByID(ctx, c.ID)
	if !cf.Subtasks[0].Dispatched || cf.Subtasks[0].DispatchAttempts != 0 {
		t.Fatalf("want dispatched and attempts reset, got %#v", cf.Subtasks[0])
	}
}

func TestDispatchReady_GatewayExhausted(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	gateway := &subtaskFailGW{inner: &gw.Stub{}, left: map[domain.SubtaskID]int{"s1": 99}}

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
	bpol := &budget.Composite{Costs: memory.NewCostStore(), Caps: memory.NewCostCapStore(), Clock: clock, DefaultCumulative: 0}
	svc := &Service{
		Convoys: convoys, Tasks: tasks, Products: products, Gateway: gateway, Budget: bpol,
		Clock: clock, IDs: ids, MaxSubtaskDispatchAttempts: 2,
	}
	c, err := svc.Create(ctx, CreateInput{ParentTaskID: tt.ID, ProductID: p.ID, Subtasks: []domain.Subtask{{ID: "s1", AgentRole: "builder"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DispatchReady(ctx, c.ID, 0); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}
	err = svc.DispatchReady(ctx, c.ID, 0)
	if err == nil || !errors.Is(err, domain.ErrGateway) {
		t.Fatalf("want ErrGateway on second dispatch, got %v", err)
	}
	cf, _ := convoys.ByID(ctx, c.ID)
	if cf.Subtasks[0].Dispatched || cf.Subtasks[0].DispatchAttempts != 2 {
		t.Fatalf("want 2 attempts, not dispatched, got %#v", cf.Subtasks[0])
	}
}

func TestDispatchReady_ParentHealthBlocks(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	health := memory.NewAgentHealthStore()
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
	if err := health.UpsertHeartbeat(ctx, tt.ID, p.ID, "stalled", "{}", clock.Now()); err != nil {
		t.Fatal(err)
	}
	bpol := &budget.Composite{Costs: memory.NewCostStore(), Caps: memory.NewCostCapStore(), Clock: clock, DefaultCumulative: 0}
	svc := &Service{
		Convoys: convoys, Tasks: tasks, Products: products, Gateway: gateway, Budget: bpol,
		Health: health, Clock: clock, IDs: ids,
	}
	c, err := svc.Create(ctx, CreateInput{ParentTaskID: tt.ID, ProductID: p.ID, Subtasks: []domain.Subtask{{ID: "s1", AgentRole: "builder"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DispatchReady(ctx, c.ID, 0); err != nil {
		t.Fatal(err)
	}
	cf, _ := convoys.ByID(ctx, c.ID)
	if cf.Subtasks[0].Dispatched {
		t.Fatal("dispatch should be deferred while parent health is stalled")
	}
	_ = health.UpsertHeartbeat(ctx, tt.ID, p.ID, "healthy", "{}", clock.Now())
	if err := svc.DispatchReady(ctx, c.ID, 0); err != nil {
		t.Fatal(err)
	}
	cf2, _ := convoys.ByID(ctx, c.ID)
	if !cf2.Subtasks[0].Dispatched {
		t.Fatal("expected dispatch after health recovered")
	}
}

func TestCreateAssignsDagLayersAndMetadata(t *testing.T) {
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
	_ = tasks.Save(ctx, tt)
	svc := &Service{Convoys: convoys, Tasks: tasks, Products: products, Gateway: gateway, Clock: clock, IDs: ids}
	bID := domain.SubtaskID("b1")
	cID := domain.SubtaskID("c1")
	out, err := svc.Create(ctx, CreateInput{
		ParentTaskID:   tt.ID,
		ProductID:      p.ID,
		MetadataJSON:   `{"plan":"alpha"}`,
		Subtasks: []domain.Subtask{
			{ID: bID, AgentRole: "builder", Title: "Build", MetadataJSON: `{"k":1}`},
			{ID: cID, AgentRole: "checker", DependsOn: []domain.SubtaskID{bID}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.MetadataJSON, "plan") {
		t.Fatalf("convoy metadata: %s", out.MetadataJSON)
	}
	var bst, cst domain.Subtask
	for i := range out.Subtasks {
		if out.Subtasks[i].ID == bID {
			bst = out.Subtasks[i]
		}
		if out.Subtasks[i].ID == cID {
			cst = out.Subtasks[i]
		}
	}
	if bst.DagLayer != 0 || cst.DagLayer != 1 {
		t.Fatalf("dag_layer b=%d c=%d", bst.DagLayer, cst.DagLayer)
	}
	if bst.Title != "Build" || !strings.Contains(bst.MetadataJSON, "k") {
		t.Fatalf("subtask fields: title=%q meta=%q", bst.Title, bst.MetadataJSON)
	}
}

func TestPostMailKindAndRecipient(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	mail := memory.NewConvoyMailStore()
	gateway := &gw.Stub{}
	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	tt := &domain.Task{
		ID: domain.TaskID("task-1"), ProductID: p.ID, IdeaID: domain.IdeaID("idea-1"),
		Spec: "s", Status: domain.StatusAssigned, PlanApproved: true,
		CreatedAt: clock.Now(), UpdatedAt: clock.Now(),
	}
	_ = tasks.Save(ctx, tt)
	svc := &Service{Convoys: convoys, Tasks: tasks, Products: products, Gateway: gateway, Mail: mail, Clock: clock, IDs: ids}
	s1 := domain.SubtaskID("s1")
	s2 := domain.SubtaskID("s2")
	c, err := svc.Create(ctx, CreateInput{ParentTaskID: tt.ID, ProductID: p.ID, Subtasks: []domain.Subtask{
		{ID: s1, AgentRole: "a"},
		{ID: s2, AgentRole: "b", DependsOn: []domain.SubtaskID{s1}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.PostMail(ctx, c.ID, domain.ConvoyMailDraft{FromSubtaskID: s1, ToSubtaskID: s2, Kind: "handoff", Body: "pass"}); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListMail(ctx, c.ID, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %#v err %v", list, err)
	}
	if list[0].Kind != "handoff" || list[0].ToSubtaskID != s2 {
		t.Fatalf("msg: %#v", list[0])
	}
}
