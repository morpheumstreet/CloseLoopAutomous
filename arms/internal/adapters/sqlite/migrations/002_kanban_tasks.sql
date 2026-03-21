-- Mission Control–style Kanban: tasks.status as TEXT + planning / reason fields.

PRAGMA foreign_keys = OFF;

CREATE TABLE tasks_new (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  idea_id TEXT NOT NULL REFERENCES ideas(id) ON DELETE CASCADE,
  spec TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  status_reason TEXT NOT NULL DEFAULT '',
  plan_approved INTEGER NOT NULL DEFAULT 0,
  clarifications_json TEXT NOT NULL DEFAULT '',
  checkpoint TEXT NOT NULL DEFAULT '',
  external_ref TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

INSERT INTO tasks_new (
  id, product_id, idea_id, spec, status, status_reason, plan_approved, clarifications_json,
  checkpoint, external_ref, created_at, updated_at
)
SELECT
  id,
  product_id,
  idea_id,
  spec,
  CASE CAST(status AS INTEGER)
    WHEN 0 THEN 'inbox'
    WHEN 1 THEN 'planning'
    WHEN 2 THEN 'assigned'
    WHEN 3 THEN 'in_progress'
    WHEN 4 THEN 'in_progress'
    WHEN 5 THEN 'done'
    WHEN 6 THEN 'failed'
    ELSE 'planning'
  END,
  '',
  0,
  '',
  checkpoint,
  external_ref,
  created_at,
  updated_at
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE INDEX IF NOT EXISTS idx_tasks_product ON tasks(product_id);

PRAGMA foreign_keys = ON;

UPDATE arms_schema_version SET version = 2 WHERE singleton = 1;
