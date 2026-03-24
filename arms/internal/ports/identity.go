package ports

import "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"

// IdentityGenerator issues opaque IDs for aggregates (DB or UUID adapter).
type IdentityGenerator interface {
	NewProductID() domain.ProductID
	NewIdeaID() domain.IdeaID
	NewTaskID() domain.TaskID
	NewConvoyID() domain.ConvoyID
	NewSubtaskID() domain.SubtaskID
	NewCostEventID() string
	NewResearchCycleID() string
	NewExecutionAgentID() string
	NewMailboxMessageID() string
	NewProductFeedbackID() string
	NewTaskChatMessageID() string
	NewGatewayEndpointID() string
}
