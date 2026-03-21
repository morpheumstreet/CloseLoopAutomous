package livefeed

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/closeloopautomous/arms/internal/ports"
)

// Hub fans out LiveActivityEvent to SSE subscribers. Safe for concurrent Publish and Subscribe.
type Hub struct {
	mu   sync.RWMutex
	subs map[int]chan []byte
	next int
}

func NewHub() *Hub {
	return &Hub{subs: make(map[int]chan []byte)}
}

var (
	_ ports.LiveActivityPublisher = (*Hub)(nil)
	_ ports.ActivityStream      = (*Hub)(nil)
)

// Publish marshals the event and broadcasts to all subscribers (non-blocking per subscriber).
func (h *Hub) Publish(ctx context.Context, ev ports.LiveActivityEvent) error {
	_ = ctx
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	h.broadcast(b)
	return nil
}

func (h *Hub) broadcast(b []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs {
		select {
		case ch <- b:
		default:
		}
	}
}

// Subscribe returns a receive-only channel of raw JSON payloads (one SSE data line body per message).
func (h *Hub) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 64)
	h.mu.Lock()
	id := h.next
	h.next++
	h.subs[id] = ch
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
		close(ch)
	}
}

// BroadcastRaw sends an already-marshaled JSON payload (e.g. from outbox relay after validation).
func (h *Hub) BroadcastRaw(b []byte) {
	h.broadcast(b)
}
