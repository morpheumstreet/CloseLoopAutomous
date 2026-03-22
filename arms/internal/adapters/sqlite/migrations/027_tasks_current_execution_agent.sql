-- #107: bind in-flight tasks to execution_agents for auto-reassign policy.

ALTER TABLE tasks ADD COLUMN current_execution_agent_id TEXT NOT NULL DEFAULT '';

UPDATE arms_schema_version SET version = 27 WHERE singleton = 1;
