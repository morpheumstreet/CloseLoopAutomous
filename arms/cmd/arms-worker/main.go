// Command arms-worker runs an Asynq consumer: autopilot cadence ticks (and optional ping for smoke tests).
package main

import (
	"context"
	"flag"
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
	var configPath string
	flag.StringVar(&configPath, "c", "", "optional path to config.json or config.toml (same keys as env vars; environment overrides file)")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	initLogging(cfg)
	if cfg.RedisAddr == "" {
		slog.Info("arms-worker: ARMS_REDIS_ADDR not set — exiting (no Redis consumer)")
		return
	}
	openCtx, cancelOpen := context.WithTimeout(context.Background(), 2*time.Minute)
	app, errOpen := platform.OpenApp(openCtx, cfg, platform.Build{})
	cancelOpen()
	if errOpen != nil {
		slog.Error("open app", "err", errOpen)
		os.Exit(1)
	}
	defer func() { _ = app.Close() }()

	enqueueClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
	defer func() { _ = enqueueClient.Close() }()

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{
			Concurrency: 4,
			Queues:      map[string]int{jobs.QueueName: 1},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
				slog.Error("asynq task failed", "type", t.Type(), "err", err)
			}),
		},
	)
	mux := asynq.NewServeMux()
	mux.HandleFunc("arms:ping", func(ctx context.Context, t *asynq.Task) error {
		slog.Debug("asynq task", "type", t.Type())
		return nil
	})
	stallH := jobs.NewStallAutoNudgeHandler(app.Handlers.Task, app.Products)
	reg := jobs.NewHandlerRegistry(app.Handlers.Autopilot, enqueueClient, app.ProductSchedules, stallH)
	reg.Register(mux)

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
