package domain

import "time"

// ResearchCycle is one persisted research run snapshot for a product (history beyond Product.ResearchSummary).
type ResearchCycle struct {
	ID              string
	ProductID       ProductID
	SummarySnapshot string
	CreatedAt       time.Time
}
