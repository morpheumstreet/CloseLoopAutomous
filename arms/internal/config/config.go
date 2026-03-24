// Package config loads arms runtime settings from the environment (Mission Control–style names where noted).
package config

import (
	"strings"
	"time"
)

// Config holds process-wide settings shared by HTTP, persistence, and the agent gateway.
//
// Environment variables:
//   - ARMS_LISTEN — HTTP bind address (default ":8080")
//   - MC_API_TOKEN — Bearer API token; empty disables auth
//   - WEBHOOK_SECRET — HMAC key for POST /api/webhooks/agent-completion and POST /api/webhooks/ci-completion
//   - ARMS_ALLOW_SAME_ORIGIN — "1" or "true" to allow same-origin browser calls without Bearer when token is set
//   - DATABASE_PATH — SQLite file path; empty uses in-memory stores
//   - ARMS_DB_BACKUP — "1" or "true" to VACUUM INTO backup before migrate
//   - OPENCLAW_DISPATCH_TIMEOUT_SEC — default per-RPC timeout when a gateway_endpoints row has timeout_sec = 0 (default 30)
//   - ARMS_NEMOCLAW_BIN — optional path to nemoclaw executable for driver nemoclaw_ws when ARMS_NEMOCLAW_AUTO_START is enabled
//   - ARMS_NEMOCLAW_AUTO_START — "1" or "true" to run `nemoclaw <device_id> start` before each dispatch (sandbox name = gateway endpoint device_id)
//   - ARMS_NEMOCLAW_DEFAULT_BLUEPRINT — reserved for future policy/blueprint hooks (parsed, not yet used by the adapter)
//   - ARMS_DEVICE_SIGNING — "1"/"true"/"yes"/"enabled"/"on" to send Mission Control–format Ed25519 device proof on OpenClaw connect
//   - ARMS_DEVICE_IDENTITY_FILE — path to device.json (version 1 PEM keys); empty defaults to ~/.mission-control/identity/device.json when signing is on
//   - ARMS_OPENCLAW_LIVE_CONTRACT — "1"/"true"/"yes" with gateway URL + session in env runs integration live gateway tests (#105); see internal/integration/openclaw_live_contract_test.go
//   - ARMS_LOG_JSON — "1" or "true" for JSON logs to stdout (default text)
//   - ARMS_ACCESS_LOG — "0", "false", "off", "no" disables per-request access logging (default on)
//   - ARMS_USE_ASYNQ_SCHEDULER — deprecated no-op (still parsed for compatibility). When ARMS_REDIS_ADDR is set, cmd/arms always uses Asynq as the scheduling plane; a warning is logged if this env is set.
//   - ARMS_AUTOPILOT_TICK_SEC — deprecated no-op (still parsed so misconfigurations can be warned). Autopilot cadence uses Redis + cmd/arms-worker: product_schedules (product:schedule:tick) and arms:product_autopilot_tick, with cmd/arms running startup + 5m resync (schedules + per-product reconcile) and HTTP hooks.
//   - ARMS_BUDGET_DEFAULT_CAP — cumulative spend ceiling per product when no cost_caps row exists (default 100); set 0 to disable
//   - ARMS_GITHUB_TOKEN — PAT with repo scope for POST /api/tasks/{id}/pull-request when using API backend (falls back to GITHUB_TOKEN if empty)
//   - ARMS_GITHUB_API_URL — optional GitHub Enterprise API root for REST backend, e.g. https://github.example.com/api/v3/
//   - ARMS_GITHUB_PR_BACKEND — pr create backend: empty or "api" uses REST + token; "gh" uses `gh pr create` (see ARMS_GH_BIN, ARMS_GITHUB_HOST)
//   - ARMS_GH_BIN — optional path to gh executable (default: look up "gh" on PATH)
//   - ARMS_GITHUB_HOST — optional GH_HOST for GitHub Enterprise when using the gh CLI backend
//   - ARMS_ENABLE_GIT_WORKTREES — "1" or "true" to allow POST /api/tasks/{id}/workspace/git-worktree (requires ARMS_WORKSPACE_ROOT + product.repo_clone_path)
//   - ARMS_GIT_BIN — git executable (default: look up "git" on PATH)
//   - ARMS_WORKSPACE_ROOT — absolute base directory for per-task worktree directories
//   - ARMS_AGENT_STALE_SEC — heartbeats older than this are flagged stale in JSON (default 300); 0 uses default
//   - ARMS_AUTO_STALL_NUDGE_ENABLED — "1" or "true" to enqueue periodic Asynq arms:stall_autonudge_tick (requires Redis + arms-worker)
//   - ARMS_AUTO_STALL_NUDGE_INTERVAL_SEC — enqueue interval for that tick (default 300); minimum 60 when enforced in cmd/arms
//   - ARMS_AUTO_STALL_NUDGE_COOLDOWN_SEC — min seconds between auto-nudges per task (default 3600)
//   - ARMS_AUTO_STALL_NUDGE_MAX_PER_DAY — max auto-nudges per task per rolling 24h (default 6); 0 disables the cap
//   - ARMS_AUTO_STALL_REASSIGN_ENABLED — "1" or "true" to re-dispatch stalled tasks to another execution agent before auto-nudge (#107; requires registry + gateway + same stall tick as nudge)
//   - ARMS_AUTO_STALL_REASSIGN_COOLDOWN_SEC — min seconds between auto-reassigns per task (default 7200 when reassign enabled and unset/0)
//   - ARMS_AUTO_STALL_REASSIGN_MAX_PER_DAY — max auto-reassigns per task per rolling 24h (default 4); 0 disables the cap
//   - ARMS_KNOWLEDGE_DISPATCH_SNIPPETS — max knowledge bullets appended per OpenClaw dispatch (default 5)
//   - ARMS_KNOWLEDGE_DISABLE_DISPATCH — "1" or "true" to keep knowledge HTTP/CRUD but skip dispatch-time injection
//   - ARMS_KNOWLEDGE_AUTO_INGEST — "0", "false", "off", "no" to disable auto-append to knowledge from swipes, product feedback, task/convoy completion (default on)
//   - ARMS_KNOWLEDGE_BACKEND — fts5 (default, SQLite FTS5) or chromem (semantic search via chromem-go; requires embedder)
//   - ARMS_CHROMEM_PERSISTENCE_PATH — directory for chromem persistent DB (default ./data/chromem-knowledge)
//   - ARMS_CHROMEM_COMPRESS — "1" or "true" to gzip chromem document blobs on disk
//   - ARMS_CHROMEM_EMBEDDER — ollama (default) or openai (OpenAI-compatible embeddings API)
//   - ARMS_CHROMEM_EMBEDDER_MODEL — Ollama embedding model (default nomic-embed-text)
//   - ARMS_CHROMEM_OLLAMA_BASE_URL — Ollama API base (default http://localhost:11434/api)
//   - ARMS_CHROMEM_OPENAI_API_KEY — OpenAI key (falls back to OPENAI_API_KEY)
//   - ARMS_CHROMEM_OPENAI_MODEL — embedding model id (default text-embedding-3-small)
//   - ARMS_LLM_BASE_URL — OpenAI-compatible API root for autopilot research/ideation (default https://api.openai.com/v1). Use your provider’s chat-completions base (e.g. https://api.deepseek.com/v1).
//   - ARMS_LLM_API_KEY — Bearer token for that API (falls back to OPENAI_API_KEY when empty; Ollama and some gateways need no key).
//   - ARMS_RESEARCH_LLM_MODEL — when non-empty, Run research uses this chat model instead of the in-process stub.
//   - ARMS_IDEATION_LLM_MODEL — when non-empty, Run ideation uses this chat model instead of the stub.
//   - ARMS_RESEARCH_LLM_TIMEOUT_SEC — HTTP-bound timeout for one research call (default 120).
//   - ARMS_IDEATION_LLM_TIMEOUT_SEC — HTTP-bound timeout for one ideation call (default 180).
//   - ARMS_GEOIP2_CITY — optional path to MaxMind GeoLite2-City.mmdb (offline); enriches synthesized agent identities with city-level geo from gateway URL host.
//   - ARMS_CORS_ALLOW_ORIGIN — optional; when non-empty, enables CORS for browser UIs on another origin (e.g. http://localhost:3000 for Fishtank). Use * only for quick local experiments.
//   - ARMS_ACL — optional HTTP Basic ACL: semicolon-separated entries "user|password|role". Role is admin (default) or read (GET/HEAD only). Non-empty enables auth when MC_API_TOKEN is empty, or adds Basic as an alternative when both are set. User/password must not contain '|' or ';'.
//   - ARMS_MERGE_BACKEND — merge queue completion: noop (default), github (REST merge PR), local (git merge in repo_clone_path)
//   - ARMS_MERGE_METHOD — github merge method: merge | squash | rebase (default merge)
//   - ARMS_MERGE_LEASE_SEC — lease TTL for merge-queue ship (default 90)
//   - ARMS_MERGE_LEASE_OWNER — optional instance id for queue leases (default hostname)
//   - ARMS_REDIS_ADDR — optional Redis (e.g. localhost:6379). When set, cmd/arms reconciles per-product arms:product_autopilot_tick on startup, every 5 minutes, and after product / product-schedule HTTP changes; cmd/arms-worker consumes the arms queue (product:schedule:tick, arms:product_autopilot_tick, arms:autopilot_tick for manual full TickScheduled sweeps). Without Redis, background autopilot is off (set Redis and run arms-worker).
//
// Config file (cmd/arms and cmd/arms-worker -c path):
//   - JSON or TOML with flat keys matching environment variable names (e.g. ARMS_LISTEN, DATABASE_PATH, MC_API_TOKEN).
//   - Keys are case-insensitive in file; hyphens are treated as underscores.
//   - Process environment always overrides values from the file when the variable is set in the environment.
type Config struct {
	ListenAddr                        string
	MCAPIToken                        string
	WebhookSecret                     string
	AllowLocalhost                    bool
	DatabasePath                      string
	DatabaseBackupBeforeMigrate       bool
	GatewayDispatchTimeout            time.Duration
	// NemoClaw (OpenShell): optional local `nemoclaw <sandbox> start` before OpenClaw-class WS dispatch.
	NemoClawBin                       string
	NemoClawAutoStart                 bool
	NemoClawDefaultBlueprint          string
	// OpenClawDeviceSigning enables Ed25519 device block on OpenClaw-class connect (ARMS_DEVICE_SIGNING).
	OpenClawDeviceSigning      bool
	OpenClawDeviceIdentityFile string // ARMS_DEVICE_IDENTITY_FILE; empty with signing uses ~/.mission-control/identity/device.json
	LogJSON                           bool
	AccessLog                         bool
	AutopilotTickSec                  int
	BudgetDefaultCap                  float64
	GitHubToken                       string
	GitHubAPIURL                      string
	GitHubPRBackend                   string
	GhPath                            string
	GitHubHost                        string
	EnableGitWorktrees                bool
	GitBin                            string
	WorkspaceRoot                     string
	AgentStaleSec                     int
	CORSAllowOrigin                   string
	ACLUsers                          []ACLUser
	MergeBackend                      string
	MergeMethod                       string
	MergeLeaseSec                     int
	MergeLeaseOwner                   string
	RedisAddr                         string
	UseAsynqScheduler                 bool
	AutoStallNudgeEnabled             bool
	AutoStallNudgeIntervalSec         int
	AutoStallNudgeCooldownSec         int
	AutoStallNudgeMaxPerDay           int
	AutoStallReassignEnabled          bool
	AutoStallReassignCooldownSec      int
	AutoStallReassignMaxPerDay        int
	KnowledgeDispatchSnippetLimit     int
	KnowledgeDisableDispatchInjection bool
	KnowledgeAutoIngest               bool
	KnowledgeBackend                  string
	ChromemPersistencePath            string
	ChromemCompress                   bool
	ChromemEmbedder                   string
	ChromemEmbedderModel              string
	ChromemOllamaBaseURL              string
	ChromemOpenAIAPIKey               string
	ChromemOpenAIModel                string
	LLMBaseURL                        string
	LLMAPIKey                         string
	ResearchLLMModel                  string
	ResearchLLMTimeout                time.Duration
	IdeationLLMModel                  string
	IdeationLLMTimeout                time.Duration
	// GeoIP2CityPath is optional MaxMind GeoLite2-City MMDB path (ARMS_GEOIP2_CITY).
	GeoIP2CityPath string
}

// ACLUser is one Basic-auth principal for coarse HTTP ACL (admin vs read-only).
type ACLUser struct {
	UserID   string
	Password string
	Role     string // "admin" or "read"
}

// LoadFromEnv reads configuration from the process environment only.
func LoadFromEnv() Config {
	return buildConfig(nil)
}

// parseARMSACL parses ARMS_ACL: "user|password|role" entries separated by ';'.
// Empty role defaults to admin. Only admin and read roles are accepted.
func parseARMSACL(s string) []ACLUser {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []ACLUser
	for _, ent := range strings.Split(s, ";") {
		ent = strings.TrimSpace(ent)
		if ent == "" {
			continue
		}
		parts := strings.SplitN(ent, "|", 3)
		if len(parts) != 3 {
			continue
		}
		uid := strings.TrimSpace(parts[0])
		pw := strings.TrimSpace(parts[1])
		role := strings.TrimSpace(strings.ToLower(parts[2]))
		if uid == "" || pw == "" {
			continue
		}
		if role == "" {
			role = "admin"
		}
		if role != "admin" && role != "read" {
			continue
		}
		out = append(out, ACLUser{UserID: uid, Password: pw, Role: role})
	}
	return out
}
