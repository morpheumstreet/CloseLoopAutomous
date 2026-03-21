// Command arms-worker runs an Asynq consumer: autopilot cadence ticks (and optional ping for smoke tests).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/closeloopautomous/arms/internal/config"
	"github.com/closeloopautomous/arms/internal/jobs"
	"github.com/closeloopautomous/arms/internal/platform"
)

func main() {
	cfg := config.LoadFromEnv()
	initLogging(cfg)
	if cfg.RedisAddr == "" {
		slog.Info("arms-worker: ARMS_REDIS_ADDR not set — exiting (no Redis consumer)")
		return
	}
	openCtx, cancelOpen := context.WithTimeout(context.Background(), 2*time.Minute)
	app, err := platform.OpenApp(openCtx, cfg)
	cancelOpen()
	if err != nil {
		slog.Error("open app", "err", err)
		os.Exit(1)
	}
	defer func() { _ = app.Close() }()

	enqueueClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
	defer func() { _ = enqueueClient.Close() }()

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{
			Concurrency: 2,
			Queues:      map[string]int{jobs.QueueDefault: 1},
		},
	)
	mux := asynq.NewServeMux()
	mux.HandleFunc("arms:ping", func(ctx context.Context, t *asynq.Task) error {
		slog.Debug("asynq task", "type", t.Type())
		return nil
	})
	mux.HandleFunc(jobs.TypeAutopilotTick, func(ctx context.Context, _ *asynq.Task) error {
		tickCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		err := app.Handlers.Autopilot.TickScheduled(tickCtx, time.Now().UTC())
		if err != nil {
			slog.Debug("autopilot tick", "err", err)
		}
		return err
	})
	mux.HandleFunc(jobs.TypeProductAutopilotTick, func(ctx context.Context, t *asynq.Task) error {
		pid, err := jobs.ParseProductAutopilotPayload(t.Payload())
		if err != nil {
			slog.Debug("product autopilot task", "err", err)
			return nil
		}
		tickCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		now := time.Now().UTC()
		err = jobs.RunProductAutopilotTask(tickCtx, enqueueClient, app.Handlers.Autopilot, pid, now)
		if err != nil {
			slog.Debug("product autopilot tick", "product_id", string(pid), "err", err)
		}
		return err
	})
	sched := jobs.NewScheduler(enqueueClient, app.ProductSchedules)
	if sched != nil {
		ph := jobs.NewProductScheduleHandler(app.Handlers.Autopilot, app.ProductSchedules, sched)
		mux.HandleFunc(jobs.TypeProductScheduleTick, ph.ProcessTask)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := srv.Start(mux); err != nil {
			slog.Error("asynq server", "err", err)
			stop()
		}
	}()
	<-ctx.Done()
	srv.Shutdown()
}

func initLogging(cfg config.Config) {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	var h slog.Handler
	if cfg.LogJSON {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}
