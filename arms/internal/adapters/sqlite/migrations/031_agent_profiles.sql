-- Unified agent identity cache (docs/scan-agents.md): synthesized from gateway_endpoints + optional GeoIP.

CREATE TABLE IF NOT EXISTS agent_profiles (
  id TEXT PRIMARY KEY,
  gateway_id TEXT NOT NULL REFERENCES gateway_endpoints(id) ON DELETE CASCADE,
  gateway_url TEXT NOT NULL DEFAULT '',
  identity_json TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'offline',
  last_updated TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_profiles_gateway_id ON agent_profiles(gateway_id);
CREATE INDEX IF NOT EXISTS idx_agent_profiles_status ON agent_profiles(status);

UPDATE arms_schema_version SET version = 31 WHERE singleton = 1;
