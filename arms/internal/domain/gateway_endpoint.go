package domain

import (
	"strings"
	"time"
)

// Gateway driver values stored on gateway_endpoints.driver and used for dispatch routing.
const (
	GatewayDriverStub       = "stub"
	GatewayDriverOpenClawWS = "openclaw_ws"
	// GatewayDriverNemoClawWS is OpenClaw-class WebSocket via NVIDIA NemoClaw / OpenShell (same wire as openclaw_ws; optional nemoclaw CLI before dial).
	GatewayDriverNemoClawWS = "nemoclaw_ws"
	// GatewayDriverNullClawWS is the legacy OpenClaw-shaped WebSocket RPC (not stock NullClaw HTTP).
	GatewayDriverNullClawWS = "nullclaw_ws"
	// GatewayDriverNullClawA2A is NullClaw's HTTP gateway: JSON-RPC 2.0 POST …/a2a (message/send).
	GatewayDriverNullClawA2A = "nullclaw_a2a"
	// GatewayDriverPicoClawWS is the Pico Protocol WebSocket used by PicoClaw (message.send + session_id).
	GatewayDriverPicoClawWS = "picoclaw_ws"
	// GatewayDriverZeroClawWS is the OpenClaw-compatible WebSocket RPC used by ZeroClaw (connect + chat.send).
	GatewayDriverZeroClawWS = "zeroclaw_ws"
	// GatewayDriverClawletWS is the OpenClaw-compatible WebSocket RPC used by Clawlet (same wire as openclaw_ws / zeroclaw_ws).
	GatewayDriverClawletWS = "clawlet_ws"
	// GatewayDriverIronClawWS is the OpenClaw-compatible WebSocket RPC used by IronClaw (Rust-native OpenClaw-class gateway).
	GatewayDriverIronClawWS = "ironclaw_ws"
	// GatewayDriverMimiClawWS is the JSON WebSocket protocol used by MimiClaw (type message + chat_id on port 18789).
	GatewayDriverMimiClawWS = "mimiclaw_ws"
	// GatewayDriverNanobotCLI runs HKUDS nanobot via `nanobot agent -m` (subprocess); not an OpenClaw WebSocket gateway.
	GatewayDriverNanobotCLI = "nanobot_cli"
	// GatewayDriverInkOSCLI runs InkOS via `inkos write next … --json` (subprocess); not an OpenClaw WebSocket gateway.
	GatewayDriverInkOSCLI = "inkos_cli"
	// GatewayDriverZClawRelayHTTP is the zclaw hosted web relay: JSON POST …/api/chat (see tnm/zclaw scripts/web_relay.py).
	GatewayDriverZClawRelayHTTP = "zclaw_relay_http"
	// GatewayDriverMisterMorphHTTP is the MisterMorph daemon/runtime task API: POST …/tasks + GET …/tasks/{id} (Bearer auth).
	GatewayDriverMisterMorphHTTP = "mistermorph_http"
	// GatewayDriverCoPawHTTP is AgentScope CoPaw: JSON-RPC 2.0 POST …/console/api (chat.send). device_id = workspace; session_key = chat/session id.
	GatewayDriverCoPawHTTP = "copaw_http"
	// GatewayDriverMetaClawHTTP is MetaClaw (or any OpenAI-compatible proxy): POST …/v1/chat/completions. device_id = optional model; session_key = OpenAI user (trace).
	GatewayDriverMetaClawHTTP = "metaclaw_http"
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
	case "nemoclaw", "nemoclaw_ws", "nemoclaw-ws", "nvidia-claw", "nvidia_claw":
		return GatewayDriverNemoClawWS
	case "nullclaw", "nullclaw_ws", "nullclaw-ws":
		return GatewayDriverNullClawWS
	case "nullclaw_a2a", "nullclaw-a2a", "nullclaw_http", "nullclaw-http":
		return GatewayDriverNullClawA2A
	case "picoclaw", "picoclaw_ws", "picoclaw-ws", "pico_claw", "pico-claw":
		return GatewayDriverPicoClawWS
	case "zeroclaw", "zeroclaw_ws", "zeroclaw-ws", "zero_claw", "zero-claw":
		return GatewayDriverZeroClawWS
	case "clawlet", "clawlet_ws", "clawlet-ws":
		return GatewayDriverClawletWS
	case "ironclaw", "ironclaw_ws", "ironclaw-ws", "iron_claw", "iron-claw":
		return GatewayDriverIronClawWS
	case "mimiclaw", "mimiclaw_ws", "mimiclaw-ws", "mimi_claw", "mimi-claw":
		return GatewayDriverMimiClawWS
	case "nanobot", "nanobot_cli", "nanobot-cli":
		return GatewayDriverNanobotCLI
	case "inkos", "inkos_cli", "inkos-cli":
		return GatewayDriverInkOSCLI
	case "zclaw", "zclaw_relay", "zclaw-relay", "zclaw_relay_http", "zclaw-relay-http", "zclaw_http", "zclaw-http":
		return GatewayDriverZClawRelayHTTP
	case "mistermorph", "mistermorph_http", "mistermorph-http", "mister_morph", "mister-morph":
		return GatewayDriverMisterMorphHTTP
	case "copaw", "copaw_http", "copaw-http", "agentscope-copaw", "agentscope_copaw":
		return GatewayDriverCoPawHTTP
	case "meta", "metaclaw", "metaclaw_http", "metaclaw-http", "meta_claw", "meta-claw", "metaclaw_openai", "metaclaw-openai":
		return GatewayDriverMetaClawHTTP
	default:
		return ""
	}
}
