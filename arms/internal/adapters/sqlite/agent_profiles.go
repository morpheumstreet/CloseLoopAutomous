package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type AgentProfileStore struct{ db *sql.DB }

func NewAgentProfileStore(db *sql.DB) *AgentProfileStore { return &AgentProfileStore{db: db} }

var _ ports.AgentProfileRepository = (*AgentProfileStore)(nil)

func (s *AgentProfileStore) Upsert(ctx context.Context, gatewayID string, ident *domain.AgentIdentity) error {
	if ident == nil {
		return nil
	}
	b, err := json.Marshal(ident)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	status := string(ident.Status)
	gwURL := ident.GatewayURL
	// Preserve created_at on update.
	var created string
	err = s.db.QueryRowContext(ctx, `SELECT created_at FROM agent_profiles WHERE id = ?`, ident.ID).Scan(&created)
	if err == sql.ErrNoRows || created == "" {
		created = now
	} else if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO agent_profiles (id, gateway_id, gateway_url, identity_json, status, last_updated, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  gateway_id = excluded.gateway_id,
  gateway_url = excluded.gateway_url,
  identity_json = excluded.identity_json,
  status = excluded.status,
  last_updated = excluded.last_updated`, ident.ID, gatewayID, gwURL, string(b), status, now, created)
	return err
}

func (s *AgentProfileStore) ByID(ctx context.Context, id string) (*domain.AgentIdentity, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT identity_json FROM agent_profiles WHERE id = ?`, id).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var ident domain.AgentIdentity
	if err := json.Unmarshal([]byte(raw), &ident); err != nil {
		return nil, err
	}
	return &ident, nil
}

func (s *AgentProfileStore) List(ctx context.Context, limit int) ([]domain.AgentIdentity, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT identity_json FROM agent_profiles ORDER BY last_updated DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AgentIdentity
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var ident domain.AgentIdentity
		if err := json.Unmarshal([]byte(raw), &ident); err != nil {
			return nil, err
		}
		out = append(out, ident)
	}
	return out, rows.Err()
}
