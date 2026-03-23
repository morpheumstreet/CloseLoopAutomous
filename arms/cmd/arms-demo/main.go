package main

import (
	"context"
	"fmt"
	"log"

	"github.com/closeloopautomous/arms/internal/adapters/ai"
	"github.com/closeloopautomous/arms/internal/adapters/budget"
	gw "github.com/closeloopautomous/arms/internal/adapters/gateway"
	"github.com/closeloopautomous/arms/internal/adapters/identity"
	"github.com/closeloopautomous/arms/internal/adapters/memory"
	"github.com/closeloopautomous/arms/internal/adapters/shipping"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/application/autopilot"
	"github.com/closeloopautomous/arms/internal/application/convoy"
	"github.com/closeloopautomous/arms/internal/application/cost"
	knowledgeapp "github.com/closeloopautomous/arms/internal/application/knowledge"
	"github.com/closeloopautomous/arms/internal/application/product"
	"github.com/closeloopautomous/arms/internal/application/task"
	"github.com/closeloopautomous/arms/internal/domain"
)

// Non-HTTP demo of the orchestration pipeline (subset of platform wiring).
// Mirrors HTTP auto-ingest: swipe via [knowledgeapp.Service.IngestFromSwipe] after SubmitSwipe;
// task completion via [task.Service.KnowledgeAutoIngest] on [task.Service.CompleteWithLiveActivity] (same hook as platform wiring).
func main() {
	ctx := context.Background()
	clock := timeadapter.System{}
	ids := &identity.Sequential{}

	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	costs := memory.NewCostStore()
	costCaps := memory.NewCostCapStore()
	checkpoints := memory.NewCheckpointStore()
	maybePool := memory.NewMaybePoolStore()
	knowledge := memory.NewKnowledgeStore()
	gateway := &gw.SimulationMockClaw{}

	knowSvc := &knowledgeapp.Service{
		Products:          products,
		Repo:              knowledge,
		Clock:             clock,
		AutoIngest:        true,
		UseFTSQuerySyntax: false, // memory substring search
	}

	productSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	autoSvc := &autopilot.Service{
		Products:   products,
		Ideas:      ideas,
		MaybePool:  maybePool,
		Research:   ai.ResearchStub{},
		Ideation:   ai.IdeationStub{},
		Clock:      clock,
		Identities: ids,
	}
	staticBudget := &budget.Static{Cap: 100, Costs: costs}
	taskSvc := &task.Service{
		Tasks:    tasks,
		Products: products,
		Ideas:    ideas,
		Gateway:  gateway,
		Budget:   staticBudget,
		Checkpt:  checkpoints,
		Clock:    clock,
		IDs:      ids,
		Ship:     shipping.PullRequestNoop{},
		KnowledgeAutoIngest: func(ctx context.Context, tt *domain.Task, source string, knowledgeSummary string) {
			_ = knowSvc.IngestFromTaskCompletion(ctx, tt, source, knowledgeSummary)
		},
	}
	convoySvc := &convoy.Service{
		Convoys:  convoys,
		Tasks:    tasks,
		Products: products,
		Gateway:  gateway,
		Budget:   staticBudget,
		Clock:    clock,
		IDs:      ids,
	}
	costSvc := &cost.Service{Costs: costs, Caps: costCaps, Budget: staticBudget, Clock: clock, IDs: ids}

	p, err := productSvc.Register(ctx, product.RegistrationInput{Name: "demo", WorkspaceID: "ws-default"})
	if err != nil {
		log.Fatal(err)
	}
	if err := autoSvc.RunResearch(ctx, p.ID); err != nil {
		log.Fatal(err)
	}
	if err := autoSvc.RunIdeation(ctx, p.ID); err != nil {
		log.Fatal(err)
	}
	list, err := ideas.ListByProduct(ctx, p.ID)
	if err != nil || len(list) == 0 {
		log.Fatal("expected ideas")
	}
	idea := list[0]
	if err := autoSvc.SubmitSwipe(ctx, idea.ID, domain.DecisionYes); err != nil {
		log.Fatal(err)
	}
	// Auto-ingest runs on HTTP swipe; mirror it here for the direct autopilot call.
	ideaAfterSwipe, err := ideas.ByID(ctx, idea.ID)
	if err != nil {
		log.Fatal(err)
	}
	if err := knowSvc.IngestFromSwipe(ctx, ideaAfterSwipe, domain.DecisionYes); err != nil {
		log.Fatal(err)
	}
	t, err := taskSvc.CreateFromApprovedIdea(ctx, idea.ID, "implement sample feature")
	if err != nil {
		log.Fatal(err)
	}
	if err := taskSvc.ApprovePlan(ctx, t.ID, ""); err != nil {
		log.Fatal(err)
	}
	if err := taskSvc.SetKanbanStatus(ctx, t.ID, domain.StatusAssigned, ""); err != nil {
		log.Fatal(err)
	}
	if err := taskSvc.Dispatch(ctx, t.ID, 5); err != nil {
		log.Fatal(err)
	}
	if err := costSvc.Record(ctx, p.ID, t.ID, 4.5, "llm", "", ""); err != nil {
		log.Fatal(err)
	}

	bID := domain.SubtaskID("builder-1")
	testerID := domain.SubtaskID("tester-1")
	conv, err := convoySvc.Create(ctx, convoy.CreateInput{
		ParentTaskID: t.ID,
		ProductID:    p.ID,
		Subtasks: []domain.Subtask{
			{ID: bID, AgentRole: "builder"},
			{ID: testerID, AgentRole: "tester", DependsOn: []domain.SubtaskID{bID}},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := convoySvc.DispatchReady(ctx, conv.ID, 0); err != nil {
		log.Fatal(err)
	}
	if err := convoySvc.CompleteSubtask(ctx, conv.ID, bID, t.ID); err != nil {
		log.Fatal(err)
	}
	if _, err := convoySvc.DispatchReady(ctx, conv.ID, 0); err != nil {
		log.Fatal(err)
	}
	cfinal, _ := convoys.ByID(ctx, conv.ID)

	// Same path as HTTP POST …/complete / webhooks: completion triggers optional knowledge auto-ingest.
	if err := taskSvc.CompleteWithLiveActivity(ctx, t.ID, "arms_demo",
		"Demo run finished: sample feature path exercised (stub gateway + convoy partial wave)."); err != nil {
		log.Fatal(err)
	}

	p2, _ := products.ByID(ctx, p.ID)
	t2, _ := tasks.ByID(ctx, t.ID)
	krows, _ := knowSvc.List(ctx, p.ID, 20)
	fmt.Printf("product stage=%s\n", p2.Stage.String())
	fmt.Printf("task status=%s ref=%s\n", t2.Status.String(), t2.ExternalRef)
	fmt.Printf("knowledge entries (swipe + CompleteWithLiveActivity)=%d\n", len(krows))
	fmt.Printf("convoy subtasks dispatched=%v/%v completed=%v/%v\n",
		cfinal.Subtasks[0].Dispatched, cfinal.Subtasks[1].Dispatched,
		cfinal.Subtasks[0].Completed, cfinal.Subtasks[1].Completed)
}
