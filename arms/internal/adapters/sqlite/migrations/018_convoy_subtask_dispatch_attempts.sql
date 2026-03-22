-- Track gateway dispatch attempts per convoy subtask (retry cap before hard failure).
ALTER TABLE convoy_subtasks ADD COLUMN dispatch_attempts INTEGER NOT NULL DEFAULT 0;

UPDATE arms_schema_version SET version = 18 WHERE singleton = 1;
