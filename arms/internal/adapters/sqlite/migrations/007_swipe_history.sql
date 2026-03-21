-- Auditable swipe log (Mission Control–style swipe_history).

CREATE TABLE IF NOT EXISTS swipe_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  idea_id TEXT NOT NULL REFERENCES ideas(id) ON DELETE CASCADE,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  decision TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_swipe_history_product ON swipe_history(product_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_swipe_history_idea ON swipe_history(idea_id, id DESC);

UPDATE arms_schema_version SET version = 7 WHERE singleton = 1;
