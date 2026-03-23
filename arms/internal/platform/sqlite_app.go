package platform

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/sqlite"
	"github.com/closeloopautomous/arms/internal/application/agentidentity"
	"github.com/closeloopautomous/arms/internal/application/livefeed"
	"github.com/closeloopautomous/arms/internal/config"
	"github.com/closeloopautomous/arms/internal/platform/geoip"
)

// OpenApp returns an in-memory app when cfg.DatabasePath is empty; otherwise opens SQLite, migrates, and wires sqlite repositories.
func OpenApp(ctx context.Context, cfg config.Config, b Build) (*App, error) {
	path := strings.TrimSpace(cfg.DatabasePath)
	if path == "" {
		return NewInMemoryApp(cfg, b), nil
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
	gatewayEndpoints := sqlite.NewGatewayEndpointStore(db)
	agentMail := sqlite.NewAgentMailboxStore(db)
	agentHealth := sqlite.NewAgentHealthStore(db)
	pref := sqlite.NewPreferenceModelStore(db)
	ops := sqlite.NewOperationsLogStore(db)
	sched := sqlite.NewProductScheduleStore(db)
	cmail := sqlite.NewConvoyMailStore(db)
	productFb := sqlite.NewProductFeedbackStore(db)
	taskChat := sqlite.NewTaskChatStore(db)
	knowledge, knowUseFTS, err := newKnowledgeRepository(ctx, cfg, db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	hub := livefeed.NewHub()
	outbox := sqlite.NewOutboxStore(db)
	relayCtx, relayCancel := context.WithCancel(ctx)
	go livefeed.RunOutboxRelay(relayCtx, outbox, hub, 200*time.Millisecond)
	taskPub := &livefeed.OutboxPublisher{Outbox: outbox}
	liveTX := sqlite.NewLiveActivityTX(db)
	geoR, geoCleanup := geoip.NewResolver(cfg.GeoIP2CityPath)
	agentProfiles := sqlite.NewAgentProfileStore(db)
	idSvc := &agentidentity.Service{
		Endpoints: gatewayEndpoints,
		Profiles:  agentProfiles,
		Geo:       geoR,
		Events:    taskPub,
	}
	h, gwCleanup := buildHandlers(cfg, products, ideas, tasks, convoys, costs, costCaps, checkpoints, ws, ws, maybePool, swipes, researchCycles, execAgents, gatewayEndpoints, agentMail, agentHealth, pref, ops, sched, cmail, productFb, taskChat, knowledge, knowUseFTS, hub, taskPub, liveTX, idSvc, sqlite.ExpectedSchemaVersion, b)
	if err := idSvc.RefreshAll(ctx); err != nil {
		slog.Default().Warn("arms agent identity bootstrap refresh", "err", err)
	}
	cleanup := func() {
		relayCancel()
		gwCleanup()
		geoCleanup()
	}
	return &App{
		Handlers:         h,
		Products:         products,
		Ideas:            ideas,
		Tasks:            tasks,
		ProductSchedules: sched,
		db:               db,
		cleanup:          cleanup,
	}, nil
}

// compile-time: *sql.DB closes
var _ dbCloser = (*sql.DB)(nil)
