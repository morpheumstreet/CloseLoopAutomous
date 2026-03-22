package platform

import (
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/ai"
	"github.com/closeloopautomous/arms/internal/adapters/budget"
	gw "github.com/closeloopautomous/arms/internal/adapters/gateway"
	"github.com/closeloopautomous/arms/internal/adapters/httpapi"
	"github.com/closeloopautomous/arms/internal/adapters/identity"
	"github.com/closeloopautomous/arms/internal/adapters/memory"
	"github.com/closeloopautomous/arms/internal/adapters/shipping"
	timeadapter "github.com/closeloopautomous/arms/internal/adapters/time"
	"github.com/closeloopautomous/arms/internal/application/agent"
	"github.com/closeloopautomous/arms/internal/application/autopilot"
	"github.com/closeloopautomous/arms/internal/application/convoy"
	"github.com/closeloopautomous/arms/internal/application/cost"
	"github.com/closeloopautomous/arms/internal/application/feedback"
	"github.com/closeloopautomous/arms/internal/application/livefeed"
	"github.com/closeloopautomous/arms/internal/application/mergequeue"
	"github.com/closeloopautomous/arms/internal/application/product"
	"github.com/closeloopautomous/arms/internal/application/task"
	"github.com/closeloopautomous/arms/internal/application/taskchat"
	"github.com/closeloopautomous/arms/internal/config"
	"github.com/closeloopautomous/arms/internal/ports"
)

// App bundles repositories and HTTP handlers; optional DB is closed by Close().
type App struct {
	Handlers         *httpapi.Handlers
	Products         ports.ProductRepository
	Ideas            ports.IdeaRepository
	Tasks            ports.TaskRepository
	ProductSchedules ports.ProductScheduleRepository
	db               dbCloser
	cleanup          func() // e.g. WebSocket gateway shutdown
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
func NewInMemoryApp(cfg config.Config, b Build) *App {
	products := memory.NewProductStore()
	ideas := memory.NewIdeaStore()
	tasks := memory.NewTaskStore()
	convoys := memory.NewConvoyStore()
	costs := memory.NewCostStore()
	costCaps := memory.NewCostCapStore()
	checkpoints := memory.NewCheckpointStore()
	ws := memory.NewWorkspaceStore()
	maybePool := memory.NewMaybePoolStore()
	swipes := memory.NewSwipeHistoryStore()
	researchCycles := memory.NewResearchCycleStore()
	execAgents := memory.NewExecutionAgentStore()
	agentMail := memory.NewAgentMailboxStore()
	hub := livefeed.NewHub()
	agentHealth := memory.NewAgentHealthStore()
	pref := memory.NewPreferenceModelStore()
	ops := memory.NewOperationsLogStore()
	sched := memory.NewProductScheduleStore()
	cmail := memory.NewConvoyMailStore()
	productFb := memory.NewProductFeedbackStore()
	taskChat := memory.NewTaskChatStore()
	h, cleanup := buildHandlers(cfg, products, ideas, tasks, convoys, costs, costCaps, checkpoints, ws, ws, maybePool, swipes, researchCycles, execAgents, agentMail, agentHealth, pref, ops, sched, cmail, productFb, taskChat, hub, hub, nil, b)
	return &App{Handlers: h, Products: products, Ideas: ideas, Tasks: tasks, ProductSchedules: sched, db: nil, cleanup: cleanup}
}

func buildHandlers(
	cfg config.Config,
	products ports.ProductRepository,
	ideas ports.IdeaRepository,
	tasks ports.TaskRepository,
	convoys ports.ConvoyRepository,
	costs ports.CostRepository,
	costCaps ports.CostCapRepository,
	checkpoints ports.CheckpointRepository,
	workspacePorts ports.WorkspacePortRepository,
	mergeQueue ports.WorkspaceMergeQueueRepository,
	maybePool ports.MaybePoolRepository,
	swipes ports.SwipeHistoryRepository,
	researchCycles ports.ResearchCycleRepository,
	execAgents ports.ExecutionAgentRegistry,
	agentMail ports.AgentMailboxRepository,
	agentHealth ports.AgentHealthRepository,
	preferenceModels ports.PreferenceModelRepository,
	operationsLog ports.OperationsLogRepository,
	productSchedules ports.ProductScheduleRepository,
	convoyMail ports.ConvoyMailRepository,
	productFeedback ports.ProductFeedbackRepository,
	taskChat ports.TaskChatRepository,
	hub *livefeed.Hub,
	taskEvents ports.LiveActivityPublisher,
	liveTX ports.LiveActivityTX,
	b Build,
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

	budgetPolicy := &budget.Composite{
		Costs:             costs,
		Caps:              costCaps,
		Clock:             clock,
		DefaultCumulative: cfg.BudgetDefaultCap,
	}

	productSvc := &product.Service{Products: products, Clock: clock, IDs: ids}
	autoSvc := &autopilot.Service{
		Products:       products,
		Ideas:          ideas,
		MaybePool:      maybePool,
		Swipes:         swipes,
		Feedback:       productFeedback,
		ResearchCycles: researchCycles,
		Schedules:      productSchedules,
		PrefModel:      preferenceModels,
		Research:       ai.ResearchStub{},
		Ideation:       ai.IdeationStub{},
		Clock:          clock,
		Identities:     ids,
	}
	agentSvc := &agent.Service{
		Registry: execAgents,
		Mailbox:  agentMail,
		Clock:    clock,
		IDs:      ids,
	}
	ship := shipping.NewPullRequestPublisher(shipping.PublisherSettings{
		PRBackend:  cfg.GitHubPRBackend,
		APIToken:   cfg.GitHubToken,
		APIBaseURL: cfg.GitHubAPIURL,
		GhPath:     cfg.GhPath,
		GitHubHost: cfg.GitHubHost,
	})
	staleThresh := time.Duration(cfg.AgentStaleSec) * time.Second
	if cfg.AgentStaleSec <= 0 {
		staleThresh = 300 * time.Second
	}
	taskSvc := &task.Service{
		Tasks:       tasks,
		Products:    products,
		Ideas:       ideas,
		Gateway:     agentGW,
		Budget:      budgetPolicy,
		Checkpt:     checkpoints,
		Clock:       clock,
		IDs:         ids,
		Events:      taskEvents,
		LiveTX:      liveTX,
		Gate:        task.NewProductGate(),
		Ship:        ship,
		AgentHealth: agentHealth,
		AutoStallNudge: task.AutoStallNudgeSettings{
			Enabled:        cfg.AutoStallNudgeEnabled,
			StaleThreshold: staleThresh,
			Cooldown:       time.Duration(cfg.AutoStallNudgeCooldownSec) * time.Second,
			MaxPerDay:      cfg.AutoStallNudgeMaxPerDay,
		},
	}
	convoySvc := &convoy.Service{
		Convoys:  convoys,
		Tasks:    tasks,
		Products: products,
		Gateway:  agentGW,
		Budget:   budgetPolicy,
		Health:   agentHealth,
		Mail:     convoyMail,
		Clock:    clock,
		IDs:      ids,
		Events:   taskEvents,
	}
	costSvc := &cost.Service{
		Costs:  costs,
		Caps:   costCaps,
		Budget: budgetPolicy,
		Clock:  clock,
		IDs:    ids,
		Events: taskEvents,
		LiveTX: liveTX,
	}

	prMerger := shipping.NewPullRequestMergerFromConfig(cfg.GitHubToken, cfg.GitHubAPIURL)
	var gateChecker ports.PullRequestMergeGateChecker
	if g, ok := prMerger.(ports.PullRequestMergeGateChecker); ok {
		gateChecker = g
	}
	var wtMerger ports.WorktreeMerger
	if strings.EqualFold(strings.TrimSpace(cfg.MergeBackend), "local") {
		wtMerger = shipping.NewLocalGitMerger()
	}
	mergeShip := mergequeue.New(mergequeue.MergeConfig{
		Backend:     cfg.MergeBackend,
		MergeMethod: cfg.MergeMethod,
		LeaseOwner:  cfg.MergeLeaseOwner,
		LeaseTTLSec: cfg.MergeLeaseSec,
		GitBin:      cfg.GitBin,
	}, mergeQueue, tasks, products, prMerger, gateChecker, wtMerger, taskEvents, clock)
	if mergeShip != nil {
		taskSvc.MergeShip = mergeShip
	}

	feedbackSvc := &feedback.Service{
		Products: products,
		Feedback: productFeedback,
		Ideas:    ideas,
		Clock:    clock,
		IDs:      ids,
	}
	taskChatSvc := &taskchat.Service{
		Chat: taskChat, Tasks: tasks, Products: products,
		Clock: clock, IDs: ids, Events: taskEvents,
	}

	buildVer := strings.TrimSpace(b.Version)
	if buildVer == "" {
		buildVer = "dev"
	}

	return &httpapi.Handlers{
		Config:         cfg,
		Product:        productSvc,
		Autopilot:      autoSvc,
		Task:           taskSvc,
		Convoy:         convoySvc,
		Agent:          agentSvc,
		Cost:           costSvc,
		Feedback:       feedbackSvc,
		TaskChat:       taskChatSvc,
		Live:           hub,
		WorkspacePorts: workspacePorts,
		MergeQueue:     mergeQueue,
		MergeShip:      mergeShip,
		AgentHealth:    agentHealth,
		PrefModel:      preferenceModels,
		OperationsLog:  operationsLog,
		BuildVersion:   buildVer,
		BuildCommit:    strings.TrimSpace(b.Commit),
	}, gwCleanup
}
