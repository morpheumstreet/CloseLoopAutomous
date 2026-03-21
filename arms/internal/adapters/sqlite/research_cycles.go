package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type ResearchCycleStore struct{ db *sql.DB }

func NewResearchCycleStore(db *sql.DB) *ResearchCycleStore { return &ResearchCycleStore{db: db} }

var _ ports.ResearchCycleRepository = (*ResearchCycleStore)(nil)

func (s *ResearchCycleStore) Append(ctx context.Context, id string, productID domain.ProductID, summary string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO research_cycles (id, product_id, summary_snapshot, created_at) VALUES (?, ?, ?, ?)`,
		id, string(productID), summary, at.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *ResearchCycleStore) ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.ResearchCycle, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, product_id, summary_snapshot, created_at FROM research_cycles
WHERE product_id = ? ORDER BY created_at DESC LIMIT ?`,
		string(productID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ResearchCycle
	for rows.Next() {
		var r domain.ResearchCycle
		var pid, cat string
		if err := rows.Scan(&r.ID, &pid, &r.SummarySnapshot, &cat); err != nil {
			return nil, err
		}
		r.ProductID = domain.ProductID(pid)
		t, err := time.Parse(time.RFC3339Nano, cat)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, cat)
		}
		r.CreatedAt = t.UTC()
		out = append(out, r)
	}
	return out, rows.Err()
}
