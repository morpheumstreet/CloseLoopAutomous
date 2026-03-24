package taskchat

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/identity"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/memory"
	timeadapter "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/time"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type capPub struct {
	mu  sync.Mutex
	evs []ports.LiveActivityEvent
}

func (c *capPub) Publish(_ context.Context, ev ports.LiveActivityEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evs = append(c.evs, ev)
	return nil
}

func TestAppendAndListChronological(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	chat := memory.NewTaskChatStore()
	pub := &capPub{}
	_ = products.Save(ctx, &domain.Product{ID: "p1", Name: "n", Stage: domain.StageResearch, WorkspaceID: "w", UpdatedAt: clock.Now()})
	_ = tasks.Save(ctx, &domain.Task{
		ID: "t1", ProductID: "p1", IdeaID: "i1", Spec: "s", Status: domain.StatusInProgress,
		PlanApproved: true, CreatedAt: clock.Now(), UpdatedAt: clock.Now(),
	})
	svc := &Service{Chat: chat, Tasks: tasks, Products: products, Clock: clock, IDs: ids, Events: pub}
	m1, err := svc.Append(ctx, "t1", "first", "operator", false)
	if err != nil {
		t.Fatal(err)
	}
	clock.T = clock.T.Add(time.Second)
	m2, err := svc.Append(ctx, "t1", "second", "agent", false)
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListByTask(ctx, "t1", 50)
	if err != nil || len(list) != 2 {
		t.Fatalf("list %v err %v", list, err)
	}
	if list[0].ID != m1.ID || list[1].ID != m2.ID {
		t.Fatalf("order: %#v", list)
	}
	pub.mu.Lock()
	n := len(pub.evs)
	pub.mu.Unlock()
	if n != 2 {
		t.Fatalf("events %d", n)
	}
}

func TestQueueAndAck(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	tasks := memory.NewTaskStore()
	chat := memory.NewTaskChatStore()
	_ = products.Save(ctx, &domain.Product{ID: "p1", Name: "n", Stage: domain.StageResearch, WorkspaceID: "w", UpdatedAt: clock.Now()})
	_ = tasks.Save(ctx, &domain.Task{
		ID: "t1", ProductID: "p1", IdeaID: "i1", Spec: "s", Status: domain.StatusInProgress,
		PlanApproved: true, CreatedAt: clock.Now(), UpdatedAt: clock.Now(),
	})
	svc := &Service{Chat: chat, Tasks: tasks, Products: products, Clock: clock, IDs: ids}
	msg, err := svc.Append(ctx, "t1", "note for agent", "operator", true)
	if err != nil {
		t.Fatal(err)
	}
	q, err := svc.ListQueue(ctx, "p1", 10)
	if err != nil || len(q) != 1 {
		t.Fatalf("queue %v err %v", q, err)
	}
	if err := svc.AckQueue(ctx, "p1", msg.ID); err != nil {
		t.Fatal(err)
	}
	q2, _ := svc.ListQueue(ctx, "p1", 10)
	if len(q2) != 0 {
		t.Fatalf("want empty queue got %d", len(q2))
	}
}
