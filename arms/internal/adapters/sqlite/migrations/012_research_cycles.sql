-- Append-only research history + placeholder for future Asynq-driven schedules.

CREATE TABLE IF NOT EXISTS research_cycles (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  summary_snapshot TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_research_cycles_product ON research_cycles(product_id, created_at DESC);

CREATE TABLE IF NOT EXISTS product_schedules (
  product_id TEXT PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
  enabled INTEGER NOT NULL DEFAULT 1,
  spec_json TEXT NOT NULL DEFAULT '{}',
  updated_at TEXT NOT NULL DEFAULT ''
);

UPDATE arms_schema_version SET version = 12 WHERE singleton = 1;
