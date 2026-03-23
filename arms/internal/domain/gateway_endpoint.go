package domain

import (
	"strings"
	"time"
)

// Gateway driver values stored on gateway_endpoints.driver and used for dispatch routing.
const (
	GatewayDriverStub        = "stub"
	GatewayDriverOpenClawWS  = "openclaw_ws"
	// GatewayDriverNullClawWS is the legacy OpenClaw-shaped WebSocket RPC (not stock NullClaw HTTP).
	GatewayDriverNullClawWS = "nullclaw_ws"
	// GatewayDriverNullClawA2A is NullClaw's HTTP gateway: JSON-RPC 2.0 POST …/a2a (message/send).
	GatewayDriverNullClawA2A = "nullclaw_a2a"
	// GatewayDriverPicoClawWS is the Pico Protocol WebSocket used by PicoClaw (message.send + session_id).
	GatewayDriverPicoClawWS = "picoclaw_ws"
	// GatewayDriverZeroClawWS is the OpenClaw-compatible WebSocket RPC used by ZeroClaw (connect + chat.send).
	GatewayDriverZeroClawWS = "zeroclaw_ws"
)

// GatewayEndpoint is a persisted remote execution plane (URL + auth + driver).
type GatewayEndpoint struct {
	ID           string
	DisplayName  string
	Driver       string
	GatewayURL   string
	GatewayToken string
	DeviceID     string
	TimeoutSec   int
	ProductID    ProductID
	CreatedAt    time.Time
}

// DispatchTarget is the resolved connection + session for one gateway RPC.
type DispatchTarget struct {
	Driver       string
	GatewayURL   string
	GatewayToken string
	DeviceID     string
	SessionKey   string
	Timeout      time.Duration
}

// NormalizeGatewayDriver returns a canonical driver; unknown becomes empty.
func NormalizeGatewayDriver(s string) string {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case "", "stub", "none", "off", "disabled":
		return GatewayDriverStub
	case "openclaw", "openclaw_ws", "openclaw-ws":
		return GatewayDriverOpenClawWS
	case "nullclaw", "nullclaw_ws", "nullclaw-ws":
		return GatewayDriverNullClawWS
	case "nullclaw_a2a", "nullclaw-a2a", "nullclaw_http", "nullclaw-http":
		return GatewayDriverNullClawA2A
	case "picoclaw", "picoclaw_ws", "picoclaw-ws", "pico_claw", "pico-claw":
		return GatewayDriverPicoClawWS
	case "zeroclaw", "zeroclaw_ws", "zeroclaw-ws", "zero_claw", "zero-claw":
		return GatewayDriverZeroClawWS
	default:
		return ""
	}
}
