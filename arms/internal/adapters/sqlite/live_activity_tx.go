package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// LiveActivityTX implements [ports.LiveActivityTX] for same-transaction task/cost/checkpoint writes + event_outbox append.
type LiveActivityTX struct{ db *sql.DB }

func NewLiveActivityTX(db *sql.DB) *LiveActivityTX { return &LiveActivityTX{db: db} }

var _ ports.LiveActivityTX = (*LiveActivityTX)(nil)

func (x *LiveActivityTX) SaveTaskWithEvent(ctx context.Context, t *domain.Task, ev ports.LiveActivityEvent) error {
	tx, err := x.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := upsertTaskTx(ctx, tx, t); err != nil {
		return err
	}
	if err := appendOutboxTx(ctx, tx, ev); err != nil {
		return err
	}
	return tx.Commit()
}

func (x *LiveActivityTX) RecordCheckpointWithEvent(ctx context.Context, taskID domain.TaskID, checkpointPayload string, t *domain.Task, ev ports.LiveActivityEvent) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := x.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO checkpoint_history (task_id, payload, created_at) VALUES (?, ?, ?)
`, string(taskID), checkpointPayload, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO checkpoints (task_id, payload) VALUES (?, ?)
ON CONFLICT(task_id) DO UPDATE SET payload = excluded.payload
`, string(taskID), checkpointPayload); err != nil {
		return err
	}
	if err := upsertTaskTx(ctx, tx, t); err != nil {
		return err
	}
	if err := appendOutboxTx(ctx, tx, ev); err != nil {
		return err
	}
	return tx.Commit()
}

func (x *LiveActivityTX) AppendCostWithEvent(ctx context.Context, e domain.CostEvent, ev ports.LiveActivityEvent) error {
	tx, err := x.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO cost_events (id, product_id, task_id, amount, note, agent, model, at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, e.ID, string(e.ProductID), string(e.TaskID), e.Amount, e.Note, e.Agent, e.Model, e.At.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if err := appendOutboxTx(ctx, tx, ev); err != nil {
		return err
	}
	return tx.Commit()
}

func (x *LiveActivityTX) CompleteTaskWithEvent(ctx context.Context, taskID domain.TaskID, at time.Time, healthStatus, healthDetailJSON string, ev ports.LiveActivityEvent) error {
	atStr := at.UTC().Format(time.RFC3339Nano)
	if healthDetailJSON == "" {
		healthDetailJSON = "{}"
	}
	tx, err := x.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var productID string
	var st string
	err = tx.QueryRowContext(ctx, `SELECT product_id, status FROM tasks WHERE id = ?`, string(taskID)).Scan(&productID, &st)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}

	switch domain.TaskStatus(st) {
	case domain.StatusDone:
		// idempotent
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusReview:
		res, err := tx.ExecContext(ctx, `
UPDATE tasks SET status = ?, status_reason = '', updated_at = ?
WHERE id = ? AND status IN (?, ?, ?)`,
			string(domain.StatusDone), atStr, string(taskID),
			string(domain.StatusInProgress), string(domain.StatusTesting), string(domain.StatusReview),
		)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("%w: complete race", domain.ErrInvalidTransition)
		}
	default:
		return fmt.Errorf("%w: complete from %s", domain.ErrInvalidTransition, st)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO task_agent_health (task_id, product_id, status, detail_json, last_heartbeat_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(task_id) DO UPDATE SET
  product_id = excluded.product_id,
  status = excluded.status,
  detail_json = excluded.detail_json,
  last_heartbeat_at = excluded.last_heartbeat_at
`, string(taskID), productID, healthStatus, healthDetailJSON, atStr); err != nil {
		return err
	}
	if err := appendOutboxTx(ctx, tx, ev); err != nil {
		return err
	}
	return tx.Commit()
}

func upsertTaskTx(ctx context.Context, tx *sql.Tx, t *domain.Task) error {
	pa := 0
	if t.PlanApproved {
		pa = 1
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO tasks (id, product_id, idea_id, spec, status, status_reason, plan_approved, clarifications_json, checkpoint, external_ref, sandbox_path, worktree_path, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  product_id = excluded.product_id,
  idea_id = excluded.idea_id,
  spec = excluded.spec,
  status = excluded.status,
  status_reason = excluded.status_reason,
  plan_approved = excluded.plan_approved,
  clarifications_json = excluded.clarifications_json,
  checkpoint = excluded.checkpoint,
  external_ref = excluded.external_ref,
  sandbox_path = excluded.sandbox_path,
  worktree_path = excluded.worktree_path,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at
`, string(t.ID), string(t.ProductID), string(t.IdeaID), t.Spec, string(t.Status), t.StatusReason, pa, t.ClarificationsJSON,
		t.Checkpoint, t.ExternalRef, t.SandboxPath, t.WorktreePath,
		t.CreatedAt.Format(time.RFC3339Nano), t.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func appendOutboxTx(ctx context.Context, tx *sql.Tx, ev ports.LiveActivityEvent) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO event_outbox (payload_json, created_at, delivered_at)
VALUES (?, ?, NULL)
`, string(b), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
