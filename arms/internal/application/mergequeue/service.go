package mergequeue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/livefeed"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// Service completes the FIFO merge queue with optional real GitHub or local git merges.
type Service struct {
	Backend       string // noop | github | local (empty = noop)
	DefaultMethod string // merge | squash | rebase
	LeaseOwner    string
	LeaseTTL      time.Duration
	Queue         ports.WorkspaceMergeQueueRepository
	Tasks         ports.TaskRepository
	Products      ports.ProductRepository
	PRMerger      ports.PullRequestMerger
	Gates         ports.PullRequestMergeGateChecker // optional; GitHub gate checks for semi_auto auto-ship
	Worktree      ports.WorktreeMerger
	Events        ports.LiveActivityPublisher
	Clock         ports.Clock
	GitBin        string
}

// New returns nil when queue is nil.
func New(cfg MergeConfig, q ports.WorkspaceMergeQueueRepository, tasks ports.TaskRepository, products ports.ProductRepository, pr ports.PullRequestMerger, gates ports.PullRequestMergeGateChecker, wt ports.WorktreeMerger, events ports.LiveActivityPublisher, clock ports.Clock) *Service {
	if q == nil {
		return nil
	}
	owner := strings.TrimSpace(cfg.LeaseOwner)
	if owner == "" {
		owner, _ = os.Hostname()
		if owner == "" {
			owner = "arms"
		}
	}
	ttl := time.Duration(cfg.LeaseTTLSec) * time.Second
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	return &Service{
		Backend:       strings.ToLower(strings.TrimSpace(cfg.Backend)),
		DefaultMethod: domain.NormalizeMergeMethod(cfg.MergeMethod),
		LeaseOwner:    owner,
		LeaseTTL:      ttl,
		Queue:         q,
		Tasks:         tasks,
		Products:      products,
		PRMerger:      pr,
		Gates:         gates,
		Worktree:      wt,
		Events:        events,
		Clock:         clock,
		GitBin:        cfg.GitBin,
	}
}

// MergeConfig is a small config slice (from [config.Config]).
type MergeConfig struct {
	Backend      string
	MergeMethod  string
	LeaseOwner   string
	LeaseTTLSec  int
	GitBin       string
}

// Complete finishes the merge-queue head for taskID. When skipRealMerge, records skipped ship and advances queue.
// Manual API calls do not enforce merge gates (operator override).
func (s *Service) Complete(ctx context.Context, taskID domain.TaskID, skipRealMerge bool) error {
	return s.complete(ctx, taskID, skipRealMerge, false)
}

// CompleteIfPolicyAllowsAuto runs an unattended ship when the product tier allows it and merge gates pass.
func (s *Service) CompleteIfPolicyAllowsAuto(ctx context.Context, taskID domain.TaskID) error {
	return s.complete(ctx, taskID, false, true)
}

func (s *Service) complete(ctx context.Context, taskID domain.TaskID, skipRealMerge, autoOnly bool) error {
	if s == nil {
		return domain.ErrNotFound
	}
	b := strings.TrimSpace(s.Backend)
	if b == "" || b == "noop" {
		return s.Queue.CompletePendingForTask(ctx, taskID)
	}

	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	p, err := s.Products.ByID(ctx, t.ProductID)
	if err != nil {
		return err
	}
	pol := domain.ParseMergePolicy(p.MergePolicyJSON)
	if o := strings.TrimSpace(pol.MergeBackendOverride); o != "" {
		b = o
	}

	if autoOnly {
		switch p.AutomationTier {
		case domain.TierSupervised:
			return nil
		case domain.TierSemiAuto:
			// gated path below before lease
		case domain.TierFullAuto:
			// proceed without gate enforcement
		default:
			return nil
		}
	}

	if autoOnly && p.AutomationTier == domain.TierSemiAuto && !skipRealMerge {
		if err := s.enforceAutoMergeGates(ctx, t, p, pol, b); err != nil {
			if errors.Is(err, domain.ErrMergeGatesNotMet) {
				return nil
			}
			return err
		}
	}

	exp := s.Clock.Now().Add(s.LeaseTTL)
	rowID, err := s.Queue.ReserveHeadForShip(ctx, taskID, s.LeaseOwner, exp)
	if err != nil {
		return err
	}

	var res domain.MergeShipResult
	var shipErr error
	if skipRealMerge {
		res = domain.MergeShipResult{State: domain.MergeShipSkipped}
	} else {
		switch b {
		case "github":
			res, shipErr = s.mergeGitHub(ctx, t, p, pol)
		case "local":
			res, shipErr = s.mergeLocal(ctx, t, p)
		default:
			res = domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: fmt.Sprintf("unknown merge backend %q", b)}
			shipErr = fmt.Errorf("%w: merge backend", domain.ErrInvalidInput)
		}
	}

	ev := ports.LiveActivityEvent{
		Type:      "merge_ship_completed",
		Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
		ProductID: string(t.ProductID),
		TaskID:    string(taskID),
		Data: map[string]any{
			"merge_queue_row_id": rowID,
			"state":              string(res.State),
			"merged_sha":         res.MergedSHA,
			"error":              res.ErrorMessage,
			"conflict_files":     res.ConflictFiles,
		},
	}

	finishErr := s.finishShipAndPublish(ctx, rowID, res, shipErr, ev)
	if finishErr != nil {
		return finishErr
	}

	if res.State == domain.MergeShipConflict || errors.Is(shipErr, domain.ErrMergeConflict) {
		return domain.ErrMergeConflict
	}
	if shipErr != nil {
		return fmt.Errorf("%w: %v", domain.ErrShipping, shipErr)
	}
	if res.State == domain.MergeShipFailed {
		msg := strings.TrimSpace(res.ErrorMessage)
		if msg == "" {
			msg = "merge failed"
		}
		return fmt.Errorf("%w: %s", domain.ErrShipping, msg)
	}
	return nil
}

func (s *Service) enforceAutoMergeGates(ctx context.Context, t *domain.Task, p *domain.Product, pol domain.MergePolicy, backend string) error {
	g := domain.EffectiveMergeExecutionGates(p, pol)
	if !g.RequireApprovedReview && !g.RequireCleanMergeable {
		return nil
	}
	if strings.ToLower(strings.TrimSpace(backend)) != "github" {
		return nil
	}
	if s.Gates == nil || t.PullRequestNumber <= 0 {
		return domain.ErrMergeGatesNotMet
	}
	owner, repo, err := domain.ParseGitHubLikeOwnerRepo(p.RepoURL)
	if err != nil {
		return err
	}
	return s.Gates.CheckMergeGates(ctx, owner, repo, t.PullRequestNumber, g)
}

func (s *Service) finishShipAndPublish(ctx context.Context, rowID int64, res domain.MergeShipResult, shipErr error, ev ports.LiveActivityEvent) error {
	useAtomic := s.Events != nil
	if useAtomic {
		if _, ok := s.Events.(*livefeed.OutboxPublisher); ok {
			if ox, ok := s.Queue.(ports.MergeShipOutboxFinisher); ok {
				return ox.FinishShipWithOutbox(ctx, rowID, s.LeaseOwner, res, shipErr, ev)
			}
		}
	}
	err := s.Queue.FinishShip(ctx, rowID, s.LeaseOwner, res, shipErr)
	if err != nil {
		return err
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ev)
	}
	return nil
}

func (s *Service) mergeGitHub(ctx context.Context, t *domain.Task, p *domain.Product, pol domain.MergePolicy) (domain.MergeShipResult, error) {
	if s.PRMerger == nil {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: "GitHub merger not configured (token?)"}, fmt.Errorf("%w: merger", domain.ErrShipping)
	}
	if t.PullRequestNumber <= 0 {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: "task has no pull_request_number; open a PR first"}, fmt.Errorf("%w: pr number", domain.ErrInvalidInput)
	}
	owner, repo, err := domain.ParseGitHubLikeOwnerRepo(p.RepoURL)
	if err != nil {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: err.Error()}, err
	}
	method := domain.NormalizeMergeMethod(s.DefaultMethod)
	if strings.TrimSpace(pol.MergeMethod) != "" {
		method = domain.NormalizeMergeMethod(pol.MergeMethod)
	}
	return mergePullRequestWithRetry(ctx, s.PRMerger, owner, repo, t.PullRequestNumber, method)
}

func (s *Service) mergeLocal(ctx context.Context, t *domain.Task, p *domain.Product) (domain.MergeShipResult, error) {
	if s.Worktree == nil {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: "local git merger not wired"}, fmt.Errorf("%w: worktree merger", domain.ErrShipping)
	}
	root := strings.TrimSpace(p.RepoClonePath)
	if root == "" {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: "product.repo_clone_path required for local merge"}, fmt.Errorf("%w: repo_clone_path", domain.ErrInvalidInput)
	}
	base := strings.TrimSpace(p.RepoBranch)
	if base == "" {
		base = "main"
	}
	head := strings.TrimSpace(t.PullRequestHeadBranch)
	if head == "" {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: "task.pull_request_head_branch required (set when opening PR)"}, fmt.Errorf("%w: head branch", domain.ErrInvalidInput)
	}
	return s.Worktree.MergeBranches(ctx, ports.WorktreeMergeInput{
		GitBin:     s.GitBin,
		RepoRoot:   root,
		BaseBranch: base,
		HeadBranch: head,
	})
}
