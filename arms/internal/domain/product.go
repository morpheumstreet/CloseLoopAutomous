package domain

import "time"

// Product is the unit of work for research → ideation → tasks.
type Product struct {
	ID              ProductID
	Name            string
	Stage           PipelineStage
	ResearchSummary string
	UpdatedAt       time.Time
	WorkspaceID     string // logical isolation key (e.g. repo / worktree scope)

	// Extended fields (Mission Control–style product profile).
	RepoURL         string
	RepoBranch      string
	Description     string // short blurb
	ProgramDocument string // product program / charter (injected into research/ideation later)
	SettingsJSON    string // opaque JSON for product-scoped settings
	IconURL         string

	// Autopilot scheduling (§5): 0 = disabled for that phase.
	ResearchCadenceSec  int
	IdeationCadenceSec  int
	AutomationTier    AutomationTier
	AutoDispatchEnabled bool
	LastAutoResearchAt  time.Time
	LastAutoIdeationAt  time.Time
	// PreferenceModelJSON stores a JSON array of swipe events for downstream learning (stub aggregator).
	PreferenceModelJSON string
}
