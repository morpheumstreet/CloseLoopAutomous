package task

import (
	"context"
	"errors"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

const (
	prCreateMaxAttempts    = 3
	prCreateRetryBaseDelay = 200 * time.Millisecond
)

func createPullRequestWithRetry(ctx context.Context, ship ports.PullRequestPublisher, in ports.CreatePullRequestInput) (ports.CreatePullRequestResult, error) {
	var lastErr error
	for attempt := 0; attempt < prCreateMaxAttempts; attempt++ {
		if attempt > 0 {
			d := prCreateRetryBaseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ports.CreatePullRequestResult{}, ctx.Err()
			case <-time.After(d):
			}
		}
		cre, err := ship.CreatePullRequest(ctx, in)
		if err == nil {
			return cre, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return ports.CreatePullRequestResult{}, ctx.Err()
		}
		if !shouldRetryPullRequestCreate(err) {
			return ports.CreatePullRequestResult{}, err
		}
	}
	return ports.CreatePullRequestResult{}, lastErr
}

func shouldRetryPullRequestCreate(err error) bool {
	return err != nil &&
		errors.Is(err, domain.ErrShipping) &&
		!errors.Is(err, domain.ErrShippingNonRetryable)
}
