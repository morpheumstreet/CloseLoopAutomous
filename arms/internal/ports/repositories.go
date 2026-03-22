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

// PreferenceModelRepository stores per-product learned preference payloads (separate from legacy products.preference_model_json).
type PreferenceModelRepository interface {
	Get(ctx context.Context, productID domain.ProductID) (modelJSON string, updatedAt time.Time, ok bool, err error)
	Upsert(ctx context.Context, productID domain.ProductID, modelJSON string, at time.Time) error
}

// OperationsLogFilter narrows audit log reads (all fields optional except Limit defaulting in stores).
type OperationsLogFilter struct {
	Limit        int
	ProductID    *domain.ProductID
	Action       string
	ResourceType string
	Since        *time.Time // inclusive lower bound on created_at (UTC)
}

// OperationsLogRepository is an append-only audit trail for operator-relevant actions.
type OperationsLogRepository interface {
	Append(ctx context.Context, e domain.OperationLogEntry) error
	List(ctx context.Context, f OperationsLogFilter) ([]domain.OperationLogEntry, error)
}

// ProductScheduleRepository stores per-product flags for autopilot tick eligibility (see product_schedules table).
// Get returns (nil, nil) when no row exists — callers treat that as “enabled” for backward compatibility.
type ProductScheduleRepository interface {
	Get(ctx context.Context, productID domain.ProductID) (*domain.ProductSchedule, error)
	Upsert(ctx context.Context, row *domain.ProductSchedule) error
	// ListEnabled returns rows with enabled = true (may be empty).
	ListEnabled(ctx context.Context) ([]domain.ProductSchedule, error)
}

// ConvoyMailRepository appends and lists messages for convoy subtasks (inter-subtask mail).
type ConvoyMailRepository interface {
	Append(ctx context.Context, convoyID domain.ConvoyID, msg domain.ConvoyMailDraft, at time.Time) error
	ListByConvoy(ctx context.Context, convoyID domain.ConvoyID, limit int) ([]domain.ConvoyMailMessage, error)
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
	ListEntriesByProduct(ctx context.Context, productID domain.ProductID) ([]domain.MaybePoolEntry, error)
	// ApplyBatchReevaluate sets last_evaluated_at=now, increments evaluation_count, appends note to evaluation_notes,
	// and sets next_evaluate_at from nextEval (zero time clears the field).
	ApplyBatchReevaluate(ctx context.Context, productID domain.ProductID, note string, nextEval time.Time, now time.Time) error
}

// ProductFeedbackRepository stores external feedback rows per product.
type ProductFeedbackRepository interface {
	Append(ctx context.Context, f *domain.ProductFeedback) error
	ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.ProductFeedback, error)
	ByID(ctx context.Context, id string) (*domain.ProductFeedback, error)
	SetProcessed(ctx context.Context, id string, processed bool) error
}

// TaskChatRepository stores per-task operator/agent chat and queued operator notes.
type TaskChatRepository interface {
	Append(ctx context.Context, m *domain.TaskChatMessage) error
	ByID(ctx context.Context, id string) (*domain.TaskChatMessage, error)
	ListByTask(ctx context.Context, taskID domain.TaskID, limit int) ([]domain.TaskChatMessage, error)
	ListPendingQueueByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.TaskChatMessage, error)
	ClearQueuePending(ctx context.Context, id string) error
}

// KnowledgeRepository stores product-scoped knowledge snippets with FTS5 search (SQLite) or in-memory scan.
type KnowledgeRepository interface {
	Create(ctx context.Context, e *domain.KnowledgeEntry) error
	Update(ctx context.Context, id int64, productID domain.ProductID, content string, metadataJSON string, at time.Time) error
	Delete(ctx context.Context, id int64, productID domain.ProductID) error
	ByID(ctx context.Context, id int64, productID domain.ProductID) (*domain.KnowledgeEntry, error)
	ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.KnowledgeEntry, error)
	Search(ctx context.Context, productID domain.ProductID, ftsQuery string, limit int) ([]domain.KnowledgeEntry, error)
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
	CountPendingByProduct(ctx context.Context, productID domain.ProductID) (int64, error)
	Enqueue(ctx context.Context, productID domain.ProductID, taskID domain.TaskID, at time.Time) error
	// GetPendingMergeQueueEntry returns a pending row by primary key, or ErrNotFound.
	GetPendingMergeQueueEntry(ctx context.Context, rowID int64) (*domain.MergeQueueEntry, error)
	ListPendingByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.MergeQueueEntry, error)
	// CompletePendingForTask marks the pending row for this task done (serialized lane advances).
	CompletePendingForTask(ctx context.Context, taskID domain.TaskID) error
	// CancelPendingForTask removes a pending row for this task. Fails with ErrMergeShipBusy if this task
	// is the queue head and holds an active merge ship lease. Non-head entries are always removable.
	CancelPendingForTask(ctx context.Context, taskID domain.TaskID) error

	// ReserveHeadForShip verifies this task owns the FIFO head and sets a lease (multi-instance safety).
	ReserveHeadForShip(ctx context.Context, taskID domain.TaskID, leaseOwner string, leaseExpires time.Time) (rowID int64, err error)
	// FinishShip records merge outcome; on merged/skipped marks the row done. Always clears the lease.
	FinishShip(ctx context.Context, rowID int64, leaseOwner string, result domain.MergeShipResult, shipOpErr error) error
	// ReleaseShipLease clears lease without changing queue position (e.g. panic recovery).
	ReleaseShipLease(ctx context.Context, rowID int64, leaseOwner string) error
}
