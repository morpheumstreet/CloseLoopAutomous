package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/httpapi"
	"github.com/closeloopautomous/arms/internal/config"
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

	if cfg.AutopilotTickSec > 0 {
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
	if cfg.MCAPIToken != "" {
		authMode = "bearer MC_API_TOKEN"
	}
	if cfg.AutopilotTickSec > 0 {
		slog.Info("arms autopilot", "tick_sec", cfg.AutopilotTickSec)
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
