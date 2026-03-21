package ports

import (
	"context"

	"github.com/closeloopautomous/arms/internal/domain"
)

// WorktreeMerger runs git merge in a local clone (product.repo_clone_path).
type WorktreeMerger interface {
	MergeBranches(ctx context.Context, in WorktreeMergeInput) (domain.MergeShipResult, error)
}

// WorktreeMergeInput describes a local merge of head into base inside repoRoot.
type WorktreeMergeInput struct {
	GitBin     string
	RepoRoot   string
	BaseBranch string
	HeadBranch string
}
