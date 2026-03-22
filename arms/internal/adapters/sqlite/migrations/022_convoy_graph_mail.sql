-- Convoy graph metadata (JSON blobs + topological layer per subtask) and richer convoy_mail.

ALTER TABLE convoys ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}';

ALTER TABLE convoy_subtasks ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE convoy_subtasks ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE convoy_subtasks ADD COLUMN dag_layer INTEGER NOT NULL DEFAULT 0;

ALTER TABLE convoy_mail ADD COLUMN kind TEXT NOT NULL DEFAULT 'note';
ALTER TABLE convoy_mail ADD COLUMN from_subtask_id TEXT NOT NULL DEFAULT '';
ALTER TABLE convoy_mail ADD COLUMN to_subtask_id TEXT NOT NULL DEFAULT '';

UPDATE convoy_mail SET from_subtask_id = subtask_id WHERE trim(from_subtask_id) = '';

UPDATE arms_schema_version SET version = 22 WHERE singleton = 1;
