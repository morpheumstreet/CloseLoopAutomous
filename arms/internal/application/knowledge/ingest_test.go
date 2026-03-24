package knowledge

import (
	"context"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/memory"
	timeadapter "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/time"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestIngestFromSwipe_disabledNoop(t *testing.T) {
	ctx := context.Background()
	products := memory.NewProductStore()
	repo := memory.NewKnowledgeStore()
	s := &Service{Products: products, Repo: repo, Clock: timeadapter.System{}, AutoIngest: false}
	idea := &domain.Idea{ID: "i1", ProductID: "p1", Title: "T", Description: "D"}
	if err := s.IngestFromSwipe(ctx, idea, domain.DecisionYes); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListByProduct(ctx, "p1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no rows, got %d", len(list))
	}
}

func TestIngestFromSwipe_createsEntry(t *testing.T) {
	ctx := context.Background()
	products := memory.NewProductStore()
	_ = products.Save(ctx, &domain.Product{ID: "p1", Name: "P", WorkspaceID: "w", UpdatedAt: time.Now().UTC()})
	repo := memory.NewKnowledgeStore()
	s := &Service{Products: products, Repo: repo, Clock: timeadapter.System{}, AutoIngest: true}
	idea := &domain.Idea{ID: "i1", ProductID: "p1", Title: "Auth", Description: "OAuth2 flow", Decision: domain.DecisionYes}
	if err := s.IngestFromSwipe(ctx, idea, domain.DecisionYes); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListByProduct(ctx, "p1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("len=%d", len(list))
	}
	if list[0].TaskID != "" {
		t.Fatalf("task id: %q", list[0].TaskID)
	}
}

func TestTruncateRunes(t *testing.T) {
	s := string([]rune{'a', 'b', 'c', 'д'}) // 4 runes
	out := truncateRunes(s, 3)
	if out == s || len([]rune(out)) < 3 {
		t.Fatalf("got %q", out)
	}
}
