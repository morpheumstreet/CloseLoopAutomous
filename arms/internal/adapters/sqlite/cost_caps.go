package sqlite

import (
	"context"
	"database/sql"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type CostCapStore struct{ db *sql.DB }

func NewCostCapStore(db *sql.DB) *CostCapStore { return &CostCapStore{db: db} }

var _ ports.CostCapRepository = (*CostCapStore)(nil)

func (s *CostCapStore) Get(ctx context.Context, productID domain.ProductID) (*domain.ProductCostCaps, error) {
	var daily, monthly, cum sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
SELECT daily_cap, monthly_cap, cumulative_cap FROM cost_caps WHERE product_id = ?`, string(productID)).Scan(&daily, &monthly, &cum)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c := &domain.ProductCostCaps{ProductID: productID}
	if daily.Valid {
		v := daily.Float64
		c.DailyCap = &v
	}
	if monthly.Valid {
		v := monthly.Float64
		c.MonthlyCap = &v
	}
	if cum.Valid {
		v := cum.Float64
		c.CumulativeCap = &v
	}
	return c, nil
}

func (s *CostCapStore) Upsert(ctx context.Context, caps *domain.ProductCostCaps) error {
	var d, m, c interface{}
	if caps.DailyCap != nil {
		d = *caps.DailyCap
	}
	if caps.MonthlyCap != nil {
		m = *caps.MonthlyCap
	}
	if caps.CumulativeCap != nil {
		c = *caps.CumulativeCap
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO cost_caps (product_id, daily_cap, monthly_cap, cumulative_cap) VALUES (?, ?, ?, ?)
ON CONFLICT(product_id) DO UPDATE SET
  daily_cap = excluded.daily_cap,
  monthly_cap = excluded.monthly_cap,
  cumulative_cap = excluded.cumulative_cap
`, string(caps.ProductID), d, m, c)
	return err
}
