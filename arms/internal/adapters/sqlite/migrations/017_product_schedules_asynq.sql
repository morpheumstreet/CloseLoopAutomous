-- Per-row Asynq scheduling: cron / one-shot delay + task metadata for visibility.

ALTER TABLE product_schedules ADD COLUMN cron_expr TEXT;
ALTER TABLE product_schedules ADD COLUMN delay_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE product_schedules ADD COLUMN asynq_task_id TEXT;
ALTER TABLE product_schedules ADD COLUMN last_enqueued_at TEXT;
ALTER TABLE product_schedules ADD COLUMN next_scheduled_at TEXT;

UPDATE arms_schema_version SET version = 17 WHERE singleton = 1;
