package autopilot

import (
	"context"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/memory"
	timeadapter "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/time"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestPatchIdeaMetadata(t *testing.T) {
	ctx := context.Background()
	clock := timeadapter.Fixed{T: time.Unix(1700000000, 0)}
	ideas := memory.NewIdeaStore()
	svc := &Service{Ideas: ideas, Clock: clock}
	iid := domain.IdeaID("i1")
	_ = ideas.Save(ctx, &domain.Idea{
		ID: iid, ProductID: "p1", Title: "t", Description: "d",
		CreatedAt: clock.Now(), Status: domain.IdeaStatusPending,
	})
	cat := "security"
	tags := []string{"a", "b"}
	err := svc.PatchIdeaMetadata(ctx, iid, IdeaMetadataPatch{
		Category: &cat,
		Tags:     &tags,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := ideas.ByID(ctx, iid)
	if got.Category != "security" || len(got.Tags) != 2 {
		t.Fatalf("patch: %+v", got)
	}
}
