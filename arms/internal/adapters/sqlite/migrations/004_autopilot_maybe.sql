-- Autopilot cadence, tier, preference stub JSON, maybe pool (Mission Control–style §5).

ALTER TABLE products ADD COLUMN research_cadence_sec INTEGER NOT NULL DEFAULT 0;
ALTER TABLE products ADD COLUMN ideation_cadence_sec INTEGER NOT NULL DEFAULT 0;
ALTER TABLE products ADD COLUMN automation_tier TEXT NOT NULL DEFAULT 'supervised';
ALTER TABLE products ADD COLUMN auto_dispatch_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE products ADD COLUMN last_auto_research_at TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN last_auto_ideation_at TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN preference_model_json TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS maybe_pool (
  idea_id TEXT PRIMARY KEY REFERENCES ideas(id) ON DELETE CASCADE,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_maybe_pool_product ON maybe_pool(product_id);

UPDATE arms_schema_version SET version = 4 WHERE singleton = 1;
