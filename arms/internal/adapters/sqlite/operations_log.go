package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type OperationsLogStore struct{ db *sql.DB }

func NewOperationsLogStore(db *sql.DB) *OperationsLogStore { return &OperationsLogStore{db: db} }

var _ ports.OperationsLogRepository = (*OperationsLogStore)(nil)

func (s *OperationsLogStore) Append(ctx context.Context, e domain.OperationLogEntry) error {
	at := e.CreatedAt
	if at.IsZero() {
		at = time.Now().UTC()
	}
	atStr := at.UTC().Format(time.RFC3339Nano)
	pid := string(e.ProductID)
	if pid == "" {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO operations_log (created_at, actor, action, resource_type, resource_id, detail_json, product_id)
VALUES (?, ?, ?, ?, ?, ?, NULL)`,
			atStr, e.Actor, e.Action, e.ResourceType, e.ResourceID, e.DetailJSON)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO operations_log (created_at, actor, action, resource_type, resource_id, detail_json, product_id)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		atStr, e.Actor, e.Action, e.ResourceType, e.ResourceID, e.DetailJSON, pid)
	return err
}

func (s *OperationsLogStore) List(ctx context.Context, f ports.OperationsLogFilter) ([]domain.OperationLogEntry, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	q := `
SELECT id, created_at, actor, action, resource_type, resource_id, detail_json, product_id
FROM operations_log WHERE 1=1`
	var args []any
	if f.ProductID != nil && *f.ProductID != "" {
		q += ` AND product_id = ?`
		args = append(args, string(*f.ProductID))
	}
	if a := strings.TrimSpace(f.Action); a != "" {
		q += ` AND action = ?`
		args = append(args, a)
	}
	if rt := strings.TrimSpace(f.ResourceType); rt != "" {
		q += ` AND resource_type = ?`
		args = append(args, rt)
	}
	if f.Since != nil {
		q += ` AND created_at >= ?`
		args = append(args, f.Since.UTC().Format(time.RFC3339Nano))
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.OperationLogEntry
	for rows.Next() {
		var e domain.OperationLogEntry
		var atStr string
		var pid sql.NullString
		if err := rows.Scan(&e.ID, &atStr, &e.Actor, &e.Action, &e.ResourceType, &e.ResourceID, &e.DetailJSON, &pid); err != nil {
			return nil, err
		}
		t, perr := time.Parse(time.RFC3339Nano, atStr)
		if perr != nil {
			t, _ = time.Parse(time.RFC3339, atStr)
		}
		e.CreatedAt = t.UTC()
		if pid.Valid {
			e.ProductID = domain.ProductID(pid.String)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
