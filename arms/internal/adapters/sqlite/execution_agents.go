package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type ExecutionAgentStore struct{ db *sql.DB }

func NewExecutionAgentStore(db *sql.DB) *ExecutionAgentStore { return &ExecutionAgentStore{db: db} }

var _ ports.ExecutionAgentRegistry = (*ExecutionAgentStore)(nil)

func (s *ExecutionAgentStore) Save(ctx context.Context, a *domain.ExecutionAgent) error {
	pid := sql.NullString{}
	if a.ProductID != "" {
		pid.String = string(a.ProductID)
		pid.Valid = true
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO execution_agents (id, display_name, product_id, source, external_ref, endpoint_id, session_key, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.DisplayName, pid, a.Source, a.ExternalRef, a.EndpointID, a.SessionKey, a.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *ExecutionAgentStore) ByID(ctx context.Context, id string) (*domain.ExecutionAgent, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, display_name, product_id, source, external_ref, endpoint_id, session_key, created_at FROM execution_agents WHERE id = ?`, id)
	var a domain.ExecutionAgent
	var pid sql.NullString
	var eid sql.NullString
	var cat string
	if err := row.Scan(&a.ID, &a.DisplayName, &pid, &a.Source, &a.ExternalRef, &eid, &a.SessionKey, &cat); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if pid.Valid {
		a.ProductID = domain.ProductID(pid.String)
	}
	if eid.Valid {
		a.EndpointID = eid.String
	}
	t, err := time.Parse(time.RFC3339Nano, cat)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, cat)
	}
	a.CreatedAt = t.UTC()
	return &a, nil
}

func (s *ExecutionAgentStore) List(ctx context.Context, limit int) ([]domain.ExecutionAgent, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, display_name, product_id, source, external_ref, endpoint_id, session_key, created_at FROM execution_agents
ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExecutionAgents(rows)
}

func (s *ExecutionAgentStore) ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.ExecutionAgent, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	pid := string(productID)
	rows, err := s.db.QueryContext(ctx, `
SELECT id, display_name, product_id, source, external_ref, endpoint_id, session_key, created_at FROM execution_agents
WHERE product_id IS NULL OR product_id = ?
ORDER BY created_at ASC LIMIT ?`, pid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExecutionAgents(rows)
}

func scanExecutionAgents(rows *sql.Rows) ([]domain.ExecutionAgent, error) {
	var out []domain.ExecutionAgent
	for rows.Next() {
		var a domain.ExecutionAgent
		var pid sql.NullString
		var eid sql.NullString
		var cat string
		if err := rows.Scan(&a.ID, &a.DisplayName, &pid, &a.Source, &a.ExternalRef, &eid, &a.SessionKey, &cat); err != nil {
			return nil, err
		}
		if pid.Valid {
			a.ProductID = domain.ProductID(pid.String)
		}
		if eid.Valid {
			a.EndpointID = eid.String
		}
		t, err := time.Parse(time.RFC3339Nano, cat)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, cat)
		}
		a.CreatedAt = t.UTC()
		out = append(out, a)
	}
	return out, rows.Err()
}

type AgentMailboxStore struct{ db *sql.DB }

func NewAgentMailboxStore(db *sql.DB) *AgentMailboxStore { return &AgentMailboxStore{db: db} }

var _ ports.AgentMailboxRepository = (*AgentMailboxStore)(nil)

func (s *AgentMailboxStore) Append(ctx context.Context, id, agentID string, taskID domain.TaskID, body string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO agent_mailbox (id, agent_id, task_id, body, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, agentID, string(taskID), body, at.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *AgentMailboxStore) ListByAgent(ctx context.Context, agentID string, limit int) ([]domain.AgentMailboxMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, agent_id, task_id, body, created_at FROM agent_mailbox
WHERE agent_id = ? ORDER BY created_at DESC LIMIT ?`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AgentMailboxMessage
	for rows.Next() {
		var m domain.AgentMailboxMessage
		var tid, cat string
		if err := rows.Scan(&m.ID, &m.AgentID, &tid, &m.Body, &cat); err != nil {
			return nil, err
		}
		m.TaskID = domain.TaskID(tid)
		t, err := time.Parse(time.RFC3339Nano, cat)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, cat)
		}
		m.CreatedAt = t.UTC()
		out = append(out, m)
	}
	return out, rows.Err()
}