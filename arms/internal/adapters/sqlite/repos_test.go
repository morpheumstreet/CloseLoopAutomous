package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

func TestProductIdeaTaskRoundTrip(t *testing.T) {
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
	now := time.Unix(1700000000, 0).UTC()
	p := &domain.Product{
		ID: domain.ProductID("prod-1"), Name: "n", Stage: domain.StageResearch, ResearchSummary: "",
		WorkspaceID: "ws", UpdatedAt: now,
		RepoURL: "https://example.com/r", RepoBranch: "main", Description: "desc",
		ProgramDocument: "program", SettingsJSON: `{"tier":1}`, IconURL: "https://example.com/i.png",
	}
	if err := ps.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	idea := &domain.Idea{
		ID: domain.IdeaID("idea-1"), ProductID: domain.ProductID("prod-1"), Title: "t", Description: "d",
		Impact: 0.5, Feasibility: 0.6, Reasoning: "r", Decided: false,
		Decision: domain.DecisionPass, CreatedAt: now,
	}
	if err := is.Save(ctx, idea); err != nil {
		t.Fatal(err)
	}
	task := &domain.Task{
		ID: domain.TaskID("task-1"), ProductID: domain.ProductID("prod-1"), IdeaID: domain.IdeaID("idea-1"), Spec: "s",
		Status: domain.StatusPlanning, PlanApproved: false, CreatedAt: now, UpdatedAt: now,
	}
	if err := ts.Save(ctx, task); err != nil {
		t.Fatal(err)
	}
	p2, err := ps.ByID(ctx, domain.ProductID("prod-1"))
	if err != nil {
		t.Fatal(err)
	}
	if p2.Name != "n" || p2.RepoURL != "https://example.com/r" || p2.ProgramDocument != "program" || p2.SettingsJSON != `{"tier":1}` {
		t.Fatalf("product %+v", p2)
	}
	list, err := is.ListByProduct(ctx, domain.ProductID("prod-1"))
	if err != nil || len(list) != 1 {
		t.Fatalf("ideas %v err %v", list, err)
	}
	tasks, err := ts.ListByProduct(ctx, domain.ProductID("prod-1"))
	if err != nil || len(tasks) != 1 || tasks[0].ID != "task-1" {
		t.Fatalf("tasks %v err %v", tasks, err)
	}
	later := now.Add(time.Hour)
	taskB := &domain.Task{
		ID: domain.TaskID("task-b"), ProductID: domain.ProductID("prod-1"), IdeaID: domain.IdeaID("idea-1"), Spec: "b",
		Status: domain.StatusInbox, PlanApproved: true, CreatedAt: later, UpdatedAt: later,
	}
	if err := ts.Save(ctx, taskB); err != nil {
		t.Fatal(err)
	}
	two, err := ts.ListByProduct(ctx, domain.ProductID("prod-1"))
	if err != nil || len(two) != 2 || two[0].ID != "task-b" {
		t.Fatalf("want newest first task-b first, got %v err %v", two, err)
	}

	cs := NewConvoyStore(db)
	conv := &domain.Convoy{
		ID: domain.ConvoyID("conv-1"), ProductID: domain.ProductID("prod-1"),
		ParentID: domain.TaskID("task-1"), CreatedAt: now,
		Subtasks: []domain.Subtask{{ID: domain.SubtaskID("s1"), AgentRole: "builder"}},
	}
	if err := cs.Save(ctx, conv); err != nil {
		t.Fatal(err)
	}
	clist, err := cs.ListByProduct(ctx, domain.ProductID("prod-1"))
	if err != nil || len(clist) != 1 || clist[0].ID != "conv-1" || len(clist[0].Subtasks) != 1 {
		t.Fatalf("convoys list: %v err %v", clist, err)
	}
}

func TestConvoyEdgesPersisted(t *testing.T) {
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
	cs := NewConvoyStore(db)
	now := time.Unix(1700000000, 0).UTC()
	p := &domain.Product{
		ID: domain.ProductID("prod-1"), Name: "n", Stage: domain.StageResearch, ResearchSummary: "",
		WorkspaceID: "ws", UpdatedAt: now,
	}
	if err := ps.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	idea := &domain.Idea{
		ID: domain.IdeaID("idea-1"), ProductID: domain.ProductID("prod-1"), Title: "t", Description: "d",
		Impact: 0.5, Feasibility: 0.6, Reasoning: "r", Decided: false,
		Decision: domain.DecisionPass, CreatedAt: now,
	}
	if err := is.Save(ctx, idea); err != nil {
		t.Fatal(err)
	}
	task := &domain.Task{
		ID: domain.TaskID("task-1"), ProductID: domain.ProductID("prod-1"), IdeaID: domain.IdeaID("idea-1"), Spec: "s",
		Status: domain.StatusPlanning, PlanApproved: false, CreatedAt: now, UpdatedAt: now,
	}
	if err := ts.Save(ctx, task); err != nil {
		t.Fatal(err)
	}
	conv := &domain.Convoy{
		ID: domain.ConvoyID("conv-1"), ProductID: domain.ProductID("prod-1"),
		ParentID: domain.TaskID("task-1"), CreatedAt: now,
		Subtasks: []domain.Subtask{
			{ID: domain.SubtaskID("s1"), AgentRole: "builder", DagLayer: 0},
			{ID: domain.SubtaskID("s2"), AgentRole: "tester", DagLayer: 1, DependsOn: []domain.SubtaskID{"s1"}},
		},
	}
	if err := cs.Save(ctx, conv); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM convoy_edges WHERE convoy_id = ?`, "conv-1").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("convoy_edges count want 1 got %d", n)
	}
	var fromID, toID string
	if err := db.QueryRowContext(ctx,
		`SELECT from_subtask_id, to_subtask_id FROM convoy_edges WHERE convoy_id = ?`, "conv-1",
	).Scan(&fromID, &toID); err != nil {
		t.Fatal(err)
	}
	if fromID != "s1" || toID != "s2" {
		t.Fatalf("edge want s1->s2 got %s->%s", fromID, toID)
	}
	// Resave: drop dependency — materialized edges must follow depends_on_json.
	conv2, err := cs.ByID(ctx, domain.ConvoyID("conv-1"))
	if err != nil {
		t.Fatal(err)
	}
	conv2.Subtasks[1].DependsOn = nil
	if err := cs.Save(ctx, conv2); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM convoy_edges WHERE convoy_id = ?`, "conv-1").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("after dep removal want 0 edges got %d", n)
	}
}

func TestConvoyByParentTaskAndDelete(t *testing.T) {
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
	cs := NewConvoyStore(db)
	now := time.Unix(1700000000, 0).UTC()
	p := &domain.Product{
		ID: domain.ProductID("prod-1"), Name: "n", Stage: domain.StageResearch, ResearchSummary: "",
		WorkspaceID: "ws", UpdatedAt: now,
	}
	if err := ps.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	idea := &domain.Idea{
		ID: domain.IdeaID("idea-1"), ProductID: domain.ProductID("prod-1"), Title: "t", Description: "d",
		Impact: 0.5, Feasibility: 0.6, Reasoning: "r", Decided: false,
		Decision: domain.DecisionPass, CreatedAt: now,
	}
	if err := is.Save(ctx, idea); err != nil {
		t.Fatal(err)
	}
	task := &domain.Task{
		ID: domain.TaskID("task-1"), ProductID: domain.ProductID("prod-1"), IdeaID: domain.IdeaID("idea-1"), Spec: "s",
		Status: domain.StatusPlanning, PlanApproved: false, CreatedAt: now, UpdatedAt: now,
	}
	if err := ts.Save(ctx, task); err != nil {
		t.Fatal(err)
	}
	conv := &domain.Convoy{
		ID: domain.ConvoyID("conv-1"), ProductID: domain.ProductID("prod-1"),
		ParentID: domain.TaskID("task-1"), CreatedAt: now,
		Subtasks: []domain.Subtask{{ID: domain.SubtaskID("s1"), AgentRole: "builder"}},
	}
	if err := cs.Save(ctx, conv); err != nil {
		t.Fatal(err)
	}
	got, err := cs.ByParentTask(ctx, domain.TaskID("task-1"))
	if err != nil || got.ID != "conv-1" {
		t.Fatalf("ByParentTask: %+v err %v", got, err)
	}
	if err := cs.Delete(ctx, domain.ConvoyID("conv-1")); err != nil {
		t.Fatal(err)
	}
	if _, err := cs.ByParentTask(ctx, domain.TaskID("task-1")); err != domain.ErrNotFound {
		t.Fatalf("after delete want ErrNotFound got %v", err)
	}
}

func TestProductSoftDelete(t *testing.T) {
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
	now := time.Unix(1700000000, 0).UTC()
	pid := domain.ProductID("prod-del")
	p := &domain.Product{
		ID: pid, Name: "n", Stage: domain.StageResearch, ResearchSummary: "",
		WorkspaceID: "ws", UpdatedAt: now,
	}
	if err := ps.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	delAt := now.Add(time.Minute)
	if err := ps.SoftDelete(ctx, pid, delAt); err != nil {
		t.Fatal(err)
	}
	if _, err := ps.ByID(ctx, pid); err != domain.ErrNotFound {
		t.Fatalf("ByID after delete: %v", err)
	}
	active, err := ps.ListAll(ctx)
	if err != nil || len(active) != 0 {
		t.Fatalf("ListAll active: %v len %d", err, len(active))
	}
	all, err := ps.ListAllIncludingDeleted(ctx)
	if err != nil || len(all) != 1 || !all[0].DeletedAt.Equal(delAt.UTC()) {
		t.Fatalf("ListAllIncludingDeleted: %+v err %v", all, err)
	}
	if err := ps.SoftDelete(ctx, pid, now); err == nil || !errors.Is(err, domain.ErrProductAlreadyDeleted) {
		t.Fatalf("second SoftDelete want ErrProductAlreadyDeleted, got %v", err)
	}
	restoreAt := delAt.Add(time.Hour)
	if err := ps.Restore(ctx, pid, restoreAt); err != nil {
		t.Fatal(err)
	}
	p2, err := ps.ByID(ctx, pid)
	if err != nil || p2.Name != "n" || p2.IsDeleted() {
		t.Fatalf("after restore: %+v err %v", p2, err)
	}
	if err := ps.Restore(ctx, pid, restoreAt); err == nil || !errors.Is(err, domain.ErrProductNotDeleted) {
		t.Fatalf("Restore when active want ErrProductNotDeleted, got %v", err)
	}
	a, d, err := ps.CountLifecycle(ctx)
	if err != nil || a != 1 || d != 0 {
		t.Fatalf("CountLifecycle after restore: active=%d deleted=%d err %v", a, d, err)
	}
}

func TestProductSaveIfUnchangedSince(t *testing.T) {
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
	now := time.Unix(1700000000, 0).UTC()
	p := &domain.Product{
		ID: domain.ProductID("p-opt"), Name: "a", Stage: domain.StageResearch, ResearchSummary: "",
		WorkspaceID: "ws", UpdatedAt: now,
	}
	if err := ps.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	loaded, err := ps.ByID(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	later := now.Add(time.Hour)
	loaded.Name = "b"
	loaded.UpdatedAt = later
	if err := ps.SaveIfUnchangedSince(ctx, loaded, now); err != nil {
		t.Fatal(err)
	}
	again, err := ps.ByID(ctx, p.ID)
	if err != nil || again.Name != "b" {
		t.Fatalf("after optimistic save: %+v err %v", again, err)
	}
	again.Name = "c"
	again.UpdatedAt = later.Add(time.Hour)
	if err := ps.SaveIfUnchangedSince(ctx, again, now); err == nil || !errors.Is(err, domain.ErrStaleEntity) {
		t.Fatalf("second save want ErrStaleEntity, got %v", err)
	}
}
