package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// —— products ——

type ProductStore struct{ db *sql.DB }

func NewProductStore(db *sql.DB) *ProductStore { return &ProductStore{db: db} }

var _ ports.ProductRepository = (*ProductStore)(nil)

func (s *ProductStore) Save(ctx context.Context, p *domain.Product) error {
	tier := string(p.AutomationTier)
	if tier == "" {
		tier = string(domain.TierSupervised)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO products (
  id, name, stage, research_summary, workspace_id,
  repo_url, repo_branch, description, program_document, settings_json, icon_url,
  research_cadence_sec, ideation_cadence_sec, automation_tier, auto_dispatch_enabled,
  last_auto_research_at, last_auto_ideation_at, preference_model_json,
  updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  name = excluded.name,
  stage = excluded.stage,
  research_summary = excluded.research_summary,
  workspace_id = excluded.workspace_id,
  repo_url = excluded.repo_url,
  repo_branch = excluded.repo_branch,
  description = excluded.description,
  program_document = excluded.program_document,
  settings_json = excluded.settings_json,
  icon_url = excluded.icon_url,
  research_cadence_sec = excluded.research_cadence_sec,
  ideation_cadence_sec = excluded.ideation_cadence_sec,
  automation_tier = excluded.automation_tier,
  auto_dispatch_enabled = excluded.auto_dispatch_enabled,
  last_auto_research_at = excluded.last_auto_research_at,
  last_auto_ideation_at = excluded.last_auto_ideation_at,
  preference_model_json = excluded.preference_model_json,
  updated_at = excluded.updated_at
`, string(p.ID), p.Name, int(p.Stage), p.ResearchSummary, p.WorkspaceID,
		p.RepoURL, p.RepoBranch, p.Description, p.ProgramDocument, p.SettingsJSON, p.IconURL,
		p.ResearchCadenceSec, p.IdeationCadenceSec, tier, boolInt(p.AutoDispatchEnabled),
		formatOptionalTime(p.LastAutoResearchAt), formatOptionalTime(p.LastAutoIdeationAt), p.PreferenceModelJSON,
		p.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

const productSelectCols = `id, name, stage, research_summary, workspace_id,
  repo_url, repo_branch, description, program_document, settings_json, icon_url,
  research_cadence_sec, ideation_cadence_sec, automation_tier, auto_dispatch_enabled,
  last_auto_research_at, last_auto_ideation_at, preference_model_json,
  updated_at`

func (s *ProductStore) ByID(ctx context.Context, id domain.ProductID) (*domain.Product, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+productSelectCols+` FROM products WHERE id = ?`, string(id))
	p, err := scanProductRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *ProductStore) ListAll(ctx context.Context) ([]domain.Product, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+productSelectCols+` FROM products ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Product
	for rows.Next() {
		p, err := scanProductRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func scanProductRow(row interface{ Scan(dest ...any) error }) (*domain.Product, error) {
	var (
		sid, name, summary, ws                                  string
		repoURL, branch, desc, program, settings, icon, updated string
		tierStr, prefJSON                                       string
		lastRes, lastIde                                        string
		stage, resCad, ideCad, autoDisp                         int
	)
	if err := row.Scan(&sid, &name, &stage, &summary, &ws,
		&repoURL, &branch, &desc, &program, &settings, &icon,
		&resCad, &ideCad, &tierStr, &autoDisp, &lastRes, &lastIde, &prefJSON,
		&updated); err != nil {
		return nil, err
	}
	tier, err := domain.ParseAutomationTier(tierStr)
	if err != nil {
		tier = domain.TierSupervised
	}
	ut, err := time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		ut, _ = time.Parse(time.RFC3339, updated)
	}
	lr, _ := parseOptionalTime(lastRes)
	li, _ := parseOptionalTime(lastIde)
	return &domain.Product{
		ID:                  domain.ProductID(sid),
		Name:                name,
		Stage:               domain.PipelineStage(stage),
		ResearchSummary:     summary,
		WorkspaceID:         ws,
		RepoURL:             repoURL,
		RepoBranch:          branch,
		Description:         desc,
		ProgramDocument:     program,
		SettingsJSON:        settings,
		IconURL:             icon,
		ResearchCadenceSec:  resCad,
		IdeationCadenceSec:  ideCad,
		AutomationTier:      tier,
		AutoDispatchEnabled: autoDisp != 0,
		LastAutoResearchAt:  lr,
		LastAutoIdeationAt:  li,
		PreferenceModelJSON: prefJSON,
		UpdatedAt:           ut,
	}, nil
}

func formatOptionalTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

func parseOptionalTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Parse(time.RFC3339, s)
	}
	return t, nil
}

// —— maybe pool ——

type MaybePoolStore struct{ db *sql.DB }

func NewMaybePoolStore(db *sql.DB) *MaybePoolStore { return &MaybePoolStore{db: db} }

var _ ports.MaybePoolRepository = (*MaybePoolStore)(nil)

func (s *MaybePoolStore) Add(ctx context.Context, ideaID domain.IdeaID, productID domain.ProductID, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO maybe_pool (idea_id, product_id, created_at) VALUES (?, ?, ?)
ON CONFLICT(idea_id) DO UPDATE SET product_id = excluded.product_id, created_at = excluded.created_at
`, string(ideaID), string(productID), at.Format(time.RFC3339Nano))
	return err
}

func (s *MaybePoolStore) Remove(ctx context.Context, ideaID domain.IdeaID) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM maybe_pool WHERE idea_id = ?`, string(ideaID))
	return err
}

func (s *MaybePoolStore) ListIdeaIDsByProduct(ctx context.Context, productID domain.ProductID) ([]domain.IdeaID, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT idea_id FROM maybe_pool WHERE product_id = ? ORDER BY created_at ASC`, string(productID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.IdeaID
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, domain.IdeaID(id))
	}
	return out, rows.Err()
}

// —— ideas ——

type IdeaStore struct{ db *sql.DB }

func NewIdeaStore(db *sql.DB) *IdeaStore { return &IdeaStore{db: db} }

var _ ports.IdeaRepository = (*IdeaStore)(nil)

func (s *IdeaStore) Save(ctx context.Context, i *domain.Idea) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO ideas (id, product_id, title, description, impact, feasibility, reasoning, decided, decision, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  product_id = excluded.product_id,
  title = excluded.title,
  description = excluded.description,
  impact = excluded.impact,
  feasibility = excluded.feasibility,
  reasoning = excluded.reasoning,
  decided = excluded.decided,
  decision = excluded.decision,
  created_at = excluded.created_at
`, string(i.ID), string(i.ProductID), i.Title, i.Description, i.Impact, i.Feasibility, i.Reasoning, boolInt(i.Decided), int(i.Decision), i.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *IdeaStore) ByID(ctx context.Context, id domain.IdeaID) (*domain.Idea, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, product_id, title, description, impact, feasibility, reasoning, decided, decision, created_at
FROM ideas WHERE id = ?`, string(id))
	return scanIdea(row)
}

func (s *IdeaStore) ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Idea, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, product_id, title, description, impact, feasibility, reasoning, decided, decision, created_at
FROM ideas WHERE product_id = ? ORDER BY created_at ASC`, string(productID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Idea
	for rows.Next() {
		i, err := scanIdea(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *i)
	}
	return out, rows.Err()
}

func scanIdea(row interface{ Scan(dest ...any) error }) (*domain.Idea, error) {
	var (
		id, pid, title, desc, reasoning, created string
		impact, feas                             float64
		decided                                  int
		decision                                 int
	)
	if err := row.Scan(&id, &pid, &title, &desc, &impact, &feas, &reasoning, &decided, &decision, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, created)
	}
	return &domain.Idea{
		ID:          domain.IdeaID(id),
		ProductID:   domain.ProductID(pid),
		Title:       title,
		Description: desc,
		Impact:      impact,
		Feasibility: feas,
		Reasoning:   reasoning,
		Decided:     decided != 0,
		Decision:    domain.SwipeDecision(decision),
		CreatedAt:   t,
	}, nil
}

// —— tasks ——

type TaskStore struct{ db *sql.DB }

func NewTaskStore(db *sql.DB) *TaskStore { return &TaskStore{db: db} }

var _ ports.TaskRepository = (*TaskStore)(nil)

func (s *TaskStore) Save(ctx context.Context, t *domain.Task) error {
	pa := 0
	if t.PlanApproved {
		pa = 1
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO tasks (id, product_id, idea_id, spec, status, status_reason, plan_approved, clarifications_json, checkpoint, external_ref, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  product_id = excluded.product_id,
  idea_id = excluded.idea_id,
  spec = excluded.spec,
  status = excluded.status,
  status_reason = excluded.status_reason,
  plan_approved = excluded.plan_approved,
  clarifications_json = excluded.clarifications_json,
  checkpoint = excluded.checkpoint,
  external_ref = excluded.external_ref,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at
`, string(t.ID), string(t.ProductID), string(t.IdeaID), t.Spec, string(t.Status), t.StatusReason, pa, t.ClarificationsJSON,
		t.Checkpoint, t.ExternalRef,
		t.CreatedAt.Format(time.RFC3339Nano), t.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *TaskStore) ByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, product_id, idea_id, spec, status, status_reason, plan_approved, clarifications_json, checkpoint, external_ref, created_at, updated_at
FROM tasks WHERE id = ?`, string(id))
	t, err := scanTaskRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (s *TaskStore) ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, product_id, idea_id, spec, status, status_reason, plan_approved, clarifications_json, checkpoint, external_ref, created_at, updated_at
FROM tasks WHERE product_id = ?
ORDER BY updated_at DESC`, string(productID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Task
	for rows.Next() {
		t, err := scanTaskRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func scanTaskRow(row *sql.Row) (*domain.Task, error) {
	var (
		sid, pid, iid, spec, st, sreason, clar, ck, ref, ca, ua string
		pa                                                      int
	)
	if err := row.Scan(&sid, &pid, &iid, &spec, &st, &sreason, &pa, &clar, &ck, &ref, &ca, &ua); err != nil {
		return nil, err
	}
	return buildTaskFromScan(sid, pid, iid, spec, st, sreason, pa, clar, ck, ref, ca, ua)
}

func scanTaskRows(rows *sql.Rows) (*domain.Task, error) {
	var (
		sid, pid, iid, spec, st, sreason, clar, ck, ref, ca, ua string
		pa                                                      int
	)
	if err := rows.Scan(&sid, &pid, &iid, &spec, &st, &sreason, &pa, &clar, &ck, &ref, &ca, &ua); err != nil {
		return nil, err
	}
	return buildTaskFromScan(sid, pid, iid, spec, st, sreason, pa, clar, ck, ref, ca, ua)
}

func buildTaskFromScan(sid, pid, iid, spec, st, sreason string, pa int, clar, ck, ref, ca, ua string) (*domain.Task, error) {
	cat, err := time.Parse(time.RFC3339Nano, ca)
	if err != nil {
		cat, _ = time.Parse(time.RFC3339, ca)
	}
	uat, err := time.Parse(time.RFC3339Nano, ua)
	if err != nil {
		uat, _ = time.Parse(time.RFC3339, ua)
	}
	return &domain.Task{
		ID:                 domain.TaskID(sid),
		ProductID:          domain.ProductID(pid),
		IdeaID:             domain.IdeaID(iid),
		Spec:               spec,
		Status:             domain.TaskStatus(st),
		StatusReason:       sreason,
		PlanApproved:       pa != 0,
		ClarificationsJSON: clar,
		Checkpoint:         ck,
		ExternalRef:        ref,
		CreatedAt:          cat,
		UpdatedAt:          uat,
	}, nil
}

// —— convoys ——

type ConvoyStore struct{ db *sql.DB }

func NewConvoyStore(db *sql.DB) *ConvoyStore { return &ConvoyStore{db: db} }

var _ ports.ConvoyRepository = (*ConvoyStore)(nil)

func (s *ConvoyStore) Save(ctx context.Context, c *domain.Convoy) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
INSERT INTO convoys (id, product_id, parent_task_id, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  product_id = excluded.product_id,
  parent_task_id = excluded.parent_task_id,
  created_at = excluded.created_at
`, string(c.ID), string(c.ProductID), string(c.ParentID), c.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM convoy_subtasks WHERE convoy_id = ?`, string(c.ID)); err != nil {
		return err
	}
	for _, st := range c.Subtasks {
		deps, err := json.Marshal(depIDs(st.DependsOn))
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO convoy_subtasks (convoy_id, id, agent_role, depends_on_json, dispatched, external_ref, last_checkpoint)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, string(c.ID), string(st.ID), st.AgentRole, string(deps), boolInt(st.Dispatched), st.ExternalRef, st.LastCheckpoint)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func depIDs(ids []domain.SubtaskID) []string {
	out := make([]string, len(ids))
	for i := range ids {
		out[i] = string(ids[i])
	}
	return out
}

func (s *ConvoyStore) ByID(ctx context.Context, id domain.ConvoyID) (*domain.Convoy, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, product_id, parent_task_id, created_at FROM convoys WHERE id = ?`, string(id))
	var cid, pid, parent, cat string
	if err := row.Scan(&cid, &pid, &parent, &cat); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	ct, err := time.Parse(time.RFC3339Nano, cat)
	if err != nil {
		ct, _ = time.Parse(time.RFC3339, cat)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, agent_role, depends_on_json, dispatched, external_ref, last_checkpoint
FROM convoy_subtasks WHERE convoy_id = ? ORDER BY id`, string(id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []domain.Subtask
	for rows.Next() {
		var sid, role, depj, xref, lcp string
		var disp int
		if err := rows.Scan(&sid, &role, &depj, &disp, &xref, &lcp); err != nil {
			return nil, err
		}
		var depStrs []string
		if depj != "" {
			_ = json.Unmarshal([]byte(depj), &depStrs)
		}
		deps := make([]domain.SubtaskID, len(depStrs))
		for i := range depStrs {
			deps[i] = domain.SubtaskID(depStrs[i])
		}
		subs = append(subs, domain.Subtask{
			ID:             domain.SubtaskID(sid),
			DependsOn:      deps,
			AgentRole:      role,
			Dispatched:     disp != 0,
			ExternalRef:    xref,
			LastCheckpoint: lcp,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &domain.Convoy{
		ID:        domain.ConvoyID(cid),
		ProductID: domain.ProductID(pid),
		ParentID:  domain.TaskID(parent),
		Subtasks:  subs,
		CreatedAt: ct,
	}, nil
}

func (s *ConvoyStore) ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Convoy, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id FROM convoys WHERE product_id = ? ORDER BY created_at DESC`, string(productID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]domain.Convoy, 0, len(ids))
	for _, id := range ids {
		c, err := s.ByID(ctx, domain.ConvoyID(id))
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, nil
}

// —— costs ——

type CostStore struct{ db *sql.DB }

func NewCostStore(db *sql.DB) *CostStore { return &CostStore{db: db} }

var _ ports.CostRepository = (*CostStore)(nil)

func (s *CostStore) Append(ctx context.Context, e domain.CostEvent) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO cost_events (id, product_id, task_id, amount, note, at)
VALUES (?, ?, ?, ?, ?, ?)
`, e.ID, string(e.ProductID), string(e.TaskID), e.Amount, e.Note, e.At.Format(time.RFC3339Nano))
	return err
}

func (s *CostStore) SumByProduct(ctx context.Context, productID domain.ProductID) (float64, error) {
	var sum sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(amount), 0) FROM cost_events WHERE product_id = ?`, string(productID)).Scan(&sum)
	if err != nil {
		return 0, err
	}
	if !sum.Valid {
		return 0, nil
	}
	return sum.Float64, nil
}

// —— checkpoints ——

type CheckpointStore struct{ db *sql.DB }

func NewCheckpointStore(db *sql.DB) *CheckpointStore { return &CheckpointStore{db: db} }

var _ ports.CheckpointRepository = (*CheckpointStore)(nil)

func (s *CheckpointStore) Save(ctx context.Context, taskID domain.TaskID, payload string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO checkpoints (task_id, payload) VALUES (?, ?)
ON CONFLICT(task_id) DO UPDATE SET payload = excluded.payload
`, string(taskID), payload)
	return err
}

func (s *CheckpointStore) Load(ctx context.Context, taskID domain.TaskID) (string, error) {
	var p string
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM checkpoints WHERE task_id = ?`, string(taskID)).Scan(&p)
	if err == sql.ErrNoRows {
		return "", domain.ErrNotFound
	}
	return p, err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
