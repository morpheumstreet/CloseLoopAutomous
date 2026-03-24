package shipping

import (
	"context"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// PullRequestNoop returns success with an empty URL until a real forge adapter is wired.
type PullRequestNoop struct{}

var _ ports.PullRequestPublisher = (*PullRequestNoop)(nil)

func (PullRequestNoop) CreatePullRequest(_ context.Context, in ports.CreatePullRequestInput) (ports.CreatePullRequestResult, error) {
	_ = in
	return ports.CreatePullRequestResult{}, nil
}
