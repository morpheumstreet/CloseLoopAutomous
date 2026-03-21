-- Arms schema v1 — core aggregates (ids map to domain string IDs; enums stored as INTEGER).

CREATE TABLE IF NOT EXISTS products (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  stage INTEGER NOT NULL,
  research_summary TEXT NOT NULL DEFAULT '',
  workspace_id TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ideas (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  impact REAL NOT NULL DEFAULT 0,
  feasibility REAL NOT NULL DEFAULT 0,
  reasoning TEXT NOT NULL DEFAULT '',
  decided INTEGER NOT NULL DEFAULT 0,
  decision INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  idea_id TEXT NOT NULL REFERENCES ideas(id) ON DELETE CASCADE,
  spec TEXT NOT NULL DEFAULT '',
  status INTEGER NOT NULL,
  checkpoint TEXT NOT NULL DEFAULT '',
  external_ref TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS convoys (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  parent_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS convoy_subtasks (
  convoy_id TEXT NOT NULL REFERENCES convoys(id) ON DELETE CASCADE,
  id TEXT NOT NULL,
  agent_role TEXT NOT NULL,
  depends_on_json TEXT NOT NULL DEFAULT '[]',
  dispatched INTEGER NOT NULL DEFAULT 0,
  external_ref TEXT NOT NULL DEFAULT '',
  last_checkpoint TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (convoy_id, id)
);

CREATE INDEX IF NOT EXISTS idx_ideas_product ON ideas(product_id);
CREATE INDEX IF NOT EXISTS idx_tasks_product ON tasks(product_id);
CREATE INDEX IF NOT EXISTS idx_convoys_parent ON convoys(parent_task_id);

CREATE TABLE IF NOT EXISTS cost_events (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL,
  amount REAL NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cost_events_product ON cost_events(product_id);

CREATE TABLE IF NOT EXISTS checkpoints (
  task_id TEXT PRIMARY KEY,
  payload TEXT NOT NULL
);

UPDATE arms_schema_version SET version = 1 WHERE singleton = 1;
