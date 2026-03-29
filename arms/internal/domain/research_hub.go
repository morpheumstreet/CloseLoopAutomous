package domain

import "time"

// ResearchHub is one ResearchClaw-compatible HTTP API root (see OpenAPI: /api/health, /api/pipeline/start, …).
type ResearchHub struct {
	ID          string
	DisplayName string
	BaseURL     string
	APIKey      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ResearchSystemSettings toggles autopilot research through a configured hub vs LLM/stub.
type ResearchSystemSettings struct {
	AutoResearchClawEnabled bool
	DefaultResearchHubID    string
}
