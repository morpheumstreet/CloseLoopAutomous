package platform

import (
	"context"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/ai"
	"github.com/closeloopautomous/arms/internal/adapters/budget"
	gw "github.com/closeloopautomous/arms/internal/adapters/gateway"
	"github.com/closeloopautomous/arms/internal/adapters/httpapi"
	"github.com/closeloopautomous/arms/internal/adapters/identity"
	"github.com/closeloopautomous/arms/internal/adapters/memory"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/application/autopilot"
	"github.com/closeloopautomous/arms/internal/application/convoy"
	"github.com/closeloopautomous/arms/internal/application/cost"
	"github.com/closeloopautomous/arms/internal/application/livefeed"
	"github.com/closeloopautomous/arms/internal/application/product"
	"github.com/closeloopautomous/arms/internal/application/task"
	"github.com/closeloopautomous/arms/internal/config"
	"github.com/closeloopautomous/arms/internal/ports"
)

// App bundles repositories and HTTP handlers; optional DB is closed by Close().
type App struct {
	Handlers *httpapi.Handlers
	Products ports.ProductRepository
	Ideas    ports.IdeaRepository
	Tasks    ports.TaskRepository
	db       dbCloser
	cleanup  func() // e.g. WebSocket gateway shutdown
}

type dbCloser interface {
	Close() error
}

// Close releases the SQLite handle when the app was opened with a file/database DSN.
func (a *App) Close() error {
	if a.cleanup != nil {
		a.cleanup()
		a.cleanup = nil
	}
	if a.db == nil {
		return nil
	}
	err := a.db.Close()
	a.db = nil
	return err
}

// NewInMemoryApp wires the hexagon with in-memory adapters (no persistence).
func NewInMemoryApp(cfg config.Config) *App {
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	costs := memory.NewCostStore()
	checkpoints := memory.NewCheckpointStore()
	maybePool := memory.NewMaybePoolStore()
	hub := livefeed.NewHub()
	h, cleanup := buildHandlers(cfg, products, ideas, tasks, convoys, costs, checkpoints, maybePool, hub, hub)
	return &App{Handlers: h, Products: products, Ideas: ideas, Tasks: tasks, db: nil, cleanup: cleanup}
}

func buildHandlers(
	cfg config.Config,
	products ports.ProductRepository,
	ideas ports.IdeaRepository,
	tasks ports.TaskRepository,
	convoys ports.ConvoyRepository,
	costs ports.CostRepository,
	checkpoints ports.CheckpointRepository,
	maybePool ports.MaybePoolRepository,
	hub *livefeed.Hub,
	taskEvents ports.LiveActivityPublisher,
) (*httpapi.Handlers, func()) {
	clock := timeadapter.System{}
	ids := &identity.Sequential{}
	agentGW, gwCleanup := gw.NewAgentGateway(
		cfg.OpenClawGatewayURL,
		cfg.OpenClawGatewayToken,
		cfg.ArmsDeviceID,
		cfg.OpenClawSessionKey,
		cfg.OpenClawDispatchTimeout,
	)

	productSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	autoSvc := &autopilot.Service{
		Products:   products,
		Ideas:      ideas,
		MaybePool:  maybePool,
		Research:   ai.ResearchStub{},
		Ideation:   ai.IdeationStub{},
		Clock:      clock,
		Identities: ids,
	}
	taskSvc := &task.Service{
		Tasks:    tasks,
		Products: products,
		Ideas:    ideas,
		Gateway:  agentGW,
		Budget:   &budget.Static{Cap: 100, Costs: costs},
		Checkpt:  checkpoints,
		Clock:    clock,
		IDs:      ids,
		Events:   taskEvents,
	}
	convoySvc := &convoy.Service{
		Convoys:  convoys,
		Tasks:    tasks,
		Products: products,
		Gateway:  agentGW,
		Clock:    clock,
		IDs:      ids,
	}
	costSvc := &cost.Service{Costs: costs, Clock: clock, IDs: ids}

	return &httpapi.Handlers{
		Config:    cfg,
		Product:   productSvc,
		Autopilot: autoSvc,
		Task:      taskSvc,
		Convoy:    convoySvc,
		Cost:      costSvc,
		Live:      hub,
	}, gwCleanup
}
