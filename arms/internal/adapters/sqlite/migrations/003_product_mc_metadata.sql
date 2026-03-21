-- Mission Control–oriented product metadata (repo, program doc, settings blob, icon).

ALTER TABLE products ADD COLUMN repo_url TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN repo_branch TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN program_document TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN settings_json TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN icon_url TEXT NOT NULL DEFAULT '';

UPDATE arms_schema_version SET version = 3 WHERE singleton = 1;
