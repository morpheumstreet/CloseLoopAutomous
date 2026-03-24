package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type AgentHealthStore struct{ db *sql.DB }

func NewAgentHealthStore(db *sql.DB) *AgentHealthStore { return &AgentHealthStore{db: db} }

var _ ports.AgentHealthRepository = (*AgentHealthStore)(nil)

func (s *AgentHealthStore) UpsertHeartbeat(ctx context.Context, taskID domain.TaskID, productID domain.ProductID, status, detailJSON string, at time.Time) error {
	if detailJSON == "" {
		detailJSON = "{}"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO task_agent_health (task_id, product_id, status, detail_json, last_heartbeat_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(task_id) DO UPDATE SET
  product_id = excluded.product_id,
  status = excluded.status,
  detail_json = excluded.detail_json,
  last_heartbeat_at = excluded.last_heartbeat_at
`, string(taskID), string(productID), status, detailJSON, at.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *AgentHealthStore) ByTask(ctx context.Context, taskID domain.TaskID) (*domain.TaskAgentHealth, error) {
	var h domain.TaskAgentHealth
	var ats string
	err := s.db.QueryRowContext(ctx, `
SELECT task_id, product_id, status, detail_json, last_heartbeat_at FROM task_agent_health WHERE task_id = ?`,
		string(taskID),
	).Scan(&h.TaskID, &h.ProductID, &h.Status, &h.DetailJSON, &ats)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, ats)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, ats)
	}
	h.LastHeartbeatAt = t.UTC()
	return &h, nil
}

func (s *AgentHealthStore) ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.TaskAgentHealth, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT task_id, product_id, status, detail_json, last_heartbeat_at FROM task_agent_health
WHERE product_id = ? ORDER BY last_heartbeat_at DESC LIMIT ?`,
		string(productID), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAgentHealthRows(rows)
}

func (s *AgentHealthStore) ListRecent(ctx context.Context, limit int) ([]domain.TaskAgentHealth, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT task_id, product_id, status, detail_json, last_heartbeat_at FROM task_agent_health
ORDER BY last_heartbeat_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAgentHealthRows(rows)
}

func scanAgentHealthRows(rows *sql.Rows) ([]domain.TaskAgentHealth, error) {
	var out []domain.TaskAgentHealth
	for rows.Next() {
		var h domain.TaskAgentHealth
		var ats string
		if err := rows.Scan(&h.TaskID, &h.ProductID, &h.Status, &h.DetailJSON, &ats); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, ats)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, ats)
		}
		h.LastHeartbeatAt = t.UTC()
		out = append(out, h)
	}
	return out, rows.Err()
}
