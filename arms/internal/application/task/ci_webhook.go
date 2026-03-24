package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// ApplyCIWebhookOutcome applies a signed CI callback: move the board or mark done / failed.
// Targets are testing, review, done, and failed only; transitions must satisfy [domain.AllowedKanbanTransition].
// done uses [Service.CompleteWithLiveActivity] (same merge-queue side effects as other completion paths).
// All automation tiers may use this webhook (unlike agent-completion optional Kanban, which only advances for full_auto/semi_auto).
func (s *Service) ApplyCIWebhookOutcome(ctx context.Context, taskID domain.TaskID, nextBoardStatus, statusReason, source string, knowledgeSummary ...string) error {
	nextBoardStatus = strings.TrimSpace(nextBoardStatus)
	if nextBoardStatus == "" {
		return fmt.Errorf("%w: next_board_status is required", domain.ErrInvalidInput)
	}
	to, err := domain.ParseTaskStatus(nextBoardStatus)
	if err != nil {
		return err
	}
	switch to {
	case domain.StatusTesting, domain.StatusReview, domain.StatusDone, domain.StatusFailed:
	default:
		return fmt.Errorf("%w: ci next_board_status must be testing, review, done, or failed", domain.ErrInvalidInput)
	}
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	reason := strings.TrimSpace(statusReason)
	if to == domain.StatusFailed && reason == "" {
		reason = "CI reported failure"
	}
	if to == domain.StatusDone {
		return s.CompleteWithLiveActivity(ctx, taskID, source, knowledgeSummary...)
	}
	move := func() error {
		return s.SetKanbanStatus(ctx, taskID, to, reason)
	}
	if s.Gate != nil {
		return s.Gate.WithLock(t.ProductID, move)
	}
	return move()
}
