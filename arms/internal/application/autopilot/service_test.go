package autopilot

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/ai"
	"github.com/closeloopautomous/arms/internal/adapters/identity"
	"github.com/closeloopautomous/arms/internal/adapters/memory"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/application/product"
	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type captureIdeation struct {
	lastSummary string
}

func (c *captureIdeation) GenerateIdeas(_ context.Context, _ domain.Product, researchSummary string) ([]domain.IdeaDraft, error) {
	c.lastSummary = researchSummary
	return []domain.IdeaDraft{{Title: "captured", Category: "feature"}}, nil
}

var _ ports.IdeationPort = (*captureIdeation)(nil)

func TestRunIdeationLinksLatestResearchCycle(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	rc := memory.NewResearchCycleStore()
	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	svc := &Service{
		Products:       products,
		Ideas:          ideas,
		ResearchCycles: rc,
		Research:       ai.ResearchStub{},
		Ideation:       ai.IdeationStub{},
		Clock:          clock,
		Identities:     ids,
	}
	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	_ = svc.RunResearch(ctx, p.ID)
	_ = svc.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	if len(list) != 1 {
		t.Fatalf("ideas: %d", len(list))
	}
	if list[0].ResearchCycleID == "" {
		t.Fatal("expected research_cycle_id on new idea")
	}
}

func TestSubmitSwipeMaybePoolAndPromote(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	pool := memory.NewMaybePoolStore()
	swipes := memory.NewSwipeHistoryStore()
	svc := &Service{
		Products:   products,
		Ideas:      ideas,
		MaybePool:  pool,
		Swipes:     swipes,
		Research:   ai.ResearchStub{},
		Ideation:   ai.IdeationStub{},
		Clock:      clock,
		Identities: ids,
	}
	prodSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	p, _ := prodSvc.Register(ctx, product.RegistrationInput{Name: "p", WorkspaceID: "w"})
	_ = svc.RunResearch(ctx, p.ID)
	_ = svc.RunIdeation(ctx, p.ID)
	list, _ := ideas.ListByProduct(ctx, p.ID)
	iid := list[0].ID

	if err := svc.SubmitSwipe(ctx, iid, domain.DecisionMaybe); err != nil {
		t.Fatal(err)
	}
	h1, _ := swipes.ListByProduct(ctx, p.ID, 10)
	if len(h1) != 1 || h1[0].Decision != "maybe" || h1[0].IdeaID != iid {
		t.Fatalf("swipe history after maybe: %#v", h1)
	}
	idsInPool, _ := pool.ListIdeaIDsByProduct(ctx, p.ID)
	if len(idsInPool) != 1 || idsInPool[0] != iid {
		t.Fatalf("pool: %#v", idsInPool)
	}
	p2, _ := products.ByID(ctx, p.ID)
	if p2.PreferenceModelJSON == "" {
		t.Fatal("expected preference_model_json append")
	}

	if err := svc.PromoteMaybe(ctx, iid); err != nil {
		t.Fatal(err)
	}
	h2, _ := swipes.ListByProduct(ctx, p.ID, 10)
	if len(h2) != 2 || h2[0].Decision != "yes" || h2[1].Decision != "maybe" {
		t.Fatalf("swipe history after promote: %#v", h2)
	}
	idea, _ := ideas.ByID(ctx, iid)
	if idea.Decision != domain.DecisionYes {
		t.Fatalf("want yes got %v", idea.Decision)
	}
	idsInPool, _ = pool.ListIdeaIDsByProduct(ctx, p.ID)
	if len(idsInPool) != 0 {
		t.Fatalf("pool should empty: %#v", idsInPool)
	}
	p3, _ := products.ByID(ctx, p.ID)
	if p3.Stage != domain.StagePlanning {
		t.Fatalf("stage %s", p3.Stage)
	}
}

func TestTickProductMatchesTickScheduled(t *testing.T) {
	ctx := context.Background()
	t0 := time.Unix(1800000000, 0).UTC()
	clock := timeadapter.Fixed{T: t0}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	svc := &Service{
		Products:   products,
		Ideas:      ideas,
		MaybePool:  memory.NewMaybePoolStore(),
		Research:   ai.ResearchStub{},
		Ideation:   ai.IdeationStub{},
		Clock:      clock,
		Identities: ids,
	}
	p := &domain.Product{
		ID:                 "prod-1",
		Name:               "x",
		Stage:              domain.StageResearch,
		WorkspaceID:        "w",
		ResearchCadenceSec: 1,
		IdeationCadenceSec: 0,
		UpdatedAt:          t0,
	}
	_ = products.Save(ctx, p)

	if err := svc.TickProduct(ctx, p.ID, t0.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	p1, _ := products.ByID(ctx, p.ID)
	if p1.Stage != domain.StageIdeation {
		t.Fatalf("after auto research want ideation got %s", p1.Stage)
	}
}

func TestNextAutopilotEnqueueDelay(t *testing.T) {
	ctx := context.Background()
	t0 := time.Unix(1800000000, 0).UTC()
	clock := timeadapter.Fixed{T: t0}
	svc := &Service{
		Products: memory.NewProductStore(),
		Research: ai.ResearchStub{},
		Ideation: ai.IdeationStub{},
		Clock:    clock,
	}
	p := &domain.Product{
		ID:                 "p1",
		Name:               "x",
		Stage:              domain.StageResearch,
		WorkspaceID:        "w",
		ResearchCadenceSec: 60,
		LastAutoResearchAt: t0,
		UpdatedAt:          t0,
	}
	_ = svc.Products.Save(ctx, p)

	d, keep, err := svc.NextAutopilotEnqueueDelay(ctx, p.ID, t0.Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !keep {
		t.Fatal("expected keep")
	}
	if d != 30*time.Second {
		t.Fatalf("delay want 30s got %v", d)
	}

	d2, keep2, err := svc.NextAutopilotEnqueueDelay(ctx, p.ID, t0.Add(60*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !keep2 || d2 != 0 {
		t.Fatalf("at due want delay 0 keep true got %v %v", d2, keep2)
	}

	p2, _ := svc.Products.ByID(ctx, p.ID)
	p2.Stage = domain.StageSwipe
	_ = svc.Products.Save(ctx, p2)
	_, keep3, err := svc.NextAutopilotEnqueueDelay(ctx, p.ID, t0)
	if err != nil {
		t.Fatal(err)
	}
	if keep3 {
		t.Fatal("swipe stage should not keep chain")
	}
}

func TestTickScheduledCadence(t *testing.T) {
	ctx := context.Background()
	t0 := time.Unix(1800000000, 0).UTC()
	clock := timeadapter.Fixed{T: t0}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	svc := &Service{
		Products:   products,
		Ideas:      ideas,
		MaybePool:  memory.NewMaybePoolStore(),
		Research:   ai.ResearchStub{},
		Ideation:   ai.IdeationStub{},
		Clock:      clock,
		Identities: ids,
	}
	p := &domain.Product{
		ID:                 "prod-1",
		Name:               "x",
		Stage:              domain.StageResearch,
		WorkspaceID:        "w",
		ResearchCadenceSec: 1,
		IdeationCadenceSec: 0,
		UpdatedAt:          t0,
	}
	_ = products.Save(ctx, p)

	if err := svc.TickScheduled(ctx, t0.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	p1, _ := products.ByID(ctx, p.ID)
	if p1.Stage != domain.StageIdeation {
		t.Fatalf("after auto research want ideation got %s", p1.Stage)
	}
	if p1.LastAutoResearchAt != t0.Add(2*time.Second) {
		t.Fatalf("last research: %v", p1.LastAutoResearchAt)
	}

	p1.IdeationCadenceSec = 1
	if err := products.Save(ctx, p1); err != nil {
		t.Fatal(err)
	}
	if err := svc.TickScheduled(ctx, t0.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	p2, _ := products.ByID(ctx, p.ID)
	if p2.Stage != domain.StageSwipe {
		t.Fatalf("after auto ideation want swipe got %s", p2.Stage)
	}
	if p2.LastAutoIdeationAt != t0.Add(4*time.Second) {
		t.Fatalf("last ideation: %v", p2.LastAutoIdeationAt)
	}
}

func TestTickScheduledSkippedWhenScheduleDisabled(t *testing.T) {
	ctx := context.Background()
	t0 := time.Unix(1800000000, 0).UTC()
	clock := timeadapter.Fixed{T: t0}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	sched := memory.NewProductScheduleStore()
	svc := &Service{
		Products:  products,
		Ideas:     ideas,
		MaybePool: memory.NewMaybePoolStore(),
		Schedules: sched,
		Research:  ai.ResearchStub{},
		Ideation:  ai.IdeationStub{},
		Clock:     clock,
		Identities: ids,
	}
	p := &domain.Product{
		ID:                 "prod-1",
		Name:               "x",
		Stage:              domain.StageResearch,
		WorkspaceID:        "w",
		ResearchCadenceSec: 1,
		UpdatedAt:          t0,
	}
	_ = products.Save(ctx, p)
	_ = sched.Upsert(ctx, &domain.ProductSchedule{ProductID: p.ID, Enabled: false, SpecJSON: "{}", UpdatedAt: t0})

	if err := svc.TickScheduled(ctx, t0.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	p1, _ := products.ByID(ctx, p.ID)
	if p1.Stage != domain.StageResearch {
		t.Fatalf("schedule disabled: want still research got %s", p1.Stage)
	}
}

func TestRecomputePreferenceModelFromSwipes(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	swipes := memory.NewSwipeHistoryStore()
	pref := memory.NewPreferenceModelStore()
	svc := &Service{
		Products:   products,
		Ideas:      ideas,
		Swipes:     swipes,
		PrefModel:  pref,
		Research:   ai.ResearchStub{},
		Ideation:   ai.IdeationStub{},
		Clock:      clock,
		Identities: ids,
	}
	p := &domain.Product{ID: "p1", Name: "n", Stage: domain.StagePlanning, WorkspaceID: "w", UpdatedAt: clock.Now()}
	_ = products.Save(ctx, p)
	_ = swipes.Append(ctx, domain.IdeaID("i1"), p.ID, "yes", clock.Now())
	_ = swipes.Append(ctx, domain.IdeaID("i2"), p.ID, "maybe", clock.Now())

	js, err := svc.RecomputePreferenceModelFromSwipes(ctx, p.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if js == "" {
		t.Fatal("empty json")
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(js), &root); err != nil {
		t.Fatal(err)
	}
	if int(root["version"].(float64)) != preferenceAggregateVersion {
		t.Fatalf("version: %#v", root["version"])
	}
	if root["source"] != "preference_aggregate_v1" {
		t.Fatalf("source: %v", root["source"])
	}
	mj, _, ok, _ := pref.Get(ctx, p.ID)
	if !ok {
		t.Fatal("want preference row")
	}
	if mj != js {
		t.Fatalf("stored != returned")
	}
}

func TestRecomputePreferenceModelIncludesFeedback(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	fb := memory.NewProductFeedbackStore()
	pref := memory.NewPreferenceModelStore()
	svc := &Service{
		Products:  products,
		Ideas:     memory.NewIdeaStore(),
		Swipes:    memory.NewSwipeHistoryStore(),
		Feedback:  fb,
		PrefModel: pref,
		Clock:     clock,
		Identities: ids,
	}
	p := &domain.Product{ID: "p1", Name: "n", Stage: domain.StagePlanning, WorkspaceID: "w", UpdatedAt: clock.Now()}
	_ = products.Save(ctx, p)
	_ = fb.Append(ctx, &domain.ProductFeedback{
		ID: "f1", ProductID: p.ID, Source: "web", Content: "love it",
		Sentiment: "positive", Category: "ux", CreatedAt: clock.Now(),
	})
	js, err := svc.RecomputePreferenceModelFromSwipes(ctx, p.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(js), &root); err != nil {
		t.Fatal(err)
	}
	fblock, ok := root["feedback"].(map[string]any)
	if !ok {
		t.Fatal("missing feedback block")
	}
	if int(fblock["sample_size"].(float64)) != 1 {
		t.Fatalf("sample_size %v", fblock["sample_size"])
	}
}

func TestRunIdeationAppendsPreferenceHints(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ids := &identity.Sequential{}
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	pref := memory.NewPreferenceModelStore()
	cap := &captureIdeation{}
	svc := &Service{
		Products:  products,
		Ideas:     ideas,
		PrefModel: pref,
		Ideation:  cap,
		Clock:     clock,
		Identities: ids,
	}
	p := &domain.Product{
		ID: "p1", Name: "n", Stage: domain.StageIdeation, WorkspaceID: "w",
		ResearchSummary: "BASE_SUMMARY", UpdatedAt: clock.Now(),
	}
	_ = products.Save(ctx, p)
	model := `{"version":1,"source":"preference_aggregate_v1","generated_at":"2020-01-01T00:00:00Z","category_affinity":{"ux":2.5,"feature":-1}}`
	_ = pref.Upsert(ctx, p.ID, model, clock.Now())
	if err := svc.RunIdeation(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cap.lastSummary, "BASE_SUMMARY") || !strings.Contains(cap.lastSummary, "Operator-derived preference") {
		t.Fatalf("summary missing expected sections: %q", cap.lastSummary)
	}
	if !strings.Contains(cap.lastSummary, "ux") {
		t.Fatalf("expected category hint in summary: %q", cap.lastSummary)
	}
}
