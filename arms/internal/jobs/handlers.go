package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/autopilot"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// HandlerRegistry wires Asynq task types to autopilot execution and product schedule chaining.
type HandlerRegistry struct {
	auto    *autopilot.Service
	enqueue *asynq.Client
	sched   *Scheduler
	schedH  *ProductScheduleHandler
	stall   *StallAutoNudgeHandler
}

// NewHandlerRegistry builds a registry. Nil auto or enqueue yields a no-op Register.
// Product schedule tasks are registered only when repo is non-nil.
func NewHandlerRegistry(auto *autopilot.Service, enqueue *asynq.Client, repo ports.ProductScheduleRepository, stall *StallAutoNudgeHandler) *HandlerRegistry {
	h := &HandlerRegistry{auto: auto, enqueue: enqueue, stall: stall}
	if auto == nil || enqueue == nil {
		return h
	}
	h.sched = NewScheduler(enqueue, repo)
	if h.sched != nil {
		h.schedH = NewProductScheduleHandler(auto, repo, h.sched)
	}
	return h
}

// Register attaches task handlers to mux. Does not register smoke-test tasks (e.g. arms:ping).
func (h *HandlerRegistry) Register(mux *asynq.ServeMux) {
	if mux == nil || h.auto == nil || h.enqueue == nil {
		return
	}
	mux.HandleFunc(TaskAutopilotGlobalTick, h.handleGlobalAutopilotTick)
	mux.HandleFunc(TaskAutopilotProductTick, h.handleProductAutopilotTick)
	if h.schedH != nil {
		mux.HandleFunc(TaskProductScheduleTick, h.schedH.ProcessTask)
	}
	if h.stall != nil {
		mux.HandleFunc(TaskStallAutoNudgeTick, h.stall.Handle)
	}
}

func (h *HandlerRegistry) handleGlobalAutopilotTick(ctx context.Context, _ *asynq.Task) error {
	tickCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	err := h.auto.TickScheduled(tickCtx, time.Now().UTC())
	if err != nil {
		slog.Debug("autopilot tick", "err", err)
	}
	return err
}

func (h *HandlerRegistry) handleProductAutopilotTick(ctx context.Context, t *asynq.Task) error {
	pid, err := ParseProductAutopilotPayload(t.Payload())
	if err != nil {
		slog.Debug("product autopilot task", "err", err)
		return nil
	}
	tickCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	now := time.Now().UTC()
	err = RunProductAutopilotTask(tickCtx, h.enqueue, h.auto, pid, now)
	if err != nil {
		slog.Debug("product autopilot tick", "product_id", string(pid), "err", err)
	}
	return err
}
