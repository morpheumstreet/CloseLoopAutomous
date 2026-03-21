package ports

import (
	"context"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

type ProductRepository interface {
	Save(ctx context.Context, p *domain.Product) error
	ByID(ctx context.Context, id domain.ProductID) (*domain.Product, error)
	ListAll(ctx context.Context) ([]domain.Product, error)
}

// SwipeHistoryRepository appends and lists human swipe decisions per product.
type SwipeHistoryRepository interface {
	Append(ctx context.Context, ideaID domain.IdeaID, productID domain.ProductID, decision string, at time.Time) error
	ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.SwipeHistoryEntry, error)
}

// MaybePoolRepository tracks ideas swiped "maybe" for later promotion (Mission Control–style).
type MaybePoolRepository interface {
	Add(ctx context.Context, ideaID domain.IdeaID, productID domain.ProductID, at time.Time) error
	Remove(ctx context.Context, ideaID domain.IdeaID) error
	ListIdeaIDsByProduct(ctx context.Context, productID domain.ProductID) ([]domain.IdeaID, error)
}

type IdeaRepository interface {
	Save(ctx context.Context, i *domain.Idea) error
	ByID(ctx context.Context, id domain.IdeaID) (*domain.Idea, error)
	ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Idea, error)
}

type TaskRepository interface {
	Save(ctx context.Context, t *domain.Task) error
	ByID(ctx context.Context, id domain.TaskID) (*domain.Task, error)
	ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Task, error)
	// TryComplete sets status to done when the task is in_progress, testing, or review.
	// Returns nil if the task is already done (idempotent). ErrNotFound if missing. ErrInvalidTransition otherwise.
	TryComplete(ctx context.Context, taskID domain.TaskID, at time.Time) error
}

type ConvoyRepository interface {
	Save(ctx context.Context, c *domain.Convoy) error
	ByID(ctx context.Context, id domain.ConvoyID) (*domain.Convoy, error)
	ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Convoy, error)
}

type CostRepository interface {
	Append(ctx context.Context, e domain.CostEvent) error
	SumByProduct(ctx context.Context, productID domain.ProductID) (float64, error)
	SumByProductSince(ctx context.Context, productID domain.ProductID, since time.Time) (float64, error)
	ListByProductBetween(ctx context.Context, productID domain.ProductID, from, to time.Time) ([]domain.CostEvent, error)
}

// CostCapRepository stores optional daily / monthly / cumulative caps per product.
type CostCapRepository interface {
	Get(ctx context.Context, productID domain.ProductID) (*domain.ProductCostCaps, error)
	Upsert(ctx context.Context, caps *domain.ProductCostCaps) error
}

type CheckpointRepository interface {
	Save(ctx context.Context, taskID domain.TaskID, payload string) error
	Load(ctx context.Context, taskID domain.TaskID) (string, error)
	ListHistory(ctx context.Context, taskID domain.TaskID, limit int) ([]domain.CheckpointHistoryEntry, error)
	HistoryByID(ctx context.Context, id int64) (*domain.CheckpointHistoryEntry, error)
}

// WorkspacePortRepository allocates TCP ports 4200–4299 per product/task.
type WorkspacePortRepository interface {
	Allocate(ctx context.Context, productID domain.ProductID, taskID domain.TaskID, at time.Time) (allocatedPort int, err error)
	Release(ctx context.Context, port int) error
	ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.AllocatedPort, error)
	ListAll(ctx context.Context) ([]domain.AllocatedPort, error)
}

// AgentHealthRepository stores last-seen heartbeats per task (Mission Control–style agent liveness).
type AgentHealthRepository interface {
	UpsertHeartbeat(ctx context.Context, taskID domain.TaskID, productID domain.ProductID, status, detailJSON string, at time.Time) error
	ByTask(ctx context.Context, taskID domain.TaskID) (*domain.TaskAgentHealth, error)
	ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.TaskAgentHealth, error)
	ListRecent(ctx context.Context, limit int) ([]domain.TaskAgentHealth, error)
}

// WorkspaceMergeQueue tracks serialized merge operations per product (FIFO pending list).
type WorkspaceMergeQueueRepository interface {
	CountPending(ctx context.Context) (int64, error)
	Enqueue(ctx context.Context, productID domain.ProductID, taskID domain.TaskID, at time.Time) error
	ListPendingByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.MergeQueueEntry, error)
	// CompletePendingForTask marks the pending row for this task done (serialized lane advances).
	CompletePendingForTask(ctx context.Context, taskID domain.TaskID) error
}
