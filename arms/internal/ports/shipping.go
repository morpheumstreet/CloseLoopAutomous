package ports

import (
	"context"

	"github.com/closeloopautomous/arms/internal/domain"
)

// CreatePullRequestInput is a minimal contract for post-ship automation (GitHub or other forge).
type CreatePullRequestInput struct {
	ProductID domain.ProductID
	TaskID    domain.TaskID
	Title     string
	Body      string
	Branch    string
}

// PullRequestPublisher opens a PR after task completion; real adapters plug in later.
type PullRequestPublisher interface {
	CreatePullRequest(ctx context.Context, in CreatePullRequestInput) (htmlURL string, err error)
}
