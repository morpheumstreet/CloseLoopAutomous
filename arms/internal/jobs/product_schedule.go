package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"

	"github.com/closeloopautomous/arms/internal/application/autopilot"
	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// ProductSchedulePayload is the Asynq payload for TaskProductScheduleTick.
type ProductSchedulePayload struct {
	ProductID string `json:"product_id"`
}

// Scheduler enqueues delayed/cron follow-ups for product_schedules rows (Redis + worker required).
type Scheduler struct {
	client *asynq.Client
	repo   ports.ProductScheduleRepository
}

// NewScheduler returns nil if repo is nil.
func NewScheduler(client *asynq.Client, repo ports.ProductScheduleRepository) *Scheduler {
	if repo == nil {
		return nil
	}
	return &Scheduler{client: client, repo: repo}
}

// EnqueueNextRun schedules the next tick from cron_expr, delay_seconds, or does nothing when neither applies.
func (s *Scheduler) EnqueueNextRun(ctx context.Context, sched *domain.ProductSchedule) error {
	if s == nil || s.client == nil {
		return nil
	}
	if !sched.Enabled {
		return nil
	}
	now := time.Now().UTC()
	var next time.Time
	switch {
	case strings.TrimSpace(sched.CronExpr) != "":
		spec, err := cron.ParseStandard(strings.TrimSpace(sched.CronExpr))
		if err != nil {
			return err
		}
		next = spec.Next(now)
	case sched.DelaySeconds > 0:
		next = now.Add(time.Duration(sched.DelaySeconds) * time.Second)
		sched.DelaySeconds = 0
	default:
		return nil
	}

	payload, err := json.Marshal(ProductSchedulePayload{ProductID: string(sched.ProductID)})
	if err != nil {
		return err
	}
	task := asynq.NewTask(TaskProductScheduleTick, payload)
	wait := next.Sub(now)
	if wait < 0 {
		wait = 0
	}
	opts := []asynq.Option{
		asynq.Queue(QueueName),
		asynq.ProcessIn(wait),
		asynq.MaxRetry(2),
		asynq.Timeout(10 * time.Minute),
	}
	info, err := s.client.Enqueue(task, opts...)
	if err != nil {
		return err
	}
	enq := now
	sched.AsynqTaskID = info.ID
	sched.LastEnqueuedAt = &enq
	sched.NextScheduledAt = &next
	return s.repo.Upsert(ctx, sched)
}

// StartProductSchedules enqueues rows that have no pending task id recorded (startup / periodic resync).
func (s *Scheduler) StartProductSchedules(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return nil
	}
	schedules, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return err
	}
	for i := range schedules {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		sched := &schedules[i]
		if strings.TrimSpace(sched.CronExpr) == "" && sched.DelaySeconds <= 0 {
			continue
		}
		if sched.AsynqTaskID != "" {
			continue
		}
		if err := s.EnqueueNextRun(ctx, sched); err != nil {
			// best-effort: other products still get scheduled
			continue
		}
	}
	return nil
}

// ResyncProduct enqueues the next run for one product when timing fields warrant it.
func (s *Scheduler) ResyncProduct(ctx context.Context, productID domain.ProductID) error {
	if s == nil {
		return nil
	}
	sched, err := s.repo.Get(ctx, productID)
	if err != nil || sched == nil || !sched.Enabled {
		return nil
	}
	if strings.TrimSpace(sched.CronExpr) == "" && sched.DelaySeconds <= 0 {
		return nil
	}
	return s.EnqueueNextRun(ctx, sched)
}

// ProductScheduleHandler runs one autopilot tick for a product and chains the next schedule.
type ProductScheduleHandler struct {
	auto *autopilot.Service
	repo ports.ProductScheduleRepository
	sch  *Scheduler
}

func NewProductScheduleHandler(auto *autopilot.Service, repo ports.ProductScheduleRepository, sch *Scheduler) *ProductScheduleHandler {
	return &ProductScheduleHandler{auto: auto, repo: repo, sch: sch}
}

// ProcessTask implements asynq.HandlerFunc.
func (h *ProductScheduleHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p ProductSchedulePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	if p.ProductID == "" {
		return nil
	}
	pid := domain.ProductID(p.ProductID)
	sched, err := h.repo.Get(ctx, pid)
	if err != nil || sched == nil || !sched.Enabled {
		return nil
	}
	now := time.Now().UTC()
	if err := h.auto.TickProduct(ctx, pid, now); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("autopilot tick: %w", err)
	}
	sched2, err := h.repo.Get(ctx, pid)
	if err != nil || sched2 == nil || !sched2.Enabled {
		return nil
	}
	if h.sch == nil {
		return nil
	}
	return h.sch.EnqueueNextRun(ctx, sched2)
}
