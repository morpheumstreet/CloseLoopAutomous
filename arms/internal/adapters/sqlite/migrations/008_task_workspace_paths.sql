-- Phase A: optional sandbox / worktree path hints on tasks (metadata for isolation; no git binary calls).

ALTER TABLE tasks ADD COLUMN sandbox_path TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN worktree_path TEXT NOT NULL DEFAULT '';

UPDATE arms_schema_version SET version = 8 WHERE singleton = 1;
