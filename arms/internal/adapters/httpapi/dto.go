package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
)

type createProductReq struct {
	Name            string `json:"name"`
	WorkspaceID     string `json:"workspace_id"`
	RepoURL         string `json:"repo_url,omitempty"`
	RepoClonePath   string `json:"repo_clone_path,omitempty"`
	RepoBranch      string `json:"repo_branch,omitempty"`
	Description     string `json:"description,omitempty"`
	ProgramDocument string `json:"program_document,omitempty"`
	SettingsJSON    string `json:"settings_json,omitempty"`
	IconURL         string `json:"icon_url,omitempty"`

	ResearchCadenceSec  *int   `json:"research_cadence_sec,omitempty"`
	IdeationCadenceSec  *int   `json:"ideation_cadence_sec,omitempty"`
	AutomationTier      string `json:"automation_tier,omitempty"`
	AutoDispatchEnabled *bool  `json:"auto_dispatch_enabled,omitempty"`
	MergePolicyJSON     string `json:"merge_policy_json,omitempty"`
}

func (r *createProductReq) validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(r.WorkspaceID) == "" {
		return fmt.Errorf("workspace_id is required")
	}
	if r.ResearchCadenceSec != nil && *r.ResearchCadenceSec < 0 {
		return fmt.Errorf("research_cadence_sec must be >= 0")
	}
	if r.IdeationCadenceSec != nil && *r.IdeationCadenceSec < 0 {
		return fmt.Errorf("ideation_cadence_sec must be >= 0")
	}
	return nil
}

type patchProductReq struct {
	Name            *string `json:"name,omitempty"`
	RepoURL         *string `json:"repo_url,omitempty"`
	RepoClonePath   *string `json:"repo_clone_path,omitempty"`
	RepoBranch      *string `json:"repo_branch,omitempty"`
	Description     *string `json:"description,omitempty"`
	ProgramDocument *string `json:"program_document,omitempty"`
	SettingsJSON    *string `json:"settings_json,omitempty"`
	IconURL         *string `json:"icon_url,omitempty"`
	MergePolicyJSON *string `json:"merge_policy_json,omitempty"`

	ResearchCadenceSec  *int    `json:"research_cadence_sec,omitempty"`
	IdeationCadenceSec  *int    `json:"ideation_cadence_sec,omitempty"`
	AutomationTier      *string `json:"automation_tier,omitempty"`
	AutoDispatchEnabled *bool   `json:"auto_dispatch_enabled,omitempty"`
}

// patchProductAuditDetail lists which patch fields were present (for operations_log).
func patchProductAuditDetail(r *patchProductReq) map[string]bool {
	m := make(map[string]bool)
	if r.Name != nil {
		m["name"] = true
	}
	if r.RepoURL != nil {
		m["repo_url"] = true
	}
	if r.RepoClonePath != nil {
		m["repo_clone_path"] = true
	}
	if r.RepoBranch != nil {
		m["repo_branch"] = true
	}
	if r.Description != nil {
		m["description"] = true
	}
	if r.ProgramDocument != nil {
		m["program_document"] = true
	}
	if r.SettingsJSON != nil {
		m["settings_json"] = true
	}
	if r.IconURL != nil {
		m["icon_url"] = true
	}
	if r.MergePolicyJSON != nil {
		m["merge_policy_json"] = true
	}
	if r.ResearchCadenceSec != nil {
		m["research_cadence_sec"] = true
	}
	if r.IdeationCadenceSec != nil {
		m["ideation_cadence_sec"] = true
	}
	if r.AutomationTier != nil {
		m["automation_tier"] = true
	}
	if r.AutoDispatchEnabled != nil {
		m["auto_dispatch_enabled"] = true
	}
	return m
}

func (r *patchProductReq) validate() error {
	if r.Name == nil && r.RepoURL == nil && r.RepoClonePath == nil && r.RepoBranch == nil && r.Description == nil &&
		r.ProgramDocument == nil && r.SettingsJSON == nil && r.IconURL == nil && r.MergePolicyJSON == nil &&
		r.ResearchCadenceSec == nil && r.IdeationCadenceSec == nil && r.AutomationTier == nil && r.AutoDispatchEnabled == nil {
		return fmt.Errorf("at least one field is required")
	}
	if r.ResearchCadenceSec != nil && *r.ResearchCadenceSec < 0 {
		return fmt.Errorf("research_cadence_sec must be >= 0")
	}
	if r.IdeationCadenceSec != nil && *r.IdeationCadenceSec < 0 {
		return fmt.Errorf("ideation_cadence_sec must be >= 0")
	}
	return nil
}

type swipeReq struct {
	Decision string `json:"decision"`
}

// patchIdeaReq updates MC-style metadata (optional fields; at least one required).
type patchIdeaReq struct {
	Title                *string   `json:"title,omitempty"`
	Description          *string   `json:"description,omitempty"`
	Reasoning            *string   `json:"reasoning,omitempty"`
	Category             *string   `json:"category,omitempty"`
	ResearchBacking      *string   `json:"research_backing,omitempty"`
	ImpactScore          *float64  `json:"impact_score,omitempty"`
	FeasibilityScore     *float64  `json:"feasibility_score,omitempty"`
	Complexity           *string   `json:"complexity,omitempty"`
	EstimatedEffortHours *float64  `json:"estimated_effort_hours,omitempty"`
	CompetitiveAnalysis  *string   `json:"competitive_analysis,omitempty"`
	TargetUserSegment    *string   `json:"target_user_segment,omitempty"`
	RevenuePotential     *string   `json:"revenue_potential,omitempty"`
	TechnicalApproach    *string   `json:"technical_approach,omitempty"`
	Risks                *string   `json:"risks,omitempty"`
	Tags                 *[]string `json:"tags,omitempty"`
	Source               *string   `json:"source,omitempty"`
	SourceResearch       *string   `json:"source_research,omitempty"`
	UserNotes            *string   `json:"user_notes,omitempty"`
}

func (r *patchIdeaReq) anySet() bool {
	if r == nil {
		return false
	}
	return r.Title != nil || r.Description != nil || r.Reasoning != nil ||
		r.Category != nil || r.ResearchBacking != nil || r.ImpactScore != nil || r.FeasibilityScore != nil ||
		r.Complexity != nil || r.EstimatedEffortHours != nil || r.CompetitiveAnalysis != nil ||
		r.TargetUserSegment != nil || r.RevenuePotential != nil || r.TechnicalApproach != nil ||
		r.Risks != nil || r.Tags != nil || r.Source != nil || r.SourceResearch != nil || r.UserNotes != nil
}

func parseSwipe(s string) (domain.SwipeDecision, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pass":
		return domain.DecisionPass, nil
	case "maybe":
		return domain.DecisionMaybe, nil
	case "yes":
		return domain.DecisionYes, nil
	case "now":
		return domain.DecisionNow, nil
	default:
		return 0, fmt.Errorf("decision must be pass, maybe, yes, or now")
	}
}

type createTaskReq struct {
	IdeaID string `json:"idea_id"`
	Spec   string `json:"spec"`
}

func (r *createTaskReq) validate() error {
	if strings.TrimSpace(r.IdeaID) == "" {
		return fmt.Errorf("idea_id is required")
	}
	if strings.TrimSpace(r.Spec) == "" {
		return fmt.Errorf("spec is required")
	}
	return nil
}

type dispatchReq struct {
	EstimatedCost float64 `json:"estimated_cost"`
}

type registerAgentReq struct {
	DisplayName string `json:"display_name"`
	ProductID   string `json:"product_id,omitempty"`
	Source      string `json:"source,omitempty"`
	ExternalRef string `json:"external_ref,omitempty"`
}

func (r *registerAgentReq) validate() error {
	if strings.TrimSpace(r.DisplayName) == "" {
		return fmt.Errorf("display_name is required")
	}
	return nil
}

type postAgentMailReq struct {
	Body   string `json:"body"`
	TaskID string `json:"task_id,omitempty"`
}

func (r *postAgentMailReq) validate() error {
	if strings.TrimSpace(r.Body) == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

// patchTaskReq supports partial updates: planning JSON, Kanban move, or status_reason alone.
type patchTaskReq struct {
	Status             *string `json:"status,omitempty"`
	StatusReason       *string `json:"status_reason,omitempty"`
	ClarificationsJSON *string `json:"clarifications_json,omitempty"`
	SandboxPath        *string `json:"sandbox_path,omitempty"`
	WorktreePath       *string `json:"worktree_path,omitempty"`
}

func (r *patchTaskReq) validate() error {
	if r.Status == nil && r.StatusReason == nil && r.ClarificationsJSON == nil && r.SandboxPath == nil && r.WorktreePath == nil {
		return fmt.Errorf("at least one of status, status_reason, clarifications_json, sandbox_path, worktree_path is required")
	}
	return nil
}

type approvePlanReq struct {
	Spec string `json:"spec,omitempty"`
}

type rejectPlanReq struct {
	StatusReason string `json:"status_reason,omitempty"`
}

type checkpointReq struct {
	Payload string `json:"payload"`
}

type stallNudgeReq struct {
	Note string `json:"note,omitempty"`
}

func (r *checkpointReq) validate() error {
	if r.Payload == "" {
		return fmt.Errorf("payload is required")
	}
	return nil
}

type subtaskDTO struct {
	ID        string   `json:"id,omitempty"`
	AgentRole string   `json:"agent_role"`
	DependsOn []string `json:"depends_on,omitempty"`
}

type createConvoyReq struct {
	ParentTaskID string       `json:"parent_task_id"`
	ProductID    string       `json:"product_id"`
	Subtasks     []subtaskDTO `json:"subtasks"`
}

func (r *createConvoyReq) validate() error {
	if strings.TrimSpace(r.ParentTaskID) == "" {
		return fmt.Errorf("parent_task_id is required")
	}
	if strings.TrimSpace(r.ProductID) == "" {
		return fmt.Errorf("product_id is required")
	}
	if len(r.Subtasks) == 0 {
		return fmt.Errorf("subtasks is required")
	}
	for i := range r.Subtasks {
		if strings.TrimSpace(r.Subtasks[i].AgentRole) == "" {
			return fmt.Errorf("subtasks[%d].agent_role is required", i)
		}
	}
	return nil
}

type recordCostReq struct {
	ProductID string  `json:"product_id"`
	TaskID    string  `json:"task_id"`
	Amount    float64 `json:"amount"`
	Note      string  `json:"note,omitempty"`
	Agent     string  `json:"agent,omitempty"`
	Model     string  `json:"model,omitempty"`
}

func (r *recordCostReq) validate() error {
	if strings.TrimSpace(r.ProductID) == "" {
		return fmt.Errorf("product_id is required")
	}
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("task_id is required")
	}
	if r.Amount < 0 {
		return fmt.Errorf("amount must be >= 0")
	}
	return nil
}

type patchCostCapsReq struct {
	DailyCap       *float64 `json:"daily_cap"`
	MonthlyCap     *float64 `json:"monthly_cap"`
	CumulativeCap  *float64 `json:"cumulative_cap"`
}

func (r *patchCostCapsReq) validate() error {
	if r.DailyCap == nil && r.MonthlyCap == nil && r.CumulativeCap == nil {
		return fmt.Errorf("at least one of daily_cap, monthly_cap, cumulative_cap is required")
	}
	return nil
}

type restoreCheckpointReq struct {
	HistoryID int64 `json:"history_id"`
}

func (r *restoreCheckpointReq) validate() error {
	if r.HistoryID < 1 {
		return fmt.Errorf("history_id is required")
	}
	return nil
}

type allocatePortReq struct {
	ProductID string `json:"product_id"`
	TaskID    string `json:"task_id"`
}

type openPullRequestReq struct {
	HeadBranch string `json:"head_branch"`
	Title      string `json:"title,omitempty"`
	Body       string `json:"body,omitempty"`
}

func (r *openPullRequestReq) validate() error {
	if strings.TrimSpace(r.HeadBranch) == "" {
		return fmt.Errorf("head_branch is required")
	}
	return nil
}

func (r *allocatePortReq) validate() error {
	if strings.TrimSpace(r.ProductID) == "" {
		return fmt.Errorf("product_id is required")
	}
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("task_id is required")
	}
	return nil
}

type agentCompletionReq struct {
	TaskID           string `json:"task_id"`
	ConvoyID         string `json:"convoy_id,omitempty"`
	SubtaskID        string `json:"subtask_id,omitempty"`
	NextBoardStatus  string `json:"next_board_status,omitempty"` // omit or "done" → complete task; "testing" | "review" → Kanban move for full_auto/semi_auto
}

func (r *agentCompletionReq) validate() error {
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("task_id is required")
	}
	c := strings.TrimSpace(r.ConvoyID)
	s := strings.TrimSpace(r.SubtaskID)
	if (c != "" || s != "") && (c == "" || s == "") {
		return fmt.Errorf("convoy_id and subtask_id must both be set when reporting convoy subtask completion")
	}
	return nil
}

type ciCompletionReq struct {
	TaskID          string `json:"task_id"`
	NextBoardStatus string `json:"next_board_status"`
	StatusReason    string `json:"status_reason,omitempty"`
}

func (r *ciCompletionReq) validate() error {
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("task_id is required")
	}
	if strings.TrimSpace(r.NextBoardStatus) == "" {
		return fmt.Errorf("next_board_status is required")
	}
	return nil
}

type patchAgentHealthReq struct {
	Status string          `json:"status"`
	Detail json.RawMessage `json:"detail,omitempty"`
}

func (r *patchAgentHealthReq) validate() error {
	if strings.TrimSpace(r.Status) == "" {
		return fmt.Errorf("status is required")
	}
	return nil
}

type gitWorktreeReq struct {
	Branch string `json:"branch"`
}

func (r *gitWorktreeReq) validate() error {
	if strings.TrimSpace(r.Branch) == "" {
		return fmt.Errorf("branch is required")
	}
	return nil
}

type putPreferenceModelReq struct {
	ModelJSON string `json:"model_json"`
}

func (r *putPreferenceModelReq) validate() error {
	s := strings.TrimSpace(r.ModelJSON)
	if s == "" {
		return fmt.Errorf("model_json is required")
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return fmt.Errorf("model_json must be valid JSON")
	}
	return nil
}

type patchProductScheduleReq struct {
	Enabled       *bool   `json:"enabled,omitempty"`
	SpecJSON      *string `json:"spec_json,omitempty"`
	CronExpr      *string `json:"cron_expr,omitempty"`
	DelaySeconds  *int    `json:"delay_seconds,omitempty"`
}

func (r *patchProductScheduleReq) validate() error {
	if r.Enabled == nil && r.SpecJSON == nil && r.CronExpr == nil && r.DelaySeconds == nil {
		return fmt.Errorf("at least one of enabled, spec_json, cron_expr, delay_seconds is required")
	}
	return nil
}

type postConvoyMailReq struct {
	SubtaskID string `json:"subtask_id"`
	Body      string `json:"body"`
}

func (r *postConvoyMailReq) validate() error {
	if strings.TrimSpace(r.SubtaskID) == "" {
		return fmt.Errorf("subtask_id is required")
	}
	if strings.TrimSpace(r.Body) == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

// mergeQueueResolveReq is optional JSON for conflict-resolution helpers (defaults to retry_merge).
type mergeQueueResolveReq struct {
	Action string `json:"action"`
}
