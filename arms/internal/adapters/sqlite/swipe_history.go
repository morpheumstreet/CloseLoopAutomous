package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

type SwipeHistoryStore struct {
	db *sql.DB
}

func NewSwipeHistoryStore(db *sql.DB) *SwipeHistoryStore {
	return &SwipeHistoryStore{db: db}
}

func (s *SwipeHistoryStore) Append(ctx context.Context, ideaID domain.IdeaID, productID domain.ProductID, decision string, at time.Time) error {
	if decision == "" {
		return domain.ErrInvalidInput
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO swipe_history (idea_id, product_id, decision, created_at) VALUES (?, ?, ?, ?)`,
		string(ideaID), string(productID), decision, at.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *SwipeHistoryStore) ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.SwipeHistoryEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, idea_id, product_id, decision, created_at FROM swipe_history WHERE product_id = ? ORDER BY id DESC LIMIT ?`,
		string(productID), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SwipeHistoryEntry
	for rows.Next() {
		var e domain.SwipeHistoryEntry
		var created string
		if err := rows.Scan(&e.ID, &e.IdeaID, &e.ProductID, &e.Decision, &created); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, created)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, created)
		}
		e.CreatedAt = t.UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}
