package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type ProductFeedbackStore struct{ db *sql.DB }

func NewProductFeedbackStore(db *sql.DB) *ProductFeedbackStore { return &ProductFeedbackStore{db: db} }

var _ ports.ProductFeedbackRepository = (*ProductFeedbackStore)(nil)

func (s *ProductFeedbackStore) Append(ctx context.Context, f *domain.ProductFeedback) error {
	iid := string(f.IdeaID)
	var ideaArg interface{}
	if iid != "" {
		ideaArg = iid
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO product_feedback (id, product_id, source, content, customer_id, category, sentiment, processed, idea_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, string(f.ProductID), f.Source, f.Content, f.CustomerID, f.Category, f.Sentiment, boolInt(f.Processed), ideaArg, f.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *ProductFeedbackStore) ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.ProductFeedback, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, product_id, source, content, customer_id, category, sentiment, processed, IFNULL(idea_id,''), created_at
FROM product_feedback WHERE product_id = ? ORDER BY created_at DESC LIMIT ?`,
		string(productID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProductFeedbackRows(rows)
}

func (s *ProductFeedbackStore) ByID(ctx context.Context, id string) (*domain.ProductFeedback, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, product_id, source, content, customer_id, category, sentiment, processed, IFNULL(idea_id,''), created_at
FROM product_feedback WHERE id = ?`, id)
	return scanProductFeedbackRow(row)
}

func (s *ProductFeedbackStore) SetProcessed(ctx context.Context, id string, processed bool) error {
	res, err := s.db.ExecContext(ctx, `UPDATE product_feedback SET processed = ? WHERE id = ?`, boolInt(processed), id)
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

func scanProductFeedbackRows(rows *sql.Rows) ([]domain.ProductFeedback, error) {
	var out []domain.ProductFeedback
	for rows.Next() {
		f, err := scanProductFeedbackRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func scanProductFeedbackRow(row interface {
	Scan(dest ...any) error
}) (*domain.ProductFeedback, error) {
	var (
		id, pid, src, content, cust, category, sentiment, iid, created string
		proc                                                         int
	)
	if err := row.Scan(&id, &pid, &src, &content, &cust, &category, &sentiment, &proc, &iid, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, created)
	}
	return &domain.ProductFeedback{
		ID:         id,
		ProductID:  domain.ProductID(pid),
		Source:     src,
		Content:    content,
		CustomerID: cust,
		Category:   category,
		Sentiment:  sentiment,
		Processed:  proc != 0,
		IdeaID:     domain.IdeaID(iid),
		CreatedAt:  t.UTC(),
	}, nil
}
