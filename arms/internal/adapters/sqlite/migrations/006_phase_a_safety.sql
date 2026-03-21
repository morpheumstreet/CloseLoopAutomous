-- Phase A: cost dimensions + caps, checkpoint history, workspace ports + merge queue skeleton.

ALTER TABLE cost_events ADD COLUMN agent TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_events ADD COLUMN model TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS cost_caps (
  product_id TEXT PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
  daily_cap REAL,
  monthly_cap REAL,
  cumulative_cap REAL
);

CREATE TABLE IF NOT EXISTS checkpoint_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_checkpoint_history_task ON checkpoint_history(task_id, id DESC);

CREATE TABLE IF NOT EXISTS workspace_ports (
  port INTEGER PRIMARY KEY CHECK (port >= 4200 AND port <= 4299),
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL,
  allocated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workspace_ports_product ON workspace_ports(product_id);

CREATE TABLE IF NOT EXISTS workspace_merge_queue (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workspace_merge_queue_product ON workspace_merge_queue(product_id, status);

UPDATE arms_schema_version SET version = 6 WHERE singleton = 1;
