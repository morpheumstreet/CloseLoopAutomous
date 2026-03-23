-- Multi-endpoint agent gateway: connection profiles + execution_agents.session_key

CREATE TABLE IF NOT EXISTS gateway_endpoints (
  id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL DEFAULT '',
  driver TEXT NOT NULL,
  gateway_url TEXT NOT NULL DEFAULT '',
  gateway_token TEXT NOT NULL DEFAULT '',
  device_id TEXT NOT NULL DEFAULT '',
  timeout_sec INTEGER NOT NULL DEFAULT 0,
  product_id TEXT REFERENCES products(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gateway_endpoints_product ON gateway_endpoints(product_id);

INSERT OR IGNORE INTO gateway_endpoints (id, display_name, driver, gateway_url, gateway_token, device_id, timeout_sec, product_id, created_at)
VALUES ('gw-stub', 'Default stub', 'stub', '', '', '', 0, NULL, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));

ALTER TABLE execution_agents ADD COLUMN endpoint_id TEXT REFERENCES gateway_endpoints(id);
ALTER TABLE execution_agents ADD COLUMN session_key TEXT NOT NULL DEFAULT '';

UPDATE execution_agents SET endpoint_id = 'gw-stub' WHERE endpoint_id IS NULL;

UPDATE arms_schema_version SET version = 30 WHERE singleton = 1;
