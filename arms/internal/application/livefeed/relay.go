package livefeed

import (
	"context"
	"encoding/json"
	"time"

	"github.com/closeloopautomous/arms/internal/ports"
)

// RunOutboxRelay polls the outbox and publishes to the hub until ctx is cancelled.
func RunOutboxRelay(ctx context.Context, outbox ports.EventOutbox, hub *Hub, interval time.Duration) {
	if interval < 10*time.Millisecond {
		interval = 50 * time.Millisecond
	}
	flushOutbox(ctx, outbox, hub)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			flushOutbox(ctx, outbox, hub)
		}
	}
}

func flushOutbox(ctx context.Context, outbox ports.EventOutbox, hub *Hub) {
	entries, err := outbox.Pending(ctx, 100)
	if err != nil || len(entries) == 0 {
		return
	}
	for _, e := range entries {
		var ev ports.LiveActivityEvent
		if json.Unmarshal(e.Payload, &ev) == nil && ev.Type != "" {
			hub.BroadcastRaw(e.Payload)
		}
		_ = outbox.MarkDelivered(ctx, e.ID)
	}
}
