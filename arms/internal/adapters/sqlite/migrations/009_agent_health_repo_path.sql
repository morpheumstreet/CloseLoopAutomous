-- Phase A: task agent heartbeats, local clone path for git worktrees, merge-queue completion timestamp.

CREATE TABLE IF NOT EXISTS task_agent_health (
  task_id TEXT NOT NULL PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'unknown',
  detail_json TEXT NOT NULL DEFAULT '{}',
  last_heartbeat_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_agent_health_product ON task_agent_health(product_id, last_heartbeat_at);

ALTER TABLE products ADD COLUMN repo_clone_path TEXT NOT NULL DEFAULT '';

ALTER TABLE workspace_merge_queue ADD COLUMN completed_at TEXT DEFAULT NULL;

UPDATE arms_schema_version SET version = 9 WHERE singleton = 1;
