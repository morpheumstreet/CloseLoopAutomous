-- Subtask completion (DAG): dependents dispatch only after upstream subtasks complete.

ALTER TABLE convoy_subtasks ADD COLUMN completed INTEGER NOT NULL DEFAULT 0;

UPDATE arms_schema_version SET version = 11 WHERE singleton = 1;
