package config

import (
	"strconv"
	"strings"
	"time"
)

// buildConfig merges defaults, optional file map (layeredSource.file), then environment (env wins).
func buildConfig(file map[string]string) Config {
	s := &layeredSource{file: file}

	addr := ":8080"
	if v := strings.TrimSpace(s.getenv("ARMS_LISTEN")); v != "" {
		addr = v
	}
	token := s.getenv("MC_API_TOKEN")
	secret := s.getenv("WEBHOOK_SECRET")
	allow := strings.EqualFold(s.getenv("ARMS_ALLOW_SAME_ORIGIN"), "1") ||
		strings.EqualFold(s.getenv("ARMS_ALLOW_SAME_ORIGIN"), "true")
	dbPath := strings.TrimSpace(s.getenv("DATABASE_PATH"))
	backup := strings.EqualFold(s.getenv("ARMS_DB_BACKUP"), "1") ||
		strings.EqualFold(s.getenv("ARMS_DB_BACKUP"), "true")
	dt := 30 * time.Second
	if v := strings.TrimSpace(s.getenv("OPENCLAW_DISPATCH_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			dt = time.Duration(n) * time.Second
		}
	}
	nemoBin := strings.TrimSpace(s.getenv("ARMS_NEMOCLAW_BIN"))
	nemoAutoStart := strings.EqualFold(s.getenv("ARMS_NEMOCLAW_AUTO_START"), "1") ||
		strings.EqualFold(s.getenv("ARMS_NEMOCLAW_AUTO_START"), "true")
	nemoBlueprint := strings.TrimSpace(s.getenv("ARMS_NEMOCLAW_DEFAULT_BLUEPRINT"))
	openClawDeviceSigning := strings.EqualFold(strings.TrimSpace(s.getenv("ARMS_DEVICE_SIGNING")), "1") ||
		strings.EqualFold(strings.TrimSpace(s.getenv("ARMS_DEVICE_SIGNING")), "true") ||
		strings.EqualFold(strings.TrimSpace(s.getenv("ARMS_DEVICE_SIGNING")), "yes") ||
		strings.EqualFold(strings.TrimSpace(s.getenv("ARMS_DEVICE_SIGNING")), "enabled") ||
		strings.EqualFold(strings.TrimSpace(s.getenv("ARMS_DEVICE_SIGNING")), "on")
	openClawDeviceIdentityFile := strings.TrimSpace(s.getenv("ARMS_DEVICE_IDENTITY_FILE"))
	logJSON := strings.EqualFold(s.getenv("ARMS_LOG_JSON"), "1") ||
		strings.EqualFold(s.getenv("ARMS_LOG_JSON"), "true")
	accessLog := true
	switch strings.ToLower(strings.TrimSpace(s.getenv("ARMS_ACCESS_LOG"))) {
	case "0", "false", "off", "no":
		accessLog = false
	}
	autopilotTick := 0
	if v := strings.TrimSpace(s.getenv("ARMS_AUTOPILOT_TICK_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			autopilotTick = n
		}
	}
	budgetCap := 100.0
	if v := strings.TrimSpace(s.getenv("ARMS_BUDGET_DEFAULT_CAP")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			budgetCap = f
		}
	}
	ghTok := strings.TrimSpace(s.getenv("ARMS_GITHUB_TOKEN"))
	if ghTok == "" {
		ghTok = strings.TrimSpace(s.getenv("GITHUB_TOKEN"))
	}
	ghAPI := strings.TrimSpace(s.getenv("ARMS_GITHUB_API_URL"))
	ghBackend := strings.ToLower(strings.TrimSpace(s.getenv("ARMS_GITHUB_PR_BACKEND")))
	ghBin := strings.TrimSpace(s.getenv("ARMS_GH_BIN"))
	ghHost := strings.TrimSpace(s.getenv("ARMS_GITHUB_HOST"))
	gitWorktrees := strings.EqualFold(s.getenv("ARMS_ENABLE_GIT_WORKTREES"), "1") ||
		strings.EqualFold(s.getenv("ARMS_ENABLE_GIT_WORKTREES"), "true")
	gitExe := strings.TrimSpace(s.getenv("ARMS_GIT_BIN"))
	wsRoot := strings.TrimSpace(s.getenv("ARMS_WORKSPACE_ROOT"))
	agentStale := 300
	if v, ok := s.lookup("ARMS_AGENT_STALE_SEC"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			agentStale = n
		}
	}
	corsOrigin := strings.TrimSpace(s.getenv("ARMS_CORS_ALLOW_ORIGIN"))
	acl := parseARMSACL(s.getenv("ARMS_ACL"))
	mergeBackend := strings.ToLower(strings.TrimSpace(s.getenv("ARMS_MERGE_BACKEND")))
	mergeMethod := strings.TrimSpace(s.getenv("ARMS_MERGE_METHOD"))
	mergeLease := 90
	if v := strings.TrimSpace(s.getenv("ARMS_MERGE_LEASE_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			mergeLease = n
		}
	}
	mergeOwner := strings.TrimSpace(s.getenv("ARMS_MERGE_LEASE_OWNER"))
	redisAddr := strings.TrimSpace(s.getenv("ARMS_REDIS_ADDR"))
	useAsynqSched := strings.EqualFold(s.getenv("ARMS_USE_ASYNQ_SCHEDULER"), "1") ||
		strings.EqualFold(s.getenv("ARMS_USE_ASYNQ_SCHEDULER"), "true")
	autoStallNudge := strings.EqualFold(s.getenv("ARMS_AUTO_STALL_NUDGE_ENABLED"), "1") ||
		strings.EqualFold(s.getenv("ARMS_AUTO_STALL_NUDGE_ENABLED"), "true")
	autoStallInterval := 300
	if v := strings.TrimSpace(s.getenv("ARMS_AUTO_STALL_NUDGE_INTERVAL_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			autoStallInterval = n
		}
	}
	autoStallCooldown := 3600
	if v, ok := s.lookup("ARMS_AUTO_STALL_NUDGE_COOLDOWN_SEC"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			autoStallCooldown = n
		}
	}
	autoStallMaxDay := 6
	if v, ok := s.lookup("ARMS_AUTO_STALL_NUDGE_MAX_PER_DAY"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			autoStallMaxDay = n
		}
	}
	autoStallReassign := strings.EqualFold(s.getenv("ARMS_AUTO_STALL_REASSIGN_ENABLED"), "1") ||
		strings.EqualFold(s.getenv("ARMS_AUTO_STALL_REASSIGN_ENABLED"), "true")
	autoStallReassignCD := 7200
	if v, ok := s.lookup("ARMS_AUTO_STALL_REASSIGN_COOLDOWN_SEC"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			autoStallReassignCD = n
		}
	}
	autoStallReassignMax := 4
	if v, ok := s.lookup("ARMS_AUTO_STALL_REASSIGN_MAX_PER_DAY"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			autoStallReassignMax = n
		}
	}
	knowSnippets := 5
	if v := strings.TrimSpace(s.getenv("ARMS_KNOWLEDGE_DISPATCH_SNIPPETS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			knowSnippets = n
		}
	}
	knowDisableInject := strings.EqualFold(s.getenv("ARMS_KNOWLEDGE_DISABLE_DISPATCH"), "1") ||
		strings.EqualFold(s.getenv("ARMS_KNOWLEDGE_DISABLE_DISPATCH"), "true")
	knowAutoIngest := true
	switch strings.ToLower(strings.TrimSpace(s.getenv("ARMS_KNOWLEDGE_AUTO_INGEST"))) {
	case "0", "false", "off", "no":
		knowAutoIngest = false
	}
	knowBackend := strings.ToLower(strings.TrimSpace(s.getenv("ARMS_KNOWLEDGE_BACKEND")))
	if knowBackend == "" {
		knowBackend = "fts5"
	}
	chromemPath := strings.TrimSpace(s.getenv("ARMS_CHROMEM_PERSISTENCE_PATH"))
	if chromemPath == "" {
		chromemPath = "./data/chromem-knowledge"
	}
	chromemCompress := strings.EqualFold(s.getenv("ARMS_CHROMEM_COMPRESS"), "1") ||
		strings.EqualFold(s.getenv("ARMS_CHROMEM_COMPRESS"), "true")
	chromemEmbedder := strings.ToLower(strings.TrimSpace(s.getenv("ARMS_CHROMEM_EMBEDDER")))
	if chromemEmbedder == "" {
		chromemEmbedder = "ollama"
	}
	chromemModel := strings.TrimSpace(s.getenv("ARMS_CHROMEM_EMBEDDER_MODEL"))
	chromemOllamaBase := strings.TrimSpace(s.getenv("ARMS_CHROMEM_OLLAMA_BASE_URL"))
	chromemOpenAIKey := strings.TrimSpace(s.getenv("ARMS_CHROMEM_OPENAI_API_KEY"))
	chromemOpenAIModel := strings.TrimSpace(s.getenv("ARMS_CHROMEM_OPENAI_MODEL"))

	llmBase := strings.TrimSpace(s.getenv("ARMS_LLM_BASE_URL"))
	if llmBase == "" {
		llmBase = "https://api.openai.com/v1"
	}
	llmKey := strings.TrimSpace(s.getenv("ARMS_LLM_API_KEY"))
	if llmKey == "" {
		llmKey = strings.TrimSpace(s.getenv("OPENAI_API_KEY"))
	}
	researchLLMModel := strings.TrimSpace(s.getenv("ARMS_RESEARCH_LLM_MODEL"))
	ideationLLMModel := strings.TrimSpace(s.getenv("ARMS_IDEATION_LLM_MODEL"))
	researchLLMTimeout := 120 * time.Second
	if v := strings.TrimSpace(s.getenv("ARMS_RESEARCH_LLM_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			researchLLMTimeout = time.Duration(n) * time.Second
		}
	}
	researchClawPollInterval := 3 * time.Second
	if v := strings.TrimSpace(s.getenv("ARMS_RESEARCH_CLAW_POLL_INTERVAL_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			researchClawPollInterval = time.Duration(n) * time.Second
		}
	}
	researchClawPollTimeout := 900 * time.Second
	if v := strings.TrimSpace(s.getenv("ARMS_RESEARCH_CLAW_POLL_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			researchClawPollTimeout = time.Duration(n) * time.Second
		}
	}
	ideationLLMTimeout := 180 * time.Second
	if v := strings.TrimSpace(s.getenv("ARMS_IDEATION_LLM_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ideationLLMTimeout = time.Duration(n) * time.Second
		}
	}
	geoIP2City := strings.TrimSpace(s.getenv("ARMS_GEOIP2_CITY"))

	return Config{
		ListenAddr:                        addr,
		MCAPIToken:                        strings.TrimSpace(token),
		WebhookSecret:                     strings.TrimSpace(secret),
		AllowLocalhost:                    allow,
		DatabasePath:                      dbPath,
		DatabaseBackupBeforeMigrate:       backup,
		GatewayDispatchTimeout:            dt,
		NemoClawBin:                       nemoBin,
		NemoClawAutoStart:                 nemoAutoStart,
		NemoClawDefaultBlueprint:          nemoBlueprint,
		OpenClawDeviceSigning:             openClawDeviceSigning,
		OpenClawDeviceIdentityFile:        openClawDeviceIdentityFile,
		LogJSON:                           logJSON,
		AccessLog:                         accessLog,
		AutopilotTickSec:                  autopilotTick,
		BudgetDefaultCap:                  budgetCap,
		GitHubToken:                       ghTok,
		GitHubAPIURL:                      ghAPI,
		GitHubPRBackend:                   ghBackend,
		GhPath:                            ghBin,
		GitHubHost:                        ghHost,
		EnableGitWorktrees:                gitWorktrees,
		GitBin:                            gitExe,
		WorkspaceRoot:                     wsRoot,
		AgentStaleSec:                     agentStale,
		CORSAllowOrigin:                   corsOrigin,
		ACLUsers:                          acl,
		MergeBackend:                      mergeBackend,
		MergeMethod:                       mergeMethod,
		MergeLeaseSec:                     mergeLease,
		MergeLeaseOwner:                   mergeOwner,
		RedisAddr:                         redisAddr,
		UseAsynqScheduler:                 useAsynqSched,
		AutoStallNudgeEnabled:             autoStallNudge,
		AutoStallNudgeIntervalSec:         autoStallInterval,
		AutoStallNudgeCooldownSec:         autoStallCooldown,
		AutoStallNudgeMaxPerDay:           autoStallMaxDay,
		AutoStallReassignEnabled:          autoStallReassign,
		AutoStallReassignCooldownSec:      autoStallReassignCD,
		AutoStallReassignMaxPerDay:        autoStallReassignMax,
		KnowledgeDispatchSnippetLimit:     knowSnippets,
		KnowledgeDisableDispatchInjection: knowDisableInject,
		KnowledgeAutoIngest:               knowAutoIngest,
		KnowledgeBackend:                  knowBackend,
		ChromemPersistencePath:            chromemPath,
		ChromemCompress:                   chromemCompress,
		ChromemEmbedder:                   chromemEmbedder,
		ChromemEmbedderModel:              chromemModel,
		ChromemOllamaBaseURL:              chromemOllamaBase,
		ChromemOpenAIAPIKey:               chromemOpenAIKey,
		ChromemOpenAIModel:                chromemOpenAIModel,
		LLMBaseURL:                        llmBase,
		LLMAPIKey:                         llmKey,
		ResearchLLMModel:                  researchLLMModel,
		ResearchLLMTimeout:                researchLLMTimeout,
		ResearchClawPollInterval:          researchClawPollInterval,
		ResearchClawPollTimeout:           researchClawPollTimeout,
		IdeationLLMModel:                  ideationLLMModel,
		IdeationLLMTimeout:                ideationLLMTimeout,
		GeoIP2CityPath:                    geoIP2City,
	}
}
