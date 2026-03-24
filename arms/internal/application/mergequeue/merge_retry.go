package mergequeue

import (
	"context"
	"errors"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

const (
	mergeShipMaxAttempts    = 3
	mergeShipRetryBaseDelay = 200 * time.Millisecond
)

func mergePullRequestWithRetry(ctx context.Context, merger ports.PullRequestMerger, owner, repo string, prNumber int, method string) (domain.MergeShipResult, error) {
	var lastRes domain.MergeShipResult
	var lastErr error
	for attempt := 0; attempt < mergeShipMaxAttempts; attempt++ {
		if attempt > 0 {
			d := mergeShipRetryBaseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return domain.MergeShipResult{}, ctx.Err()
			case <-time.After(d):
			}
		}
		res, err := merger.MergePullRequest(ctx, owner, repo, prNumber, method)
		if err == nil {
			return res, nil
		}
		lastRes, lastErr = res, err
		if ctx.Err() != nil {
			return lastRes, ctx.Err()
		}
		if !shouldRetryMergeShip(err, res) {
			return res, err
		}
	}
	return lastRes, lastErr
}

func shouldRetryMergeShip(err error, res domain.MergeShipResult) bool {
	if errors.Is(err, domain.ErrMergeConflict) || res.State == domain.MergeShipConflict {
		return false
	}
	if errors.Is(err, domain.ErrShippingNonRetryable) {
		return false
	}
	return errors.Is(err, domain.ErrShipping)
}
