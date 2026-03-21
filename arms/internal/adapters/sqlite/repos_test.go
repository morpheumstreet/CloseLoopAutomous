package sqlite

import (
	"context"
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
