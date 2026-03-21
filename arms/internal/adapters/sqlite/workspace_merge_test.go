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

func TestWorkspaceMergeQueueReserveAndFinishShip(t *testing.T) {
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
		ID: domain.ProductID("prod-ship"), Name: "n", Stage: domain.StageResearch,
		WorkspaceID: "ws", UpdatedAt: now,
	}
	_ = ps.Save(ctx, p)
	idea := &domain.Idea{
		ID: domain.IdeaID("idea-s"), ProductID: p.ID, Title: "t", Description: "d",
		Impact: 0.5, Feasibility: 0.5, Reasoning: "r", Decided: true, Decision: domain.DecisionYes, CreatedAt: now,
	}
	_ = is.Save(ctx, idea)
	task := &domain.Task{
		ID: "t-ship", ProductID: p.ID, IdeaID: idea.ID, Spec: "s",
		Status: domain.StatusInProgress, PlanApproved: true, CreatedAt: now, UpdatedAt: now,
	}
	_ = ts.Save(ctx, task)
	_ = ws.Enqueue(ctx, p.ID, task.ID, now)
	rowID, err := ws.ReserveHeadForShip(ctx, task.ID, "worker-1", now.Add(time.Minute))
	if err != nil || rowID == 0 {
		t.Fatalf("reserve: %v id=%d", err, rowID)
	}
	if err := ws.FinishShip(ctx, rowID, "worker-1", domain.MergeShipResult{State: domain.MergeShipMerged, MergedSHA: "abc"}, nil); err != nil {
		t.Fatal(err)
	}
	list, err := ws.ListPendingByProduct(ctx, p.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("want no pending after merge, got %d", len(list))
	}
}

func TestWorkspaceMergeQueueCancelTail(t *testing.T) {
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
		ID: domain.ProductID("prod-cancel"), Name: "n", Stage: domain.StageResearch,
		WorkspaceID: "ws", UpdatedAt: now,
	}
	_ = ps.Save(ctx, p)
	idea := &domain.Idea{
		ID: domain.IdeaID("idea-c"), ProductID: p.ID, Title: "t", Description: "d",
		Impact: 0.5, Feasibility: 0.5, Reasoning: "r", Decided: true, Decision: domain.DecisionYes, CreatedAt: now,
	}
	_ = is.Save(ctx, idea)
	for _, id := range []domain.TaskID{"t-h", "t-t"} {
		task := &domain.Task{
			ID: id, ProductID: p.ID, IdeaID: idea.ID, Spec: "s",
			Status: domain.StatusInProgress, PlanApproved: true, CreatedAt: now, UpdatedAt: now,
		}
		_ = ts.Save(ctx, task)
	}
	_ = ws.Enqueue(ctx, p.ID, "t-h", now)
	_ = ws.Enqueue(ctx, p.ID, "t-t", now.Add(time.Second))
	if err := ws.CancelPendingForTask(ctx, "t-t"); err != nil {
		t.Fatal(err)
	}
	n, _ := ws.CountPendingByProduct(ctx, p.ID)
	if n != 1 {
		t.Fatalf("pending count want 1 got %d", n)
	}
	if err := ws.CancelPendingForTask(ctx, "t-h"); err != nil {
		t.Fatal(err)
	}
	n2, _ := ws.CountPendingByProduct(ctx, p.ID)
	if n2 != 0 {
		t.Fatalf("after head cancel want 0 pending got %d", n2)
	}
}

func TestWorkspaceMergeQueueCancelHeadBlockedByLease(t *testing.T) {
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
	now := time.Now().UTC().Truncate(time.Millisecond)
	p := &domain.Product{
		ID: domain.ProductID("prod-lease"), Name: "n", Stage: domain.StageResearch,
		WorkspaceID: "ws", UpdatedAt: now,
	}
	_ = ps.Save(ctx, p)
	idea := &domain.Idea{
		ID: domain.IdeaID("idea-l"), ProductID: p.ID, Title: "t", Description: "d",
		Impact: 0.5, Feasibility: 0.5, Reasoning: "r", Decided: true, Decision: domain.DecisionYes, CreatedAt: now,
	}
	_ = is.Save(ctx, idea)
	task := &domain.Task{
		ID: "t-lease", ProductID: p.ID, IdeaID: idea.ID, Spec: "s",
		Status: domain.StatusInProgress, PlanApproved: true, CreatedAt: now, UpdatedAt: now,
	}
	_ = ts.Save(ctx, task)
	_ = ws.Enqueue(ctx, p.ID, task.ID, now)
	_, err = ws.ReserveHeadForShip(ctx, task.ID, "w1", time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	err = ws.CancelPendingForTask(ctx, task.ID)
	if err == nil {
		t.Fatal("expected merge ship busy")
	}
	if !errors.Is(err, domain.ErrMergeShipBusy) {
		t.Fatalf("got %v want ErrMergeShipBusy", err)
	}
}
