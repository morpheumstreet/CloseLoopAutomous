-- Optional team-facing statements (Mission page / charter context).
ALTER TABLE products ADD COLUMN mission_statement TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN vision_statement TEXT NOT NULL DEFAULT '';

UPDATE arms_schema_version SET version = 29 WHERE singleton = 1;
