package ports

import (
	"context"

	"github.com/closeloopautomous/arms/internal/domain"
)

// CreatePullRequestInput is the contract for opening a PR on GitHub (or compatible API).
type CreatePullRequestInput struct {
	ProductID  domain.ProductID
	TaskID     domain.TaskID
	Owner      string
	Repo       string
	Title      string
	Body       string
	HeadBranch string
	BaseBranch string
}

// PullRequestPublisher opens a PR after task completion; noop until forge token is configured.
type PullRequestPublisher interface {
	CreatePullRequest(ctx context.Context, in CreatePullRequestInput) (htmlURL string, err error)
}
