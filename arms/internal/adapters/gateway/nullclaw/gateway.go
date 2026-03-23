// Package nullclaw integrates arms with NullClaw (https://github.com/nullclaw/nullclaw).
//
// Stock NullClaw uses HTTP JSON-RPC POST /a2a — use [New] with gateway driver nullclaw_a2a.
//
// Driver nullclaw_ws keeps the legacy OpenClaw-shaped WebSocket RPC (connect + chat.send) via [NewOpenClawCompatible].
package nullclaw

import (
	"github.com/closeloopautomous/arms/internal/adapters/gateway/openclaw"
)

// NewOpenClawCompatible returns the WebSocket client for OpenClaw-class gateways (including nullclaw_ws).
func NewOpenClawCompatible(opts openclaw.Options) *openclaw.Client {
	return openclaw.New(opts)
}
