package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type ProductScheduleStore struct{ db *sql.DB }

func NewProductScheduleStore(db *sql.DB) *ProductScheduleStore { return &ProductScheduleStore{db: db} }

var _ ports.ProductScheduleRepository = (*ProductScheduleStore)(nil)

func parseScheduleTime(s sql.NullString) (*time.Time, error) {
	if !s.Valid || s.String == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s.String)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s.String)
		if err != nil {
			return nil, err
		}
	}
	utc := t.UTC()
	return &utc, nil
}

func scanProductSchedule(row scanner) (*domain.ProductSchedule, error) {
	var pid, spec, cronExpr, asynqID string
	var en, delay int
	var lastS, nextS, atStr sql.NullString
	if err := row.Scan(&pid, &en, &spec, &cronExpr, &delay, &asynqID, &lastS, &nextS, &atStr); err != nil {
		return nil, err
	}
	lastAt, err := parseScheduleTime(lastS)
	if err != nil {
		return nil, err
	}
	nextAt, err := parseScheduleTime(nextS)
	if err != nil {
		return nil, err
	}
	if !atStr.Valid || atStr.String == "" {
		return &domain.ProductSchedule{
			ProductID:       domain.ProductID(pid),
			Enabled:         en != 0,
			SpecJSON:        spec,
			CronExpr:        cronExpr,
			DelaySeconds:    delay,
			AsynqTaskID:     asynqID,
			LastEnqueuedAt:  lastAt,
			NextScheduledAt: nextAt,
			UpdatedAt:       time.Time{},
		}, nil
	}
	t, perr := time.Parse(time.RFC3339Nano, atStr.String)
	if perr != nil {
		t, _ = time.Parse(time.RFC3339, atStr.String)
	}
	return &domain.ProductSchedule{
		ProductID:       domain.ProductID(pid),
		Enabled:         en != 0,
		SpecJSON:        spec,
		CronExpr:        cronExpr,
		DelaySeconds:    delay,
		AsynqTaskID:     asynqID,
		LastEnqueuedAt:  lastAt,
		NextScheduledAt: nextAt,
		UpdatedAt:       t.UTC(),
	}, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *ProductScheduleStore) Get(ctx context.Context, productID domain.ProductID) (*domain.ProductSchedule, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT product_id, enabled, spec_json, IFNULL(cron_expr,''), IFNULL(delay_seconds,0), IFNULL(asynq_task_id,''),
       last_enqueued_at, next_scheduled_at, updated_at
FROM product_schedules WHERE product_id = ?`, string(productID))
	out, err := scanProductSchedule(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

func sqlNullableTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func (s *ProductScheduleStore) Upsert(ctx context.Context, row *domain.ProductSchedule) error {
	atStr := row.UpdatedAt.UTC().Format(time.RFC3339Nano)
	en := 0
	if row.Enabled {
		en = 1
	}
	spec := row.SpecJSON
	if spec == "" {
		spec = "{}"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO product_schedules (
  product_id, enabled, spec_json, updated_at,
  cron_expr, delay_seconds, asynq_task_id, last_enqueued_at, next_scheduled_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(product_id) DO UPDATE SET
  enabled = excluded.enabled,
  spec_json = excluded.spec_json,
  updated_at = excluded.updated_at,
  cron_expr = excluded.cron_expr,
  delay_seconds = excluded.delay_seconds,
  asynq_task_id = excluded.asynq_task_id,
  last_enqueued_at = excluded.last_enqueued_at,
  next_scheduled_at = excluded.next_scheduled_at
`, string(row.ProductID), en, spec, atStr,
		nullIfEmpty(row.CronExpr), row.DelaySeconds, nullIfEmpty(row.AsynqTaskID),
		sqlNullableTimePtr(row.LastEnqueuedAt), sqlNullableTimePtr(row.NextScheduledAt))
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *ProductScheduleStore) ListEnabled(ctx context.Context) ([]domain.ProductSchedule, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT product_id, enabled, spec_json, IFNULL(cron_expr,''), IFNULL(delay_seconds,0), IFNULL(asynq_task_id,''),
       last_enqueued_at, next_scheduled_at, updated_at
FROM product_schedules WHERE enabled = 1 ORDER BY product_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.ProductSchedule
	for rows.Next() {
		r, err := scanProductSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}
