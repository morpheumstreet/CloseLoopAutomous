package domain

import "time"

// SwipeHistoryEntry is one recorded swipe decision (audit / preference pipelines).
type SwipeHistoryEntry struct {
	ID        int64
	IdeaID    IdeaID
	ProductID ProductID
	Decision  string // pass | maybe | yes | now
	CreatedAt time.Time
}

// SwipeDecisionKey returns the stable string stored in APIs and swipe_history.decision.
func SwipeDecisionKey(d SwipeDecision) string {
	switch d {
	case DecisionPass:
		return "pass"
	case DecisionMaybe:
		return "maybe"
	case DecisionYes:
		return "yes"
	case DecisionNow:
		return "now"
	default:
		return ""
	}
}
