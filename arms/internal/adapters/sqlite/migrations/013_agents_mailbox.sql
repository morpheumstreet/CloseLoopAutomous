-- Execution agent registry + mailbox (Mission Control–style agent slots; not task heartbeats).

CREATE TABLE IF NOT EXISTS execution_agents (
  id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL DEFAULT '',
  product_id TEXT REFERENCES products(id) ON DELETE SET NULL,
  source TEXT NOT NULL DEFAULT 'manual',
  external_ref TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_execution_agents_product ON execution_agents(product_id);

CREATE TABLE IF NOT EXISTS agent_mailbox (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL REFERENCES execution_agents(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_mailbox_agent ON agent_mailbox(agent_id, created_at DESC);

UPDATE arms_schema_version SET version = 13 WHERE singleton = 1;
