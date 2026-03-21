package ports

import "context"

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
