package domain

import "time"

// SwipeDecision matches the Tinder-style decisions from the design.
type SwipeDecision int

const (
	DecisionPass SwipeDecision = iota
	DecisionMaybe
	DecisionYes
	DecisionNow
)

func (d SwipeDecision) Approved() bool {
	return d == DecisionYes || d == DecisionNow
}

// Idea is a scored feature candidate (Mission Control–style metadata + swipe workflow).
type Idea struct {
	ID          IdeaID
	ProductID   ProductID
	Title       string
	Description string
	Impact      float64
	Feasibility float64
	Reasoning   string
	Decided     bool
	Decision    SwipeDecision
	CreatedAt   time.Time
	UpdatedAt   time.Time

	ResearchCycleID string
	// Category is an MC enum (feature, improvement, ux, …); see AllowedIdeaCategory.
	Category            string
	ResearchBacking     string
	ImpactScore         float64
	FeasibilityScore    float64
	Complexity          string // S, M, L, XL or empty
	EstimatedEffortHours float64
	CompetitiveAnalysis string
	TargetUserSegment   string
	RevenuePotential    string
	TechnicalApproach   string
	Risks               string
	Tags                []string
	Source              string // research, manual, resurfaced, feedback
	SourceResearch      string
	// Status is MC workflow: pending, approved, rejected, maybe, building, built, shipped.
	Status   string
	SwipedAt time.Time
	// LinkedTaskID is set when a Kanban task is created from this idea (MC task_id).
	LinkedTaskID TaskID
	UserNotes    string
	ResurfacedFrom   IdeaID
	ResurfacedReason string
}
