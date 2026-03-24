package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type PreferenceModelStore struct{ db *sql.DB }

func NewPreferenceModelStore(db *sql.DB) *PreferenceModelStore { return &PreferenceModelStore{db: db} }

var _ ports.PreferenceModelRepository = (*PreferenceModelStore)(nil)

func (s *PreferenceModelStore) Get(ctx context.Context, productID domain.ProductID) (modelJSON string, updatedAt time.Time, ok bool, err error) {
	row := s.db.QueryRowContext(ctx, `
SELECT model_json, updated_at FROM preference_models WHERE product_id = ?`, string(productID))
	var mj, atStr string
	if err := row.Scan(&mj, &atStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", time.Time{}, false, nil
		}
		return "", time.Time{}, false, err
	}
	t, perr := time.Parse(time.RFC3339Nano, atStr)
	if perr != nil {
		t, _ = time.Parse(time.RFC3339, atStr)
	}
	return mj, t.UTC(), true, nil
}

func (s *PreferenceModelStore) Upsert(ctx context.Context, productID domain.ProductID, modelJSON string, at time.Time) error {
	atStr := at.UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO preference_models (product_id, model_json, updated_at) VALUES (?, ?, ?)
ON CONFLICT(product_id) DO UPDATE SET model_json = excluded.model_json, updated_at = excluded.updated_at
`, string(productID), modelJSON, atStr)
	return err
}
