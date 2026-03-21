package ports

import (
	"context"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

// LiveActivityEvent is the JSON shape for SSE / future WebSocket activity feeds.
type LiveActivityEvent struct {
	Type      string         `json:"type"`
	Ts        string         `json:"ts"`
	ProductID string         `json:"product_id,omitempty"`
	TaskID    string         `json:"task_id,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// LiveActivityPublisher emits activity for realtime subscribers (in-memory hub and/or outbox).
type LiveActivityPublisher interface {
	Publish(ctx context.Context, ev LiveActivityEvent) error
}

// ActivityStream exposes SSE (or WS) subscribers; implemented by [livefeed.Hub].
type ActivityStream interface {
	Subscribe() (ch <-chan []byte, unsub func())
}

// EventOutbox persists activity for relay to subscribers (SQLite). Same payload JSON as SSE line body.
type EventOutbox interface {
	Append(ctx context.Context, payloadJSON []byte) error
	Pending(ctx context.Context, limit int) ([]OutboxEntry, error)
	MarkDelivered(ctx context.Context, id int64) error
}

type OutboxEntry struct {
	ID      int64
	Payload []byte
}

// LiveActivityTX persists domain writes and the matching live-activity payload in one DB transaction (SQLite).
type LiveActivityTX interface {
	SaveTaskWithEvent(ctx context.Context, t *domain.Task, ev LiveActivityEvent) error
	RecordCheckpointWithEvent(ctx context.Context, taskID domain.TaskID, checkpointPayload string, t *domain.Task, ev LiveActivityEvent) error
	AppendCostWithEvent(ctx context.Context, e domain.CostEvent, ev LiveActivityEvent) error
	// CompleteTaskWithEvent sets the task to done (when allowed), upserts task_agent_health, and appends the outbox row in one transaction.
	// Idempotent when the task is already done (still upserts health + outbox).
	CompleteTaskWithEvent(ctx context.Context, taskID domain.TaskID, at time.Time, healthStatus, healthDetailJSON string, ev LiveActivityEvent) error
}
