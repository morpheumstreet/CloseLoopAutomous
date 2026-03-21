package httpapi

import (
	"fmt"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
)

type createProductReq struct {
	Name            string `json:"name"`
	WorkspaceID     string `json:"workspace_id"`
	RepoURL         string `json:"repo_url,omitempty"`
	RepoBranch      string `json:"repo_branch,omitempty"`
	Description     string `json:"description,omitempty"`
	ProgramDocument string `json:"program_document,omitempty"`
	SettingsJSON    string `json:"settings_json,omitempty"`
	IconURL         string `json:"icon_url,omitempty"`

	ResearchCadenceSec  *int   `json:"research_cadence_sec,omitempty"`
	IdeationCadenceSec  *int   `json:"ideation_cadence_sec,omitempty"`
	AutomationTier      string `json:"automation_tier,omitempty"`
	AutoDispatchEnabled *bool  `json:"auto_dispatch_enabled,omitempty"`
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
	RepoBranch      *string `json:"repo_branch,omitempty"`
	Description     *string `json:"description,omitempty"`
	ProgramDocument *string `json:"program_document,omitempty"`
	SettingsJSON    *string `json:"settings_json,omitempty"`
	IconURL         *string `json:"icon_url,omitempty"`

	ResearchCadenceSec  *int    `json:"research_cadence_sec,omitempty"`
	IdeationCadenceSec  *int    `json:"ideation_cadence_sec,omitempty"`
	AutomationTier      *string `json:"automation_tier,omitempty"`
	AutoDispatchEnabled *bool   `json:"auto_dispatch_enabled,omitempty"`
}

func (r *patchProductReq) validate() error {
	if r.Name == nil && r.RepoURL == nil && r.RepoBranch == nil && r.Description == nil &&
		r.ProgramDocument == nil && r.SettingsJSON == nil && r.IconURL == nil &&
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

// patchTaskReq supports partial updates: planning JSON, Kanban move, or status_reason alone.
type patchTaskReq struct {
	Status             *string `json:"status,omitempty"`
	StatusReason       *string `json:"status_reason,omitempty"`
	ClarificationsJSON *string `json:"clarifications_json,omitempty"`
}

func (r *patchTaskReq) validate() error {
	if r.Status == nil && r.StatusReason == nil && r.ClarificationsJSON == nil {
		return fmt.Errorf("at least one of status, status_reason, clarifications_json is required")
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

type agentCompletionReq struct {
	TaskID string `json:"task_id"`
}

func (r *agentCompletionReq) validate() error {
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("task_id is required")
	}
	return nil
}
