-- ResearchClaw (HTTP) hubs + system preference for autopilot research routing.

CREATE TABLE IF NOT EXISTS research_hubs (
  id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL DEFAULT '',
  base_url TEXT NOT NULL,
  api_key TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS research_system_settings (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  auto_research_claw_enabled INTEGER NOT NULL DEFAULT 0,
  default_research_hub_id TEXT REFERENCES research_hubs(id) ON DELETE SET NULL
);

INSERT OR IGNORE INTO research_system_settings (singleton, auto_research_claw_enabled, default_research_hub_id)
VALUES (1, 0, NULL);

UPDATE arms_schema_version SET version = 33 WHERE singleton = 1;
