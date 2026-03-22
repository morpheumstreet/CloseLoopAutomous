package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// KnowledgeStore is SQLite + FTS5 for product knowledge (#90).
type KnowledgeStore struct{ db *sql.DB }

// NewKnowledgeStore returns a FTS-backed knowledge repository.
func NewKnowledgeStore(db *sql.DB) *KnowledgeStore { return &KnowledgeStore{db: db} }

var _ ports.KnowledgeRepository = (*KnowledgeStore)(nil)

func (s *KnowledgeStore) Create(ctx context.Context, e *domain.KnowledgeEntry) error {
	if e == nil {
		return domain.ErrInvalidInput
	}
	meta := strings.TrimSpace(e.MetadataJSON)
	if meta == "" {
		meta = "{}"
	}
	tid := strings.TrimSpace(string(e.TaskID))
	res, err := s.db.ExecContext(ctx, `
INSERT INTO knowledge_entries (product_id, task_id, content, metadata_json, created_at, updated_at)
VALUES (?, NULLIF(?, ''), ?, ?, ?, ?)`,
		string(e.ProductID), tid, e.Content, meta, e.CreatedAt.UTC().Format(time.RFC3339Nano), e.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

func (s *KnowledgeStore) Update(ctx context.Context, id int64, productID domain.ProductID, content string, metadataJSON string, at time.Time) error {
	meta := strings.TrimSpace(metadataJSON)
	if meta == "" {
		meta = "{}"
	}
	r, err := s.db.ExecContext(ctx, `
UPDATE knowledge_entries SET content = ?, metadata_json = ?, updated_at = ?
WHERE id = ? AND product_id = ?`,
		content, meta, at.UTC().Format(time.RFC3339Nano), id, string(productID))
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *KnowledgeStore) Delete(ctx context.Context, id int64, productID domain.ProductID) error {
	r, err := s.db.ExecContext(ctx, `DELETE FROM knowledge_entries WHERE id = ? AND product_id = ?`, id, string(productID))
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *KnowledgeStore) ByID(ctx context.Context, id int64, productID domain.ProductID) (*domain.KnowledgeEntry, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, product_id, IFNULL(task_id, ''), content, metadata_json, created_at, updated_at
FROM knowledge_entries WHERE id = ? AND product_id = ?`, id, string(productID))
	return scanKnowledgeEntry(row)
}

func (s *KnowledgeStore) ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.KnowledgeEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, product_id, IFNULL(task_id, ''), content, metadata_json, created_at, updated_at
FROM knowledge_entries WHERE product_id = ? ORDER BY updated_at DESC LIMIT ?`,
		string(productID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeRows(rows)
}

func (s *KnowledgeStore) Search(ctx context.Context, productID domain.ProductID, ftsQuery string, limit int) ([]domain.KnowledgeEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	q := strings.TrimSpace(ftsQuery)
	if q == "" {
		return s.ListByProduct(ctx, productID, limit)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT e.id, e.product_id, IFNULL(e.task_id, ''), e.content, e.metadata_json, e.created_at, e.updated_at
FROM knowledge_entries e
INNER JOIN knowledge_fts fts ON fts.rowid = e.id
WHERE e.product_id = ? AND fts MATCH ?
ORDER BY e.updated_at DESC
LIMIT ?`,
		string(productID), q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeRows(rows)
}

func scanKnowledgeEntry(row *sql.Row) (*domain.KnowledgeEntry, error) {
	var e domain.KnowledgeEntry
	var pid, tid string
	var cas, uas string
	if err := row.Scan(&e.ID, &pid, &tid, &e.Content, &e.MetadataJSON, &cas, &uas); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	e.ProductID = domain.ProductID(pid)
	e.TaskID = domain.TaskID(tid)
	ct, err1 := time.Parse(time.RFC3339Nano, cas)
	if err1 != nil {
		ct, _ = time.Parse(time.RFC3339, cas)
	}
	ut, err2 := time.Parse(time.RFC3339Nano, uas)
	if err2 != nil {
		ut, _ = time.Parse(time.RFC3339, uas)
	}
	e.CreatedAt = ct.UTC()
	e.UpdatedAt = ut.UTC()
	return &e, nil
}

func scanKnowledgeRows(rows *sql.Rows) ([]domain.KnowledgeEntry, error) {
	var out []domain.KnowledgeEntry
	for rows.Next() {
		var e domain.KnowledgeEntry
		var pid, tid string
		var cas, uas string
		if err := rows.Scan(&e.ID, &pid, &tid, &e.Content, &e.MetadataJSON, &cas, &uas); err != nil {
			return nil, err
		}
		e.ProductID = domain.ProductID(pid)
		e.TaskID = domain.TaskID(tid)
		ct, _ := time.Parse(time.RFC3339Nano, cas)
		if ct.IsZero() {
			ct, _ = time.Parse(time.RFC3339, cas)
		}
		ut, _ := time.Parse(time.RFC3339Nano, uas)
		if ut.IsZero() {
			ut, _ = time.Parse(time.RFC3339, uas)
		}
		e.CreatedAt = ct.UTC()
		e.UpdatedAt = ut.UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}
