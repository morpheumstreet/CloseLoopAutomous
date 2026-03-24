package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/task"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// StallAutoNudgeHandler runs [task.Service.RunAutoStallNudgeSweep] from Asynq.
type StallAutoNudgeHandler struct {
	Task     *task.Service
	Products ports.ProductRepository
}

// NewStallAutoNudgeHandler returns nil if task or products is nil.
func NewStallAutoNudgeHandler(taskSvc *task.Service, products ports.ProductRepository) *StallAutoNudgeHandler {
	if taskSvc == nil || products == nil {
		return nil
	}
	return &StallAutoNudgeHandler{Task: taskSvc, Products: products}
}

// Handle implements the Asynq task arms:stall_autonudge_tick.
func (h *StallAutoNudgeHandler) Handle(ctx context.Context, _ *asynq.Task) error {
	if h == nil || h.Task == nil || h.Products == nil {
		return nil
	}
	if !h.Task.AutoStallNudge.Enabled {
		return nil
	}
	tctx, cancel := context.WithTimeout(ctx, 2*60*time.Second)
	defer cancel()
	if err := h.Task.RunAutoStallNudgeSweep(tctx, h.Products); err != nil {
		slog.Debug("stall autonudge sweep", "err", err)
		return err
	}
	return nil
}
