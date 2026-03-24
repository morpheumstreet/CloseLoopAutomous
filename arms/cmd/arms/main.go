package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/httpapi"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/config"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/jobs"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/platform"
)

// Version and Commit are set at link time via -ldflags (see repo Makefile).
var (
	Version = "dev"
	Commit  = ""
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, errOpen := platform.OpenApp(ctx, cfg, platform.Build{Version: Version, Commit: Commit})
	if errOpen != nil {
		slog.Error("open app", "err", errOpen)
		os.Exit(1)
	}
	defer func() { _ = app.Close() }()

	if cfg.RedisAddr != "" {
		asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
		defer func() { _ = asynqClient.Close() }()

		if cfg.AutopilotTickSec > 0 {
			slog.Warn("arms autopilot", "msg", "ARMS_AUTOPILOT_TICK_SEC is deprecated and ignored when ARMS_REDIS_ADDR is set; autopilot uses Asynq (product:schedule:tick, arms:product_autopilot_tick) with startup + 5m resync of schedules and per-product reconcile")
		}
		if cfg.UseAsynqScheduler {
			slog.Warn("arms autopilot", "msg", "ARMS_USE_ASYNQ_SCHEDULER is deprecated and ignored; Asynq is always authoritative when Redis is configured")
		}

		sched := jobs.NewScheduler(asynqClient, app.ProductSchedules)
		reconcile := func(innerCtx context.Context) {
			if innerCtx == nil {
				innerCtx = context.Background()
			}
			c, cancel := context.WithTimeout(innerCtx, 2*time.Minute)
			defer cancel()
			err := jobs.ReconcileProductAutopilotTasks(c, asynqClient, app.Handlers.Autopilot, time.Now().UTC())
			if err != nil {
				slog.Debug("autopilot schedule reconcile", "err", err)
			}
		}
		if sched != nil {
			if err := sched.StartProductSchedules(context.Background()); err != nil {
				slog.Debug("product schedule start", "err", err)
			}
		}
		reconcile(context.Background())
		app.Handlers.AutopilotScheduleReconcile = reconcile
		app.Handlers.ResyncProductSchedule = func(innerCtx context.Context, pid domain.ProductID) {
			if sched == nil {
				return
			}
			c, cancel := context.WithTimeout(innerCtx, 30*time.Second)
			defer cancel()
			if err := sched.ResyncProduct(c, pid); err != nil {
				slog.Debug("product schedule resync", "product_id", string(pid), "err", err)
			}
		}

		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					inner, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					if sched != nil {
						_ = sched.StartProductSchedules(inner)
					}
					reconcile(inner)
					cancel()
				}
			}
		}()

		if cfg.AutoStallNudgeEnabled {
			interval := time.Duration(cfg.AutoStallNudgeIntervalSec) * time.Second
			if interval < time.Minute {
				interval = time.Minute
			}
			go func(client *asynq.Client, iv time.Duration) {
				enqueue := func() {
					_, err := client.Enqueue(asynq.NewTask(jobs.TaskStallAutoNudgeTick, nil), asynq.Queue(jobs.QueueName))
					if err != nil {
						slog.Debug("stall autonudge enqueue", "err", err)
					}
				}
				enqueue()
				ticker := time.NewTicker(iv)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						enqueue()
					}
				}
			}(asynqClient, interval)
			slog.Info("arms stall autonudge", "interval", interval.String(), "queue", jobs.QueueName, "task", jobs.TaskStallAutoNudgeTick)
		}
	} else {
		if cfg.AutoStallNudgeEnabled {
			slog.Warn("arms stall autonudge", "msg", "ARMS_AUTO_STALL_NUDGE_ENABLED set but ARMS_REDIS_ADDR empty — enqueue disabled; set Redis and run cmd/arms-worker for arms:stall_autonudge_tick")
		}
		if cfg.AutopilotTickSec > 0 {
			slog.Warn("arms autopilot", "msg", "ARMS_AUTOPILOT_TICK_SEC is deprecated and ignored without ARMS_REDIS_ADDR; periodic autopilot requires Redis and cmd/arms-worker (product_schedules + arms:product_autopilot_tick)")
		}
		if cfg.UseAsynqScheduler {
			slog.Warn("arms autopilot", "msg", "ARMS_USE_ASYNQ_SCHEDULER set but ARMS_REDIS_ADDR empty — ignored; set Redis and run arms-worker for background autopilot")
		}
	}

	handler := httpapi.NewRouter(cfg, app.Handlers)
	handler = httpapi.CORSMiddleware(cfg.CORSAllowOrigin, handler)
	if cfg.DatabasePath != "" {
		slog.Info("arms persistence", "database_path", cfg.DatabasePath)
	} else {
		slog.Info("arms persistence", "mode", "in-memory", "hint", "set DATABASE_PATH for SQLite")
	}
	slog.Info("arms gateway", "dispatch_default_timeout", cfg.GatewayDispatchTimeout.String(), "hint", "POST /api/gateway-endpoints + POST /api/agents with gateway_endpoint_id and session_key")
	authMode := "disabled"
	switch {
	case cfg.MCAPIToken != "" && len(cfg.ACLUsers) > 0:
		authMode = "bearer MC_API_TOKEN and/or Basic ARMS_ACL"
	case cfg.MCAPIToken != "":
		authMode = "bearer MC_API_TOKEN"
	case len(cfg.ACLUsers) > 0:
		authMode = "HTTP Basic (ARMS_ACL)"
	}
	switch {
	case cfg.RedisAddr != "":
		slog.Info("arms autopilot", "mode", "asynq", "redis", cfg.RedisAddr, "periodic_resync", "5m product_schedules + per-product autopilot reconcile")
	default:
		slog.Info("arms autopilot", "mode", "off", "hint", "set ARMS_REDIS_ADDR and run cmd/arms-worker for product:schedule:tick and arms:product_autopilot_tick")
	}
	slog.Info("arms listening", "addr", cfg.ListenAddr, "auth", authMode)

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: handler,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
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
