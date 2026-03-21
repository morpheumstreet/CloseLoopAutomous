// Command arms-worker runs an Asynq consumer for future Redis-scheduled jobs (product_schedules / autopilot offload).
// Today it registers a no-op handler for arms:ping; extend handlers to call platform.OpenApp + autopilot when ready.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/closeloopautomous/arms/internal/config"
)

const queueDefault = "arms"

func main() {
	cfg := config.LoadFromEnv()
	initLogging(cfg)
	if cfg.RedisAddr == "" {
		slog.Info("arms-worker: ARMS_REDIS_ADDR not set — exiting (no Redis consumer)")
		return
	}
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{
			Concurrency: 2,
			Queues:      map[string]int{queueDefault: 1},
		},
	)
	mux := asynq.NewServeMux()
	mux.HandleFunc("arms:ping", func(ctx context.Context, t *asynq.Task) error {
		slog.Debug("asynq task", "type", t.Type())
		return nil
	})
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
