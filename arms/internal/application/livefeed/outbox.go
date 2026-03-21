package livefeed

import (
	"context"
	"encoding/json"

	"github.com/closeloopautomous/arms/internal/ports"
)

// OutboxPublisher appends JSON events for [RunOutboxRelay] to deliver to a [Hub].
type OutboxPublisher struct {
	Outbox ports.EventOutbox
}

var _ ports.LiveActivityPublisher = (*OutboxPublisher)(nil)

func (p *OutboxPublisher) Publish(ctx context.Context, ev ports.LiveActivityEvent) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return p.Outbox.Append(ctx, b)
}
