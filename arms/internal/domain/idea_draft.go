package domain

// IdeaDraft is produced by ideation before persistence assigns IdeaID.
type IdeaDraft struct {
	Title       string
	Description string
	Impact      float64
	Feasibility float64
	Reasoning   string

	Category            string
	ResearchBacking     string
	Complexity          string
	EstimatedEffortHours float64
	CompetitiveAnalysis string
	TargetUserSegment   string
	RevenuePotential    string
	TechnicalApproach   string
	Risks               string
	Tags                []string
	SourceResearch      string
}
