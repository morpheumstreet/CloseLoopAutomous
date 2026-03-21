package domain

import "time"

// CostEvent accumulates spend for budget guardrails.
type CostEvent struct {
	ID        string
	ProductID ProductID
	TaskID    TaskID
	Amount    float64
	Note      string
	Agent     string
	Model     string
	At        time.Time
}

// ProductCostCaps optional per-product limits (nil pointer field = no limit for that axis).
type ProductCostCaps struct {
	ProductID     ProductID
	DailyCap      *float64
	MonthlyCap    *float64
	CumulativeCap *float64
}
