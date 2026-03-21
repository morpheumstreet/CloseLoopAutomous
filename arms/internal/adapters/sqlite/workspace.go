package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

const workspacePortMin, workspacePortMax = 4200, 4299

// WorkspaceStore implements port allocation and merge-queue metrics.
type WorkspaceStore struct{ db *sql.DB }

func NewWorkspaceStore(db *sql.DB) *WorkspaceStore { return &WorkspaceStore{db: db} }

var (
	_ ports.WorkspacePortRepository       = (*WorkspaceStore)(nil)
	_ ports.WorkspaceMergeQueueRepository = (*WorkspaceStore)(nil)
)

func (s *WorkspaceStore) Allocate(ctx context.Context, productID domain.ProductID, taskID domain.TaskID, at time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT port FROM workspace_ports`)
	if err != nil {
		return 0, err
	}
	used := make(map[int]bool)
	for rows.Next() {
		var p int
		if err := rows.Scan(&p); err != nil {
			_ = rows.Close()
			return 0, err
		}
		used[p] = true
	}
	_ = rows.Close()
	for p := workspacePortMin; p <= workspacePortMax; p++ {
		if used[p] {
			continue
		}
		_, err := s.db.ExecContext(ctx, `
INSERT INTO workspace_ports (port, product_id, task_id, allocated_at) VALUES (?, ?, ?, ?)
`, p, string(productID), string(taskID), at.UTC().Format(time.RFC3339Nano))
		if err == nil {
			return p, nil
		}
	}
	return 0, errors.New("no free workspace port in range 4200-4299")
}

func (s *WorkspaceStore) Release(ctx context.Context, port int) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM workspace_ports WHERE port = ?`, port)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *WorkspaceStore) ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.AllocatedPort, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT port, product_id, task_id, allocated_at FROM workspace_ports WHERE product_id = ? ORDER BY port`, string(productID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAllocatedPorts(rows)
}

func (s *WorkspaceStore) ListAll(ctx context.Context) ([]domain.AllocatedPort, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT port, product_id, task_id, allocated_at FROM workspace_ports ORDER BY port`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAllocatedPorts(rows)
}

func scanAllocatedPorts(rows *sql.Rows) ([]domain.AllocatedPort, error) {
	var out []domain.AllocatedPort
	for rows.Next() {
		var a domain.AllocatedPort
		var pid, tid, ats string
		if err := rows.Scan(&a.Port, &pid, &tid, &ats); err != nil {
			return nil, err
		}
		a.ProductID = domain.ProductID(pid)
		a.TaskID = domain.TaskID(tid)
		t, err := time.Parse(time.RFC3339Nano, ats)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, ats)
		}
		a.AllocatedAt = t
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *WorkspaceStore) CountPending(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM workspace_merge_queue WHERE status = 'pending'`).Scan(&n)
	return n, err
}

func (s *WorkspaceStore) CountPendingByProduct(ctx context.Context, productID domain.ProductID) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM workspace_merge_queue WHERE product_id = ? AND status = 'pending'`,
		string(productID),
	).Scan(&n)
	return n, err
}

func (s *WorkspaceStore) Enqueue(ctx context.Context, productID domain.ProductID, taskID domain.TaskID, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var dup int64
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM workspace_merge_queue WHERE task_id = ? AND status = 'pending'`,
		string(taskID),
	).Scan(&dup)
	if err != nil {
		return err
	}
	if dup > 0 {
		return domain.ErrConflict
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO workspace_merge_queue (product_id, task_id, status, created_at) VALUES (?, ?, 'pending', ?)`,
		string(productID), string(taskID), at.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *WorkspaceStore) ListPendingByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.MergeQueueEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, product_id, task_id, status, created_at,
  lease_owner, lease_expires_at, merge_ship_state, merged_sha, merge_error, conflict_files_json
FROM workspace_merge_queue
WHERE product_id = ? AND status = 'pending' ORDER BY id ASC LIMIT ?`,
		string(productID), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MergeQueueEntry
	for rows.Next() {
		e, err := scanMergeQueueEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanMergeQueueEntry(rows *sql.Rows) (domain.MergeQueueEntry, error) {
	var e domain.MergeQueueEntry
	var pid, tid, sts, created string
	var leaseOwner, leaseExp, mss, msha, merr, cj sql.NullString
	if err := rows.Scan(&e.ID, &pid, &tid, &sts, &created,
		&leaseOwner, &leaseExp, &mss, &msha, &merr, &cj); err != nil {
		return e, err
	}
	e.ProductID = domain.ProductID(pid)
	e.TaskID = domain.TaskID(tid)
	e.Status = sts
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, created)
	}
	e.CreatedAt = t.UTC()
	e.LeaseOwner = leaseOwner.String
	if leaseExp.Valid && strings.TrimSpace(leaseExp.String) != "" {
		lt, err := time.Parse(time.RFC3339Nano, leaseExp.String)
		if err != nil {
			lt, _ = time.Parse(time.RFC3339, leaseExp.String)
		}
		e.LeaseExpiresAt = lt.UTC()
	}
	e.MergeShipState = domain.MergeShipState(mss.String)
	e.MergedSHA = msha.String
	e.MergeError = merr.String
	e.ConflictFilesJSON = cj.String
	return e, nil
}

func (s *WorkspaceStore) CompletePendingForTask(ctx context.Context, taskID domain.TaskID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var productID string
	err = tx.QueryRowContext(ctx, `SELECT product_id FROM tasks WHERE id = ?`, string(taskID)).Scan(&productID)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	var headID int64
	err = tx.QueryRowContext(ctx, `
SELECT id FROM workspace_merge_queue
WHERE product_id = ? AND status = 'pending' ORDER BY id ASC LIMIT 1`,
		productID,
	).Scan(&headID)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	var myID int64
	err = tx.QueryRowContext(ctx, `
SELECT id FROM workspace_merge_queue WHERE task_id = ? AND status = 'pending'`,
		string(taskID),
	).Scan(&myID)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	if myID != headID {
		return domain.ErrNotMergeQueueHead
	}
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx,
		`UPDATE workspace_merge_queue SET status = 'done', completed_at = ? WHERE id = ? AND status = 'pending'`,
		nowStr, myID,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *WorkspaceStore) CancelPendingForTask(ctx context.Context, taskID domain.TaskID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var productID string
	err = tx.QueryRowContext(ctx, `SELECT product_id FROM tasks WHERE id = ?`, string(taskID)).Scan(&productID)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	var myRowID int64
	err = tx.QueryRowContext(ctx, `
SELECT id FROM workspace_merge_queue WHERE task_id = ? AND status = 'pending'`,
		string(taskID),
	).Scan(&myRowID)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	var headID sql.NullInt64
	err = tx.QueryRowContext(ctx, `
SELECT id FROM workspace_merge_queue
WHERE product_id = ? AND status = 'pending' ORDER BY id ASC LIMIT 1`,
		productID,
	).Scan(&headID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if headID.Valid && headID.Int64 == myRowID {
		var leaseOwner, leaseExp sql.NullString
		if err := tx.QueryRowContext(ctx, `
SELECT lease_owner, lease_expires_at FROM workspace_merge_queue WHERE id = ?`, myRowID,
		).Scan(&leaseOwner, &leaseExp); err != nil {
			return err
		}
		owner := strings.TrimSpace(leaseOwner.String)
		if owner != "" && leaseExp.Valid && strings.TrimSpace(leaseExp.String) != "" {
			exp, expErr := time.Parse(time.RFC3339Nano, leaseExp.String)
			if expErr != nil {
				exp, expErr = time.Parse(time.RFC3339, leaseExp.String)
			}
			if expErr == nil && exp.After(time.Now().UTC()) {
				return domain.ErrMergeShipBusy
			}
		}
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM workspace_merge_queue WHERE id = ? AND status = 'pending'`, myRowID)
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
	return tx.Commit()
}

func (s *WorkspaceStore) ReserveHeadForShip(ctx context.Context, taskID domain.TaskID, leaseOwner string, leaseExpires time.Time) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var productID string
	err = tx.QueryRowContext(ctx, `SELECT product_id FROM tasks WHERE id = ?`, string(taskID)).Scan(&productID)
	if err == sql.ErrNoRows {
		return 0, domain.ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	var headID int64
	err = tx.QueryRowContext(ctx, `
SELECT id FROM workspace_merge_queue
WHERE product_id = ? AND status = 'pending' ORDER BY id ASC LIMIT 1`,
		productID,
	).Scan(&headID)
	if err == sql.ErrNoRows {
		return 0, domain.ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	var myID int64
	err = tx.QueryRowContext(ctx, `
SELECT id FROM workspace_merge_queue WHERE task_id = ? AND status = 'pending'`,
		string(taskID),
	).Scan(&myID)
	if err == sql.ErrNoRows {
		return 0, domain.ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if myID != headID {
		return 0, domain.ErrNotMergeQueueHead
	}
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	expStr := leaseExpires.UTC().Format(time.RFC3339Nano)
	res, err := tx.ExecContext(ctx, `
UPDATE workspace_merge_queue
SET lease_owner = ?, lease_expires_at = ?
WHERE id = ? AND status = 'pending'
  AND (IFNULL(lease_owner,'') = '' OR lease_expires_at IS NULL OR trim(lease_expires_at) = '' OR lease_expires_at < ?)`,
		strings.TrimSpace(leaseOwner), expStr, myID, nowStr,
	)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, domain.ErrMergeShipBusy
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return myID, nil
}

func (s *WorkspaceStore) FinishShip(ctx context.Context, rowID int64, leaseOwner string, result domain.MergeShipResult, shipOpErr error) error {
	r := result
	if errors.Is(shipOpErr, domain.ErrMergeConflict) && r.State == domain.MergeShipNone {
		r.State = domain.MergeShipConflict
		if strings.TrimSpace(r.ErrorMessage) == "" && shipOpErr != nil {
			r.ErrorMessage = shipOpErr.Error()
		}
	}
	if shipOpErr != nil && r.State == domain.MergeShipNone {
		r.State = domain.MergeShipFailed
		if strings.TrimSpace(r.ErrorMessage) == "" {
			r.ErrorMessage = shipOpErr.Error()
		}
	}
	cfj, _ := json.Marshal(r.ConflictFiles)
	if len(cfj) == 0 {
		cfj = []byte("[]")
	}
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var cur string
	err = tx.QueryRowContext(ctx, `SELECT lease_owner FROM workspace_merge_queue WHERE id = ?`, rowID).Scan(&cur)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(cur) != strings.TrimSpace(leaseOwner) {
		return domain.ErrMergeShipBusy
	}

	switch r.State {
	case domain.MergeShipMerged, domain.MergeShipSkipped:
		_, err = tx.ExecContext(ctx, `
UPDATE workspace_merge_queue SET
  status = 'done',
  completed_at = ?,
  merge_ship_state = ?,
  merged_sha = ?,
  merge_error = ?,
  conflict_files_json = ?,
  lease_owner = '',
  lease_expires_at = NULL
WHERE id = ? AND lease_owner = ?`,
			nowStr, string(r.State), strings.TrimSpace(r.MergedSHA), strings.TrimSpace(r.ErrorMessage), string(cfj),
			rowID, strings.TrimSpace(leaseOwner),
		)
	default:
		_, err = tx.ExecContext(ctx, `
UPDATE workspace_merge_queue SET
  merge_ship_state = ?,
  merged_sha = '',
  merge_error = ?,
  conflict_files_json = ?,
  lease_owner = '',
  lease_expires_at = NULL
WHERE id = ? AND lease_owner = ?`,
			string(r.State), strings.TrimSpace(r.ErrorMessage), string(cfj),
			rowID, strings.TrimSpace(leaseOwner),
		)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *WorkspaceStore) ReleaseShipLease(ctx context.Context, rowID int64, leaseOwner string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE workspace_merge_queue SET lease_owner = '', lease_expires_at = NULL
WHERE id = ? AND lease_owner = ?`,
		rowID, strings.TrimSpace(leaseOwner),
	)
	return err
}
