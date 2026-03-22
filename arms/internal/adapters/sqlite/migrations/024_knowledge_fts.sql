-- Product knowledge base + FTS5 index for dispatch-time retrieval (#90).

CREATE TABLE IF NOT EXISTS knowledge_entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  task_id TEXT,
  content TEXT NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_knowledge_entries_product ON knowledge_entries(product_id, updated_at DESC);

CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts USING fts5(
  content,
  tokenize = 'unicode61 remove_diacritics 1'
);

CREATE TRIGGER IF NOT EXISTS knowledge_entries_ai AFTER INSERT ON knowledge_entries BEGIN
  INSERT INTO knowledge_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS knowledge_entries_ad AFTER DELETE ON knowledge_entries BEGIN
  INSERT INTO knowledge_fts(knowledge_fts, rowid) VALUES('delete', old.id);
END;

CREATE TRIGGER IF NOT EXISTS knowledge_entries_au AFTER UPDATE OF content ON knowledge_entries BEGIN
  INSERT INTO knowledge_fts(knowledge_fts, rowid) VALUES('delete', old.id);
  INSERT INTO knowledge_fts(rowid, content) VALUES (new.id, new.content);
END;

UPDATE arms_schema_version SET version = 24 WHERE singleton = 1;
