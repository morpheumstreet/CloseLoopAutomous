package mergequeue

import (
	"context"
	"fmt"
	"testing"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

type flakyMerger struct {
	failRemaining int
	calls         int
}

func (f *flakyMerger) MergePullRequest(_ context.Context, _, _ string, _ int, _ string) (domain.MergeShipResult, error) {
	f.calls++
	if f.failRemaining > 0 {
		f.failRemaining--
		return domain.MergeShipResult{State: domain.MergeShipFailed}, fmt.Errorf("%w: transient", domain.ErrShipping)
	}
	return domain.MergeShipResult{State: domain.MergeShipMerged, MergedSHA: "abc123"}, nil
}

func TestMergePullRequestWithRetry(t *testing.T) {
	ctx := context.Background()
	f := &flakyMerger{failRemaining: 2}
	res, err := mergePullRequestWithRetry(ctx, f, "o", "r", 1, "merge")
	if err != nil {
		t.Fatal(err)
	}
	if res.State != domain.MergeShipMerged || f.calls != 3 {
		t.Fatalf("state=%s calls=%d", res.State, f.calls)
	}
}

func TestMergePullRequestWithRetryNoRetryOnConflict(t *testing.T) {
	ctx := context.Background()
	m := &conflictMerger{}
	_, err := mergePullRequestWithRetry(ctx, m, "o", "r", 1, "merge")
	if err == nil {
		t.Fatal("want error")
	}
	if m.calls != 1 {
		t.Fatalf("calls=%d", m.calls)
	}
}

type conflictMerger struct{ calls int }

func (c *conflictMerger) MergePullRequest(_ context.Context, _, _ string, _ int, _ string) (domain.MergeShipResult, error) {
	c.calls++
	return domain.MergeShipResult{State: domain.MergeShipConflict}, fmt.Errorf("%w: conflict", domain.ErrMergeConflict)
}
