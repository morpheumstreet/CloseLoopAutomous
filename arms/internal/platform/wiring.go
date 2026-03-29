package platform

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/ai"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/sqlite"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/budget"
	gw "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/nemoclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/httpapi"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/identity"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/memory"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/shipping"
	timeadapter "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/time"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/agent"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/agentidentity"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/autopilot"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/convoy"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/cost"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/feedback"
	knowledgeapp "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/knowledge"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/livefeed"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/mergequeue"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/product"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/task"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/taskchat"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/config"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/platform/geoip"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
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
	gatewayEndpoints := memory.NewGatewayEndpointStore()
	agentMail := memory.NewAgentMailboxStore()
	hub := livefeed.NewHub()
	agentHealth := memory.NewAgentHealthStore()
	pref := memory.NewPreferenceModelStore()
	ops := memory.NewOperationsLogStore()
	sched := memory.NewProductScheduleStore()
	cmail := memory.NewConvoyMailStore()
	productFb := memory.NewProductFeedbackStore()
	taskChat := memory.NewTaskChatStore()
	knowledge := memory.NewKnowledgeStore()
	if strings.EqualFold(strings.TrimSpace(cfg.KnowledgeBackend), "chromem") {
		slog.Default().Warn("ARMS_KNOWLEDGE_BACKEND=chromem is ignored when DATABASE_PATH is empty (in-memory mode); using in-memory knowledge store")
	}
	geoR, geoCleanup := geoip.NewResolver(cfg.GeoIP2CityPath)
	agentProfiles := memory.NewAgentProfileStore()
	researchHubs := memory.NewResearchHubStore()
	researchSettings := memory.NewResearchSystemSettingsStore()
	h, idSvc, gwCleanup := buildHandlers(cfg, products, ideas, tasks, convoys, costs, costCaps, checkpoints, ws, ws, maybePool, swipes, researchCycles, execAgents, gatewayEndpoints, researchHubs, researchSettings, agentMail, agentHealth, pref, ops, sched, cmail, productFb, taskChat, knowledge, false, hub, hub, nil, agentProfiles, geoR, sqlite.ExpectedSchemaVersion, b)
	if err := idSvc.RefreshAll(context.Background()); err != nil {
		slog.Default().Warn("arms agent identity bootstrap refresh", "err", err)
	}
	cleanup := func() {
		gwCleanup()
		geoCleanup()
	}
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
	gatewayEndpoints ports.GatewayEndpointRegistry,
	researchHubs ports.ResearchHubRegistry,
	researchSettings ports.ResearchSystemSettingsRepository,
	agentMail ports.AgentMailboxRepository,
	agentHealth ports.AgentHealthRepository,
	preferenceModels ports.PreferenceModelRepository,
	operationsLog ports.OperationsLogRepository,
	productSchedules ports.ProductScheduleRepository,
	convoyMail ports.ConvoyMailRepository,
	productFeedback ports.ProductFeedbackRepository,
	taskChat ports.TaskChatRepository,
	knowledge ports.KnowledgeRepository,
	knowledgeUseFTSQuerySyntax bool,
	hub *livefeed.Hub,
	taskEvents ports.LiveActivityPublisher,
	liveTX ports.LiveActivityTX,
	agentProfiles ports.AgentProfileRepository,
	geoR ports.GeoIPResolver,
	expectedSchemaVersion int,
	b Build,
) (*httpapi.Handlers, *agentidentity.Service, func()) {
	clock := timeadapter.System{}
	ids := &identity.Sequential{}
	knowSvc := &knowledgeapp.Service{
		Products:             products,
		Repo:                 knowledge,
		Clock:                clock,
		DispatchSnippetLimit: cfg.KnowledgeDispatchSnippetLimit,
		UseFTSQuerySyntax:    knowledgeUseFTSQuerySyntax,
		AutoIngest:           cfg.KnowledgeAutoIngest,
	}
	var knowHook func(context.Context, domain.ProductID, string) (string, error)
	if !cfg.KnowledgeDisableDispatchInjection {
		knowHook = knowSvc.DispatchHook()
	}
	if err := gw.EnsureDefaultStubEndpoint(context.Background(), gatewayEndpoints, clock); err != nil {
		slog.Error("arms gateway endpoint seed", "err", err)
	}
	agentGW, gwCleanup := gw.NewRoutingGateway(gatewayEndpoints, execAgents, knowHook, cfg.GatewayDispatchTimeout, nemoclaw.PoolSettings{
		BinaryPath:       cfg.NemoClawBin,
		AutoStart:        cfg.NemoClawAutoStart,
		DefaultBlueprint: cfg.NemoClawDefaultBlueprint,
	}, openclaw.ConnectEnv{
		DeviceSigning:      cfg.OpenClawDeviceSigning,
		DeviceIdentityFile: cfg.OpenClawDeviceIdentityFile,
	})
	idSvc := &agentidentity.Service{
		Endpoints: gatewayEndpoints,
		Profiles:  agentProfiles,
		Registry:  execAgents,
		Source:    agentGW,
		Geo:       geoR,
		Events:    taskEvents,
	}

	budgetPolicy := &budget.Composite{
		Costs:             costs,
		Caps:              costCaps,
		Clock:             clock,
		DefaultCumulative: cfg.BudgetDefaultCap,
	}

	productSvc := &product.Service{Products: products, Clock: clock, IDs: ids}

	llmChat := &ai.ChatClient{
		BaseURL: cfg.LLMBaseURL,
		APIKey:  cfg.LLMAPIKey,
		HTTP:    ai.DefaultHTTPClient(),
	}
	fallbackResearch := ports.ResearchPort(ai.ResearchStub{})
	if strings.TrimSpace(cfg.ResearchLLMModel) != "" {
		fallbackResearch = &ai.ResearchLLM{
			Client:  llmChat,
			Model:   strings.TrimSpace(cfg.ResearchLLMModel),
			Timeout: cfg.ResearchLLMTimeout,
		}
		slog.Info("arms autopilot", "research_llm", cfg.ResearchLLMModel, "base", cfg.LLMBaseURL)
	}
	researchPort := fallbackResearch
	if researchHubs != nil && researchSettings != nil {
		researchPort = &ai.ResearchRouter{
			Settings:     researchSettings,
			Hubs:         researchHubs,
			Fallback:     fallbackResearch,
			HTTP:         ai.DefaultHTTPClient(),
			PollInterval: cfg.ResearchClawPollInterval,
			PollTimeout:  cfg.ResearchClawPollTimeout,
		}
	}
	ideationPort := ports.IdeationPort(ai.IdeationStub{})
	if strings.TrimSpace(cfg.IdeationLLMModel) != "" {
		ideationPort = &ai.IdeationLLM{
			Client:  llmChat,
			Model:   strings.TrimSpace(cfg.IdeationLLMModel),
			Timeout: cfg.IdeationLLMTimeout,
		}
		slog.Info("arms autopilot", "ideation_llm", cfg.IdeationLLMModel, "base", cfg.LLMBaseURL)
	}

	autoSvc := &autopilot.Service{
		Products:       products,
		Ideas:          ideas,
		MaybePool:      maybePool,
		Swipes:         swipes,
		Feedback:       productFeedback,
		ResearchCycles: researchCycles,
		Schedules:      productSchedules,
		PrefModel:      preferenceModels,
		Research:       researchPort,
		Ideation:       ideationPort,
		Clock:          clock,
		Identities:     ids,
	}
	agentSvc := &agent.Service{
		Registry:  execAgents,
		Endpoints: gatewayEndpoints,
		Mailbox:   agentMail,
		Clock:     clock,
		IDs:       ids,
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
	reassignCD := time.Duration(cfg.AutoStallReassignCooldownSec) * time.Second
	if cfg.AutoStallReassignEnabled && reassignCD <= 0 {
		reassignCD = 2 * time.Hour
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
		ExecAgents:  execAgents,
		AutoStallNudge: task.AutoStallNudgeSettings{
			Enabled:        cfg.AutoStallNudgeEnabled,
			StaleThreshold: staleThresh,
			Cooldown:       time.Duration(cfg.AutoStallNudgeCooldownSec) * time.Second,
			MaxPerDay:      cfg.AutoStallNudgeMaxPerDay,
		},
		AutoStallReassign: task.AutoStallReassignSettings{
			Enabled:   cfg.AutoStallReassignEnabled,
			Cooldown:  reassignCD,
			MaxPerDay: cfg.AutoStallReassignMaxPerDay,
		},
		KnowledgeAutoIngest: func(ctx context.Context, t *domain.Task, source string, knowledgeSummary string) {
			_ = knowSvc.IngestFromTaskCompletion(ctx, t, source, knowledgeSummary)
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
		Config:                cfg,
		Product:               productSvc,
		Autopilot:             autoSvc,
		Task:                  taskSvc,
		Convoy:                convoySvc,
		Agent:                 agentSvc,
		GatewayEndpoints:      gatewayEndpoints,
		ResearchHubs:          researchHubs,
		ResearchSettings:      researchSettings,
		IDs:                   ids,
		Cost:                  costSvc,
		Feedback:              feedbackSvc,
		TaskChat:              taskChatSvc,
		Knowledge:             knowSvc,
		Live:                  hub,
		WorkspacePorts:        workspacePorts,
		MergeQueue:            mergeQueue,
		MergeShip:             mergeShip,
		AgentHealth:           agentHealth,
		PrefModel:             preferenceModels,
		OperationsLog:         operationsLog,
		BuildVersion:          buildVer,
		BuildCommit:           strings.TrimSpace(b.Commit),
		ExpectedSchemaVersion: expectedSchemaVersion,
		AgentIdentity:         idSvc,
	}, idSvc, gwCleanup
}
