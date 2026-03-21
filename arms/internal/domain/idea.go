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

// Idea is a scored feature candidate before human swipe.
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
}
