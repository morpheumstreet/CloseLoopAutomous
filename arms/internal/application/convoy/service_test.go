package convoy

import (
	"context"
	"testing"
	"time"

	gw "github.com/closeloopautomous/arms/internal/adapters/gateway"
	"github.com/closeloopautomous/arms/internal/adapters/identity"
	"github.com/closeloopautomous/arms/internal/adapters/memory"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/application/product"
	"github.com/closeloopautomous/arms/internal/domain"
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

	svc := &Service{
		Convoys:  convoys,
		Tasks:    tasks,
		Products: products,
		Gateway:  gateway,
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
