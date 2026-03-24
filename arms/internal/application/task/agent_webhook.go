package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// ApplyAgentWebhookOutcome handles a verified agent-completion webhook: optional Kanban advance for
// full_auto / semi_auto (testing / review) or mark done (default). source is stored on task_completed (done path only).
// knowledgeSummary (optional) is forwarded to [Service.CompleteWithLiveActivity] when the task ends in done.
func (s *Service) ApplyAgentWebhookOutcome(ctx context.Context, taskID domain.TaskID, nextBoardStatus, source string, knowledgeSummary ...string) error {
	nextBoardStatus = strings.TrimSpace(nextBoardStatus)
	if nextBoardStatus == "" || strings.EqualFold(nextBoardStatus, string(domain.StatusDone)) {
		return s.CompleteWithLiveActivity(ctx, taskID, source, knowledgeSummary...)
	}
	switch strings.ToLower(nextBoardStatus) {
	case "testing", "review":
	default:
		return fmt.Errorf("%w: next_board_status must be testing, review, or done (or omit for done)", domain.ErrInvalidInput)
	}
	to, err := domain.ParseTaskStatus(nextBoardStatus)
	if err != nil {
		return fmt.Errorf("%w: next_board_status", domain.ErrInvalidInput)
	}
	t, p, err := s.taskAndActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	if p.AutomationTier != domain.TierFullAuto && p.AutomationTier != domain.TierSemiAuto {
		return s.CompleteWithLiveActivity(ctx, taskID, source, knowledgeSummary...)
	}
	if !domain.AllowedKanbanTransition(t.Status, to) {
		return fmt.Errorf("%w: webhook cannot move %s -> %s", domain.ErrInvalidTransition, t.Status, to)
	}
	move := func() error {
		return s.SetKanbanStatus(ctx, taskID, to, "")
	}
	if s.Gate != nil {
		return s.Gate.WithLock(t.ProductID, move)
	}
	return move()
}
