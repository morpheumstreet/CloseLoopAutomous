-- Mission Control–style idea scoring and metadata (parity with MC `ideas` table).

ALTER TABLE ideas ADD COLUMN research_cycle_id TEXT REFERENCES research_cycles(id);
ALTER TABLE ideas ADD COLUMN category TEXT NOT NULL DEFAULT 'feature';
ALTER TABLE ideas ADD COLUMN research_backing TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN impact_score REAL;
ALTER TABLE ideas ADD COLUMN feasibility_score REAL;
ALTER TABLE ideas ADD COLUMN complexity TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN estimated_effort_hours REAL;
ALTER TABLE ideas ADD COLUMN competitive_analysis TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN target_user_segment TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN revenue_potential TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN technical_approach TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN risks TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN tags TEXT NOT NULL DEFAULT '[]';
ALTER TABLE ideas ADD COLUMN source TEXT NOT NULL DEFAULT 'research';
ALTER TABLE ideas ADD COLUMN source_research TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE ideas ADD COLUMN swiped_at TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN task_id TEXT REFERENCES tasks(id);
ALTER TABLE ideas ADD COLUMN user_notes TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN resurfaced_from TEXT REFERENCES ideas(id);
ALTER TABLE ideas ADD COLUMN resurfaced_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE ideas ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';

-- Backfill scores + workflow status from legacy columns.
UPDATE ideas SET
  impact_score = impact,
  feasibility_score = feasibility,
  research_backing = reasoning,
  status = CASE
    WHEN decided = 0 THEN 'pending'
    WHEN decision = 0 THEN 'rejected'
    WHEN decision = 1 THEN 'maybe'
    WHEN decision = 2 THEN 'approved'
    WHEN decision = 3 THEN 'approved'
    ELSE 'pending'
  END,
  updated_at = created_at;

UPDATE arms_schema_version SET version = 19 WHERE singleton = 1;
