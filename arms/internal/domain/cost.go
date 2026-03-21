package domain

import "time"

// CostEvent accumulates spend for budget guardrails.
type CostEvent struct {
	ID        string
	ProductID ProductID
	TaskID    TaskID
	Amount    float64
	Note      string
	At        time.Time
}
