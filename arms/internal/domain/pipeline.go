package domain

// PipelineStage mirrors the autopilot phases described in the design doc.
type PipelineStage int

const (
	StageResearch PipelineStage = iota
	StageIdeation
	StageSwipe
	StagePlanning
	StageExecution
	StageReview
	StageShipped
)

func (s PipelineStage) String() string {
	switch s {
	case StageResearch:
		return "research"
	case StageIdeation:
		return "ideation"
	case StageSwipe:
		return "swipe"
	case StagePlanning:
		return "planning"
	case StageExecution:
		return "execution"
	case StageReview:
		return "review"
	case StageShipped:
		return "shipped"
	default:
		return "unknown"
	}
}
