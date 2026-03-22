-- Fix knowledge FTS triggers: INSERT … VALUES('delete', rowid) is only valid for
-- contentless/external-content FTS5 tables. Our index is a normal fts5 table; use DELETE instead.

DROP TRIGGER IF EXISTS knowledge_entries_ad;
DROP TRIGGER IF EXISTS knowledge_entries_au;

CREATE TRIGGER knowledge_entries_ad AFTER DELETE ON knowledge_entries BEGIN
  DELETE FROM knowledge_fts WHERE rowid = old.id;
END;

CREATE TRIGGER knowledge_entries_au AFTER UPDATE OF content ON knowledge_entries BEGIN
  DELETE FROM knowledge_fts WHERE rowid = old.id;
  INSERT INTO knowledge_fts(rowid, content) VALUES (new.id, new.content);
END;

UPDATE arms_schema_version SET version = 28 WHERE singleton = 1;
