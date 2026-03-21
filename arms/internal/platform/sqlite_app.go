package platform

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/sqlite"
	"github.com/closeloopautomous/arms/internal/application/livefeed"
	"github.com/closeloopautomous/arms/internal/config"
)

// OpenApp returns an in-memory app when cfg.DatabasePath is empty; otherwise opens SQLite, migrates, and wires sqlite repositories.
func OpenApp(ctx context.Context, cfg config.Config) (*App, error) {
	path := strings.TrimSpace(cfg.DatabasePath)
	if path == "" {
		return NewInMemoryApp(cfg), nil
	}
	db, err := sqlite.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	if cfg.DatabaseBackupBeforeMigrate {
		if err := sqlite.BackupBeforeMigrate(ctx, db, path); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	if err := sqlite.Migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	products := sqlite.NewProductStore(db)
	ideas := sqlite.NewIdeaStore(db)
	tasks := sqlite.NewTaskStore(db)
	convoys := sqlite.NewConvoyStore(db)
	costs := sqlite.NewCostStore(db)
	costCaps := sqlite.NewCostCapStore(db)
	checkpoints := sqlite.NewCheckpointStore(db)
	ws := sqlite.NewWorkspaceStore(db)
	maybePool := sqlite.NewMaybePoolStore(db)
	swipes := sqlite.NewSwipeHistoryStore(db)
	researchCycles := sqlite.NewResearchCycleStore(db)
	execAgents := sqlite.NewExecutionAgentStore(db)
	agentMail := sqlite.NewAgentMailboxStore(db)
	agentHealth := sqlite.NewAgentHealthStore(db)
	hub := livefeed.NewHub()
	outbox := sqlite.NewOutboxStore(db)
	relayCtx, relayCancel := context.WithCancel(ctx)
	go livefeed.RunOutboxRelay(relayCtx, outbox, hub, 200*time.Millisecond)
	taskPub := &livefeed.OutboxPublisher{Outbox: outbox}
	liveTX := sqlite.NewLiveActivityTX(db)
	h, gwCleanup := buildHandlers(cfg, products, ideas, tasks, convoys, costs, costCaps, checkpoints, ws, ws, maybePool, swipes, researchCycles, execAgents, agentMail, agentHealth, hub, taskPub, liveTX)
	cleanup := func() {
		relayCancel()
		gwCleanup()
	}
	return &App{
		Handlers: h,
		Products: products,
		Ideas:    ideas,
		Tasks:    tasks,
		db:       db,
		cleanup:  cleanup,
	}, nil
}

// compile-time: *sql.DB closes
var _ dbCloser = (*sql.DB)(nil)
