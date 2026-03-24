package ports

import (
	"context"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
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

// MergeQueueShipper completes the FIFO merge queue head for a task (optional automation hook).
type MergeQueueShipper interface {
	Complete(ctx context.Context, taskID domain.TaskID, skipRealMerge bool) error
	// CompleteIfPolicyAllowsAuto performs a real merge only when tier + merge gates allow unattended ship.
	CompleteIfPolicyAllowsAuto(ctx context.Context, taskID domain.TaskID) error
}
