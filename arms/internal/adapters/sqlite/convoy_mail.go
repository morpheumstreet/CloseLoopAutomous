package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type ConvoyMailStore struct{ db *sql.DB }

func NewConvoyMailStore(db *sql.DB) *ConvoyMailStore { return &ConvoyMailStore{db: db} }

var _ ports.ConvoyMailRepository = (*ConvoyMailStore)(nil)

func (s *ConvoyMailStore) Append(ctx context.Context, convoyID domain.ConvoyID, msg domain.ConvoyMailDraft, at time.Time) error {
	id := uuid.NewString()
	atStr := at.UTC().Format(time.RFC3339Nano)
	from := string(msg.FromSubtaskID)
	to := string(msg.ToSubtaskID)
	kind := domain.NormalizeConvoyMailKind(msg.Kind)
	// subtask_id column kept for backward compatibility and indexes — mirror sender.
	_, err := s.db.ExecContext(ctx, `
INSERT INTO convoy_mail (id, convoy_id, subtask_id, body, kind, from_subtask_id, to_subtask_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, string(convoyID), from, msg.Body, kind, from, to, atStr)
	return err
}

func (s *ConvoyMailStore) ListByConvoy(ctx context.Context, convoyID domain.ConvoyID, limit int) ([]domain.ConvoyMailMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, convoy_id, subtask_id, body, kind, from_subtask_id, to_subtask_id, created_at FROM convoy_mail
WHERE convoy_id = ? ORDER BY created_at DESC LIMIT ?`, string(convoyID), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.ConvoyMailMessage
	for rows.Next() {
		var m domain.ConvoyMailMessage
		var legacySub, kind, fromSt, toSt, atStr string
		if err := rows.Scan(&m.ID, &m.ConvoyID, &legacySub, &m.Body, &kind, &fromSt, &toSt, &atStr); err != nil {
			return nil, err
		}
		if strings.TrimSpace(fromSt) == "" {
			fromSt = legacySub
		}
		m.SubtaskID = domain.SubtaskID(fromSt)
		m.FromSubtaskID = domain.SubtaskID(fromSt)
		m.ToSubtaskID = domain.SubtaskID(toSt)
		m.Kind = domain.NormalizeConvoyMailKind(kind)
		t, perr := time.Parse(time.RFC3339Nano, atStr)
		if perr != nil {
			t, _ = time.Parse(time.RFC3339, atStr)
		}
		m.CreatedAt = t.UTC()
		out = append(out, m)
	}
	return out, rows.Err()
}
