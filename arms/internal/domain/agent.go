package domain

import "time"

// ExecutionAgent is a registered logical agent slot (registry), distinct from per-task agent health heartbeats.
type ExecutionAgent struct {
	ID          string
	DisplayName string
	ProductID   ProductID // optional scope; empty = global
	Source      string    // e.g. manual, gateway
	ExternalRef string
	// EndpointID references gateway_endpoints; session_key is the dispatch target on that endpoint
	// (OpenClaw/PicoClaw: session id; NullClaw A2A: contextId).
	EndpointID  string
	SessionKey  string
	CreatedAt   time.Time
}

// AgentMailboxMessage is an append-only note to an execution agent (inter-agent / operator mail stub).
type AgentMailboxMessage struct {
	ID        string
	AgentID   string
	TaskID    TaskID // optional correlation
	Body      string
	CreatedAt time.Time
}
