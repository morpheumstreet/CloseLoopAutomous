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
	}
}
