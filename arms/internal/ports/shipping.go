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

// CreatePullRequestResult is returned when a PR is created (Number may be 0 if the backend cannot determine it).
type CreatePullRequestResult struct {
	HTMLURL string
	Number  int
}

// PullRequestPublisher opens a PR after task completion; noop until forge token is configured.
type PullRequestPublisher interface {
	CreatePullRequest(ctx context.Context, in CreatePullRequestInput) (CreatePullRequestResult, error)
}

// PullRequestMerger merges an existing pull request via GitHub REST (or compatible API).
type PullRequestMerger interface {
	MergePullRequest(ctx context.Context, owner, repo string, prNumber int, mergeMethod string) (domain.MergeShipResult, error)
}
