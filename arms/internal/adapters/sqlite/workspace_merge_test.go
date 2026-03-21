package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

func TestWorkspaceMergeQueueCompletesHeadFirst(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	ps := NewProductStore(db)
	is := NewIdeaStore(db)
	ts := NewTaskStore(db)
	ws := NewWorkspaceStore(db)
	now := time.Unix(1700000000, 0).UTC()
	p := &domain.Product{
		ID: domain.ProductID("prod-mq"), Name: "n", Stage: domain.StageResearch, ResearchSummary: "",
		WorkspaceID: "ws", UpdatedAt: now,
	}
	if err := ps.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	idea := &domain.Idea{
		ID: domain.IdeaID("idea-x"), ProductID: p.ID, Title: "t", Description: "d",
		Impact: 0.5, Feasibility: 0.5, Reasoning: "r", Decided: true, Decision: domain.DecisionYes, CreatedAt: now,
	}
	if err := is.Save(ctx, idea); err != nil {
		t.Fatal(err)
	}
	for _, id := range []domain.TaskID{"t-a", "t-b"} {
		task := &domain.Task{
			ID: id, ProductID: p.ID, IdeaID: domain.IdeaID("idea-x"), Spec: "s",
			Status: domain.StatusInProgress, PlanApproved: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := ts.Save(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	if err := ws.Enqueue(ctx, p.ID, "t-a", now); err != nil {
		t.Fatal(err)
	}
	if err := ws.Enqueue(ctx, p.ID, "t-b", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	err = ws.CompletePendingForTask(ctx, "t-b")
	if err == nil {
		t.Fatal("expected not head error")
	}
	if !errors.Is(err, domain.ErrNotMergeQueueHead) {
		t.Fatalf("got %v want ErrNotMergeQueueHead", err)
	}
	if err := ws.CompletePendingForTask(ctx, "t-a"); err != nil {
		t.Fatal(err)
	}
	if err := ws.CompletePendingForTask(ctx, "t-b"); err != nil {
		t.Fatal(err)
	}
}
