-- Real merge ship metadata: PR fields on tasks, policy JSON on products, leases + outcomes on merge queue.

ALTER TABLE products ADD COLUMN merge_policy_json TEXT NOT NULL DEFAULT '{}';

ALTER TABLE tasks ADD COLUMN pull_request_url TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN pull_request_number INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN pull_request_head_branch TEXT NOT NULL DEFAULT '';

ALTER TABLE workspace_merge_queue ADD COLUMN lease_owner TEXT NOT NULL DEFAULT '';
ALTER TABLE workspace_merge_queue ADD COLUMN lease_expires_at TEXT DEFAULT NULL;
ALTER TABLE workspace_merge_queue ADD COLUMN merge_ship_state TEXT NOT NULL DEFAULT '';
ALTER TABLE workspace_merge_queue ADD COLUMN merged_sha TEXT NOT NULL DEFAULT '';
ALTER TABLE workspace_merge_queue ADD COLUMN merge_error TEXT NOT NULL DEFAULT '';
ALTER TABLE workspace_merge_queue ADD COLUMN conflict_files_json TEXT NOT NULL DEFAULT '';

UPDATE arms_schema_version SET version = 10 WHERE singleton = 1;
