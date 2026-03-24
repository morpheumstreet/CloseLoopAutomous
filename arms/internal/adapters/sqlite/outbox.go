package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type OutboxStore struct{ db *sql.DB }

func NewOutboxStore(db *sql.DB) *OutboxStore { return &OutboxStore{db: db} }

var _ ports.EventOutbox = (*OutboxStore)(nil)

func (s *OutboxStore) Append(ctx context.Context, payloadJSON []byte) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO event_outbox (payload_json, created_at, delivered_at)
VALUES (?, ?, NULL)
`, string(payloadJSON), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *OutboxStore) Pending(ctx context.Context, limit int) ([]ports.OutboxEntry, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, payload_json FROM event_outbox
WHERE delivered_at IS NULL
ORDER BY id ASC
LIMIT ?
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.OutboxEntry
	for rows.Next() {
		var id int64
		var payload string
		if err := rows.Scan(&id, &payload); err != nil {
			return nil, err
		}
		out = append(out, ports.OutboxEntry{ID: id, Payload: []byte(payload)})
	}
	return out, rows.Err()
}

func (s *OutboxStore) MarkDelivered(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE event_outbox SET delivered_at = ? WHERE id = ? AND delivered_at IS NULL
`, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}
