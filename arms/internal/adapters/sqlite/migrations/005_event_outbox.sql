-- Transactional outbox for live activity (relay → SSE / future WS).

CREATE TABLE IF NOT EXISTS event_outbox (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  delivered_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_event_outbox_pending ON event_outbox(id) WHERE delivered_at IS NULL;

UPDATE arms_schema_version SET version = 5 WHERE singleton = 1;
