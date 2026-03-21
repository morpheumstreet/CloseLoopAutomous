package task

import (
	"context"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
)

// reviewColumnAutoPRTiers: auto-open PR when landing in review from execution columns (full_auto only).
func reviewColumnAutoPRTiers(tier domain.AutomationTier) bool {
	return tier == domain.TierFullAuto
}

// mergePrepAutoPRTiers: ensure PR exists before unattended merge-queue ship.
func mergePrepAutoPRTiers(tier domain.AutomationTier) bool {
	return tier == domain.TierFullAuto || tier == domain.TierSemiAuto
}

// openPullRequestWhenEligible opens a PR when the task has head branch + no URL yet and tier passes tierEligible.
func (s *Service) openPullRequestWhenEligible(ctx context.Context, taskID domain.TaskID, tier domain.AutomationTier, tierEligible func(domain.AutomationTier) bool) error {
	if s.Ship == nil || !tierEligible(tier) {
		return nil
	}
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	head := strings.TrimSpace(t.PullRequestHeadBranch)
	if head == "" || strings.TrimSpace(t.PullRequestURL) != "" {
		return nil
	}
	_, _, err = s.OpenPullRequest(ctx, taskID, head, "", "")
	return err
}
