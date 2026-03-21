package shipping

import (
	"context"

	"github.com/closeloopautomous/arms/internal/ports"
)

// PullRequestNoop returns success with an empty URL until a real forge adapter is wired.
type PullRequestNoop struct{}

var _ ports.PullRequestPublisher = (*PullRequestNoop)(nil)

func (PullRequestNoop) CreatePullRequest(_ context.Context, in ports.CreatePullRequestInput) (string, error) {
	_ = in
	return "", nil
}
