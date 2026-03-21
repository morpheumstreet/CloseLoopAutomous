package ports

import "github.com/closeloopautomous/arms/internal/domain"

// IdentityGenerator issues opaque IDs for aggregates (DB or UUID adapter).
type IdentityGenerator interface {
	NewProductID() domain.ProductID
	NewIdeaID() domain.IdeaID
	NewTaskID() domain.TaskID
	NewConvoyID() domain.ConvoyID
	NewSubtaskID() domain.SubtaskID
	NewCostEventID() string
}
