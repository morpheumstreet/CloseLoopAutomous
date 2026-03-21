package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/closeloopautomous/arms/internal/adapters/httpapi"
	"github.com/closeloopautomous/arms/internal/config"
	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/jobs"
	"github.com/closeloopautomous/arms/internal/platform"
)

func main() {
	cfg := config.LoadFromEnv()
	initLogging(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := platform.OpenApp(ctx, cfg)
	if err != nil {
		slog.Error("open app", "err", err)
		os.Exit(1)
	}
	defer func() { _ = app.Close() }()

	if cfg.RedisAddr != "" {
		asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
		defer func() { _ = asynqClient.Close() }()

		sched := jobs.NewScheduler(asynqClient, app.ProductSchedules)
		if sched != nil {
			if err := sched.StartProductSchedules(context.Background()); err != nil {
				slog.Debug("product schedule start", "err", err)
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
						_ = sched.StartProductSchedules(inner)
						cancel()
					}
				}
			}()
		}
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
		reconcile(context.Background())
		app.Handlers.AutopilotScheduleReconcile = reconcile

		if cfg.AutopilotTickSec > 0 && !cfg.UseAsynqScheduler {
			go func() {
				t := time.NewTicker(time.Duration(cfg.AutopilotTickSec) * time.Second)
				defer t.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-t.C:
						reconcile(context.Background())
					}
				}
			}()
		} else if cfg.AutopilotTickSec > 0 && cfg.UseAsynqScheduler {
			slog.Warn("arms autopilot", "msg", "ARMS_AUTOPILOT_TICK_SEC ignored when ARMS_USE_ASYNQ_SCHEDULER=true; periodic reconcile disabled (startup + HTTP hooks + per-product Asynq chain)")
		}
	} else if cfg.AutopilotTickSec > 0 {
		if cfg.UseAsynqScheduler {
			slog.Warn("arms autopilot", "msg", "ARMS_USE_ASYNQ_SCHEDULER set but ARMS_REDIS_ADDR empty — using in-process TickScheduled; set Redis and run arms-worker for Asynq mode")
		}
		go func() {
			t := time.NewTicker(time.Duration(cfg.AutopilotTickSec) * time.Second)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					tickCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					err := app.Handlers.Autopilot.TickScheduled(tickCtx, time.Now())
					cancel()
					if err != nil {
						slog.Debug("autopilot tick", "err", err)
					}
				}
			}
		}()
	}

	handler := httpapi.NewRouter(cfg, app.Handlers)
	handler = httpapi.CORSMiddleware(cfg.CORSAllowOrigin, handler)
	if cfg.DatabasePath != "" {
		slog.Info("arms persistence", "database_path", cfg.DatabasePath)
	} else {
		slog.Info("arms persistence", "mode", "in-memory", "hint", "set DATABASE_PATH for SQLite")
	}
	if cfg.OpenClawGatewayURL != "" {
		if cfg.OpenClawSessionKey == "" {
			slog.Warn("openclaw gateway url set but session key empty — dispatch will fail until ARMS_OPENCLAW_SESSION_KEY is set")
		}
		slog.Info("arms gateway", "kind", "openclaw_ws", "dispatch_timeout", cfg.OpenClawDispatchTimeout.String())
	} else {
		slog.Info("arms gateway", "kind", "stub")
	}
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
	case cfg.RedisAddr != "" && cfg.UseAsynqScheduler:
		slog.Info("arms autopilot", "mode", "asynq_authoritative", "redis", cfg.RedisAddr, "use_asynq_scheduler", true)
	case cfg.RedisAddr != "" && cfg.AutopilotTickSec > 0:
		slog.Info("arms autopilot", "mode", "asynq_per_product", "redis", cfg.RedisAddr, "reconcile_sec", cfg.AutopilotTickSec)
	case cfg.RedisAddr != "":
		slog.Info("arms autopilot", "mode", "asynq_per_product", "redis", cfg.RedisAddr, "reconcile_sec", 0, "hint", "set ARMS_AUTOPILOT_TICK_SEC>0 for periodic reconcile, or ARMS_USE_ASYNQ_SCHEDULER=true to rely on worker chain only")
	case cfg.AutopilotTickSec > 0:
		slog.Info("arms autopilot", "mode", "in_process", "tick_sec", cfg.AutopilotTickSec)
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
