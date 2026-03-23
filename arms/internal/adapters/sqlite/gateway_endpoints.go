package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type GatewayEndpointStore struct{ db *sql.DB }

func NewGatewayEndpointStore(db *sql.DB) *GatewayEndpointStore { return &GatewayEndpointStore{db: db} }

var _ ports.GatewayEndpointRegistry = (*GatewayEndpointStore)(nil)

func (s *GatewayEndpointStore) Save(ctx context.Context, e *domain.GatewayEndpoint) error {
	pid := sql.NullString{}
	if e.ProductID != "" {
		pid.String = string(e.ProductID)
		pid.Valid = true
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO gateway_endpoints (id, display_name, driver, gateway_url, gateway_token, device_id, timeout_sec, product_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.DisplayName, e.Driver, e.GatewayURL, e.GatewayToken, e.DeviceID, e.TimeoutSec, pid, e.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *GatewayEndpointStore) ByID(ctx context.Context, id string) (*domain.GatewayEndpoint, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, display_name, driver, gateway_url, gateway_token, device_id, timeout_sec, product_id, created_at
FROM gateway_endpoints WHERE id = ?`, id)
	var e domain.GatewayEndpoint
	var pid sql.NullString
	var cat string
	if err := row.Scan(&e.ID, &e.DisplayName, &e.Driver, &e.GatewayURL, &e.GatewayToken, &e.DeviceID, &e.TimeoutSec, &pid, &cat); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if pid.Valid {
		e.ProductID = domain.ProductID(pid.String)
	}
	t, err := time.Parse(time.RFC3339Nano, cat)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, cat)
	}
	e.CreatedAt = t.UTC()
	return &e, nil
}

func (s *GatewayEndpointStore) List(ctx context.Context, limit int) ([]domain.GatewayEndpoint, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, display_name, driver, gateway_url, gateway_token, device_id, timeout_sec, product_id, created_at
FROM gateway_endpoints ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGatewayEndpoints(rows)
}

func (s *GatewayEndpointStore) Update(ctx context.Context, e *domain.GatewayEndpoint) error {
	pid := sql.NullString{}
	if e.ProductID != "" {
		pid.String = string(e.ProductID)
		pid.Valid = true
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE gateway_endpoints SET display_name=?, driver=?, gateway_url=?, gateway_token=?, device_id=?, timeout_sec=?, product_id=?
WHERE id=?`,
		e.DisplayName, e.Driver, e.GatewayURL, e.GatewayToken, e.DeviceID, e.TimeoutSec, pid, e.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *GatewayEndpointStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM gateway_endpoints WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanGatewayEndpoints(rows *sql.Rows) ([]domain.GatewayEndpoint, error) {
	var out []domain.GatewayEndpoint
	for rows.Next() {
		var e domain.GatewayEndpoint
		var pid sql.NullString
		var cat string
		if err := rows.Scan(&e.ID, &e.DisplayName, &e.Driver, &e.GatewayURL, &e.GatewayToken, &e.DeviceID, &e.TimeoutSec, &pid, &cat); err != nil {
			return nil, err
		}
		if pid.Valid {
			e.ProductID = domain.ProductID(pid.String)
		}
		t, err := time.Parse(time.RFC3339Nano, cat)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, cat)
		}
		e.CreatedAt = t.UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}
