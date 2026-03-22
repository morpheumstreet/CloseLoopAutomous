-- Normalized convoy DAG edges for SQL analytics / MC-style queries (depends_on_json remains canonical for loads).
CREATE TABLE IF NOT EXISTS convoy_edges (
  convoy_id TEXT NOT NULL,
  from_subtask_id TEXT NOT NULL,
  to_subtask_id TEXT NOT NULL,
  PRIMARY KEY (convoy_id, from_subtask_id, to_subtask_id),
  FOREIGN KEY (convoy_id) REFERENCES convoys(id) ON DELETE CASCADE,
  FOREIGN KEY (convoy_id, from_subtask_id) REFERENCES convoy_subtasks(convoy_id, id) ON DELETE CASCADE,
  FOREIGN KEY (convoy_id, to_subtask_id) REFERENCES convoy_subtasks(convoy_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_convoy_edges_convoy_to ON convoy_edges (convoy_id, to_subtask_id);
CREATE INDEX IF NOT EXISTS idx_convoy_edges_convoy_from ON convoy_edges (convoy_id, from_subtask_id);

INSERT OR IGNORE INTO convoy_edges (convoy_id, from_subtask_id, to_subtask_id)
SELECT c.convoy_id, j.value, c.id
FROM convoy_subtasks c, json_each(c.depends_on_json) AS j
WHERE json_valid(c.depends_on_json)
  AND EXISTS (
    SELECT 1 FROM convoy_subtasks p
    WHERE p.convoy_id = c.convoy_id AND p.id = j.value
  );

UPDATE arms_schema_version SET version = 26 WHERE singleton = 1;
