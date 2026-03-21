// Package config loads arms runtime settings from the environment (Mission Control–style names where noted).
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds process-wide settings shared by HTTP, persistence, and the agent gateway.
//
// Environment variables:
//   - ARMS_LISTEN — HTTP bind address (default ":8080")
//   - MC_API_TOKEN — Bearer API token; empty disables auth
//   - WEBHOOK_SECRET — HMAC key for POST /api/webhooks/agent-completion
//   - ARMS_ALLOW_SAME_ORIGIN — "1" or "true" to allow same-origin browser calls without Bearer when token is set
//   - DATABASE_PATH — SQLite file path; empty uses in-memory stores
//   - ARMS_DB_BACKUP — "1" or "true" to VACUUM INTO backup before migrate
//   - OPENCLAW_GATEWAY_URL — WebSocket gateway URL; empty uses stub gateway
//   - OPENCLAW_GATEWAY_TOKEN — Bearer token on WS handshake
//   - OPENCLAW_DISPATCH_TIMEOUT_SEC — dispatch RPC timeout seconds (default 30)
//   - ARMS_DEVICE_ID — optional X-Arms-Device-Id on WS handshake
//   - ARMS_OPENCLAW_SESSION_KEY — sessionKey for chat.send dispatch
//   - ARMS_LOG_JSON — "1" or "true" for JSON logs to stdout (default text)
//   - ARMS_ACCESS_LOG — "0", "false", "off", "no" disables per-request access logging (default on)
//   - ARMS_AUTOPILOT_TICK_SEC — interval for in-process autopilot cadence ticks; 0 or unset disables (default 0)
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
//   - ARMS_CORS_ALLOW_ORIGIN — optional; when non-empty, enables CORS for browser UIs on another origin (e.g. http://localhost:3000 for Fishtank). Use * only for quick local experiments.
//   - ARMS_ACL — optional HTTP Basic ACL: semicolon-separated entries "user|password|role". Role is admin (default) or read (GET/HEAD only). Non-empty enables auth when MC_API_TOKEN is empty, or adds Basic as an alternative when both are set. User/password must not contain '|' or ';'.
type Config struct {
	ListenAddr                  string
	MCAPIToken                  string
	WebhookSecret               string
	AllowLocalhost              bool
	DatabasePath                string
	DatabaseBackupBeforeMigrate bool
	OpenClawGatewayURL          string
	OpenClawGatewayToken        string
	OpenClawDispatchTimeout     time.Duration
	ArmsDeviceID                string
	OpenClawSessionKey          string
	LogJSON                     bool
	AccessLog                   bool
	AutopilotTickSec            int
	BudgetDefaultCap            float64
	GitHubToken                 string
	GitHubAPIURL                string
	GitHubPRBackend             string
	GhPath                      string
	GitHubHost                  string
	EnableGitWorktrees          bool
	GitBin                      string
	WorkspaceRoot               string
	AgentStaleSec               int
	CORSAllowOrigin             string
	ACLUsers                    []ACLUser
}

// ACLUser is one Basic-auth principal for coarse HTTP ACL (admin vs read-only).
type ACLUser struct {
	UserID   string
	Password string
	Role     string // "admin" or "read"
}

// LoadFromEnv reads configuration from the process environment.
func LoadFromEnv() Config {
	addr := os.Getenv("ARMS_LISTEN")
	if addr == "" {
		addr = ":8080"
	}
	token := os.Getenv("MC_API_TOKEN")
	secret := os.Getenv("WEBHOOK_SECRET")
	allow := strings.EqualFold(os.Getenv("ARMS_ALLOW_SAME_ORIGIN"), "1") ||
		strings.EqualFold(os.Getenv("ARMS_ALLOW_SAME_ORIGIN"), "true")
	dbPath := strings.TrimSpace(os.Getenv("DATABASE_PATH"))
	backup := strings.EqualFold(os.Getenv("ARMS_DB_BACKUP"), "1") ||
		strings.EqualFold(os.Getenv("ARMS_DB_BACKUP"), "true")
	ocURL := strings.TrimSpace(os.Getenv("OPENCLAW_GATEWAY_URL"))
	ocTok := strings.TrimSpace(os.Getenv("OPENCLAW_GATEWAY_TOKEN"))
	dt := 30 * time.Second
	if s := strings.TrimSpace(os.Getenv("OPENCLAW_DISPATCH_TIMEOUT_SEC")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			dt = time.Duration(n) * time.Second
		}
	}
	device := strings.TrimSpace(os.Getenv("ARMS_DEVICE_ID"))
	sessionKey := strings.TrimSpace(os.Getenv("ARMS_OPENCLAW_SESSION_KEY"))
	logJSON := strings.EqualFold(os.Getenv("ARMS_LOG_JSON"), "1") ||
		strings.EqualFold(os.Getenv("ARMS_LOG_JSON"), "true")
	accessLog := true
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ARMS_ACCESS_LOG"))) {
	case "0", "false", "off", "no":
		accessLog = false
	}
	autopilotTick := 0
	if s := strings.TrimSpace(os.Getenv("ARMS_AUTOPILOT_TICK_SEC")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			autopilotTick = n
		}
	}
	budgetCap := 100.0
	if s := strings.TrimSpace(os.Getenv("ARMS_BUDGET_DEFAULT_CAP")); s != "" {
		if f, err := strconv.ParseFloat(s, 64); err == nil && f >= 0 {
			budgetCap = f
		}
	}
	ghTok := strings.TrimSpace(os.Getenv("ARMS_GITHUB_TOKEN"))
	if ghTok == "" {
		ghTok = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}
	ghAPI := strings.TrimSpace(os.Getenv("ARMS_GITHUB_API_URL"))
	ghBackend := strings.ToLower(strings.TrimSpace(os.Getenv("ARMS_GITHUB_PR_BACKEND")))
	ghBin := strings.TrimSpace(os.Getenv("ARMS_GH_BIN"))
	ghHost := strings.TrimSpace(os.Getenv("ARMS_GITHUB_HOST"))
	gitWorktrees := strings.EqualFold(os.Getenv("ARMS_ENABLE_GIT_WORKTREES"), "1") ||
		strings.EqualFold(os.Getenv("ARMS_ENABLE_GIT_WORKTREES"), "true")
	gitExe := strings.TrimSpace(os.Getenv("ARMS_GIT_BIN"))
	wsRoot := strings.TrimSpace(os.Getenv("ARMS_WORKSPACE_ROOT"))
	agentStale := 300
	if s, ok := os.LookupEnv("ARMS_AGENT_STALE_SEC"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			agentStale = n
		}
	}
	corsOrigin := strings.TrimSpace(os.Getenv("ARMS_CORS_ALLOW_ORIGIN"))
	acl := parseARMSACL(os.Getenv("ARMS_ACL"))
	return Config{
		ListenAddr:                  addr,
		MCAPIToken:                  strings.TrimSpace(token),
		WebhookSecret:               strings.TrimSpace(secret),
		AllowLocalhost:              allow,
		DatabasePath:                dbPath,
		DatabaseBackupBeforeMigrate: backup,
		OpenClawGatewayURL:          ocURL,
		OpenClawGatewayToken:        ocTok,
		OpenClawDispatchTimeout:     dt,
		ArmsDeviceID:                device,
		OpenClawSessionKey:          sessionKey,
		LogJSON:                     logJSON,
		AccessLog:                   accessLog,
		AutopilotTickSec:            autopilotTick,
		BudgetDefaultCap:            budgetCap,
		GitHubToken:                 ghTok,
		GitHubAPIURL:                ghAPI,
		GitHubPRBackend:             ghBackend,
		GhPath:                      ghBin,
		GitHubHost:                  ghHost,
		EnableGitWorktrees:          gitWorktrees,
		GitBin:                      gitExe,
		WorkspaceRoot:               wsRoot,
		AgentStaleSec:               agentStale,
		CORSAllowOrigin:             corsOrigin,
		ACLUsers:                    acl,
	}
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
