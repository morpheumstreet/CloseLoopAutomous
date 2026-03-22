-- Per-task operator/agent chat + queued operator notes (Mission Control–style operator loop).

CREATE TABLE IF NOT EXISTS task_chat_messages (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  author TEXT NOT NULL DEFAULT 'operator',
  body TEXT NOT NULL,
  queue_pending INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_chat_task_created ON task_chat_messages(task_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_chat_product_queue ON task_chat_messages(product_id, queue_pending, created_at ASC);

UPDATE arms_schema_version SET version = 23 WHERE singleton = 1;
