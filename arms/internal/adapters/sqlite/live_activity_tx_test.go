package sqlite

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

func TestLiveActivityTXCompleteTaskWithEvent(t *testing.T) {
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
	x := NewLiveActivityTX(db)
	now := time.Unix(1700000000, 0).UTC()
	p := &domain.Product{
		ID: domain.ProductID("prod-1"), Name: "n", Stage: domain.StageResearch,
		WorkspaceID: "ws", UpdatedAt: now,
	}
	if err := ps.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	idea := &domain.Idea{
		ID: domain.IdeaID("idea-1"), ProductID: domain.ProductID("prod-1"), Title: "t",
		Decided: true, Decision: domain.DecisionYes, CreatedAt: now,
	}
	if err := is.Save(ctx, idea); err != nil {
		t.Fatal(err)
	}
	task := &domain.Task{
		ID: domain.TaskID("task-1"), ProductID: domain.ProductID("prod-1"), IdeaID: domain.IdeaID("idea-1"),
		Spec: "s", Status: domain.StatusInProgress, PlanApproved: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := ts.Save(ctx, task); err != nil {
		t.Fatal(err)
	}
	detail := `{"source":"test"}`
	ev := ports.LiveActivityEvent{
		Type: "task_completed", Ts: now.Format(time.RFC3339Nano),
		ProductID: "prod-1", TaskID: "task-1", Data: map[string]any{"source": "test"},
	}
	if err := x.CompleteTaskWithEvent(ctx, task.ID, now, "completed", detail, ev); err != nil {
		t.Fatal(err)
	}
	t2, err := ts.ByID(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if t2.Status != domain.StatusDone {
		t.Fatalf("want done got %s", t2.Status)
	}
	ah := NewAgentHealthStore(db)
	row, err := ah.ByTask(ctx, task.ID)
	if err != nil || row == nil || row.Status != "completed" || row.DetailJSON != detail {
		t.Fatalf("agent health %+v err %v", row, err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_outbox WHERE delivered_at IS NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 pending outbox row, got %d", n)
	}
	var payload string
	if err := db.QueryRowContext(ctx, `SELECT payload_json FROM event_outbox ORDER BY id DESC LIMIT 1`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var got ports.LiveActivityEvent
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if got.Type != "task_completed" || got.TaskID != "task-1" {
		t.Fatalf("outbox payload: %+v", got)
	}
	// Idempotent: already done
	if err := x.CompleteTaskWithEvent(ctx, task.ID, now.Add(time.Minute), "completed", `{"source":"again"}`, ev); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_outbox WHERE delivered_at IS NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("want 2 pending outbox rows after idempotent complete, got %d", n)
	}
}
