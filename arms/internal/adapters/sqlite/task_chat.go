package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type TaskChatStore struct{ db *sql.DB }

func NewTaskChatStore(db *sql.DB) *TaskChatStore { return &TaskChatStore{db: db} }

var _ ports.TaskChatRepository = (*TaskChatStore)(nil)

func (s *TaskChatStore) Append(ctx context.Context, m *domain.TaskChatMessage) error {
	atStr := m.CreatedAt.UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO task_chat_messages (id, product_id, task_id, author, body, queue_pending, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.ID, string(m.ProductID), string(m.TaskID), m.Author, m.Body, boolInt(m.QueuePending), atStr)
	return err
}

func (s *TaskChatStore) ByID(ctx context.Context, id string) (*domain.TaskChatMessage, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, product_id, task_id, author, body, queue_pending, created_at
FROM task_chat_messages WHERE id = ?`, id)
	return scanTaskChatRow(row)
}

func scanTaskChatRow(row *sql.Row) (*domain.TaskChatMessage, error) {
	var m domain.TaskChatMessage
	var pid, tid, atStr string
	var qp int
	if err := row.Scan(&m.ID, &pid, &tid, &m.Author, &m.Body, &qp, &atStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	m.ProductID = domain.ProductID(pid)
	m.TaskID = domain.TaskID(tid)
	m.QueuePending = qp != 0
	t, perr := time.Parse(time.RFC3339Nano, atStr)
	if perr != nil {
		t, _ = time.Parse(time.RFC3339, atStr)
	}
	m.CreatedAt = t.UTC()
	return &m, nil
}

func (s *TaskChatStore) ListByTask(ctx context.Context, taskID domain.TaskID, limit int) ([]domain.TaskChatMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, product_id, task_id, author, body, queue_pending, created_at
FROM task_chat_messages WHERE task_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`,
		string(taskID), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var list []domain.TaskChatMessage
	for rows.Next() {
		var m domain.TaskChatMessage
		var pid, tid, atStr string
		var qp int
		if err := rows.Scan(&m.ID, &pid, &tid, &m.Author, &m.Body, &qp, &atStr); err != nil {
			return nil, err
		}
		m.ProductID = domain.ProductID(pid)
		m.TaskID = domain.TaskID(tid)
		m.QueuePending = qp != 0
		t, perr := time.Parse(time.RFC3339Nano, atStr)
		if perr != nil {
			t, _ = time.Parse(time.RFC3339, atStr)
		}
		m.CreatedAt = t.UTC()
		list = append(list, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// chronological (oldest first) for chat UI
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, nil
}

func (s *TaskChatStore) ListPendingQueueByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.TaskChatMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, product_id, task_id, author, body, queue_pending, created_at
FROM task_chat_messages
WHERE product_id = ? AND queue_pending = 1
ORDER BY created_at ASC, id ASC LIMIT ?`,
		string(productID), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var list []domain.TaskChatMessage
	for rows.Next() {
		var m domain.TaskChatMessage
		var pid, tid, atStr string
		var qp int
		if err := rows.Scan(&m.ID, &pid, &tid, &m.Author, &m.Body, &qp, &atStr); err != nil {
			return nil, err
		}
		m.ProductID = domain.ProductID(pid)
		m.TaskID = domain.TaskID(tid)
		m.QueuePending = qp != 0
		t, perr := time.Parse(time.RFC3339Nano, atStr)
		if perr != nil {
			t, _ = time.Parse(time.RFC3339, atStr)
		}
		m.CreatedAt = t.UTC()
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *TaskChatStore) ClearQueuePending(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE task_chat_messages SET queue_pending = 0 WHERE id = ? AND queue_pending = 1`, id)
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
