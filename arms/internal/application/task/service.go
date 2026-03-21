package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service owns task lifecycle up to dispatch; execution is delegated to AgentGateway.
type Service struct {
	Tasks    ports.TaskRepository
	Products ports.ProductRepository
	Ideas    ports.IdeaRepository
	Gateway  ports.AgentGateway
	Budget   ports.BudgetPolicy
	Checkpt  ports.CheckpointRepository
	Clock    ports.Clock
	IDs      ports.IdentityGenerator
	Events   ports.LiveActivityPublisher // optional: live activity / outbox
}

// CreateFromApprovedIdea starts the Kanban in planning until ApprovePlan moves to inbox.
func (s *Service) CreateFromApprovedIdea(ctx context.Context, ideaID domain.IdeaID, spec string) (*domain.Task, error) {
	idea, err := s.Ideas.ByID(ctx, ideaID)
	if err != nil {
		return nil, err
	}
	if !idea.Decided || !idea.Decision.Approved() {
		return nil, fmt.Errorf("%w: idea not approved", domain.ErrInvalidTransition)
	}
	now := s.Clock.Now()
	t := &domain.Task{
		ID:           s.IDs.NewTaskID(),
		ProductID:    idea.ProductID,
		IdeaID:       ideaID,
		Spec:         spec,
		Status:       domain.StatusPlanning,
		PlanApproved: false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.Tasks.Save(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ListByProduct returns tasks for a product (newest first), or ErrNotFound if the product does not exist.
func (s *Service) ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Task, error) {
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.Tasks.ListByProduct(ctx, productID)
}

// ApprovePlan clears the planning gate and moves the task to inbox (MC-style).
func (s *Service) ApprovePlan(ctx context.Context, taskID domain.TaskID, spec string) error {
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	if t.Status != domain.StatusPlanning {
		return fmt.Errorf("%w: task not in planning", domain.ErrInvalidTransition)
	}
	t.PlanApproved = true
	if strings.TrimSpace(spec) != "" {
		t.Spec = spec
	}
	t.Status = domain.StatusInbox
	t.StatusReason = ""
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// ReturnToPlanning revokes plan approval and moves the task back to planning from inbox or assigned (before dispatch).
func (s *Service) ReturnToPlanning(ctx context.Context, taskID domain.TaskID, statusReason string) error {
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	reason := strings.TrimSpace(statusReason)
	switch t.Status {
	case domain.StatusPlanning:
		return fmt.Errorf("%w: already in planning", domain.ErrInvalidTransition)
	case domain.StatusInbox:
		if !domain.AllowedKanbanTransition(t.Status, domain.StatusPlanning) {
			return fmt.Errorf("%w: inbox -> planning", domain.ErrInvalidTransition)
		}
		t.Status = domain.StatusPlanning
		t.PlanApproved = false
		t.StatusReason = reason
	case domain.StatusAssigned:
		if t.ExternalRef != "" {
			return fmt.Errorf("%w: cannot recall after dispatch", domain.ErrInvalidTransition)
		}
		if !domain.AllowedKanbanTransition(domain.StatusAssigned, domain.StatusInbox) ||
			!domain.AllowedKanbanTransition(domain.StatusInbox, domain.StatusPlanning) {
			return fmt.Errorf("%w: assigned -> planning", domain.ErrInvalidTransition)
		}
		t.Status = domain.StatusPlanning
		t.PlanApproved = false
		t.StatusReason = reason
	default:
		return fmt.Errorf("%w: return to planning from %s", domain.ErrInvalidTransition, t.Status)
	}
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// SetKanbanStatus moves the task on the board when AllowedKanbanTransition permits it.
func (s *Service) SetKanbanStatus(ctx context.Context, taskID domain.TaskID, to domain.TaskStatus, statusReason string) error {
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	if !domain.AllowedKanbanTransition(t.Status, to) {
		return fmt.Errorf("%w: %s -> %s", domain.ErrInvalidTransition, t.Status, to)
	}
	if to == domain.StatusAssigned && !t.PlanApproved {
		return fmt.Errorf("%w: assign requires approved plan", domain.ErrInvalidTransition)
	}
	t.Status = to
	t.StatusReason = strings.TrimSpace(statusReason)
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// UpdatePlanningArtifacts stores opaque planning JSON (e.g. clarifying Q&A) while in planning.
func (s *Service) UpdatePlanningArtifacts(ctx context.Context, taskID domain.TaskID, clarificationsJSON string) error {
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	if t.Status != domain.StatusPlanning {
		return fmt.Errorf("%w: not in planning", domain.ErrInvalidTransition)
	}
	t.ClarificationsJSON = clarificationsJSON
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// SetStatusReasonOnly updates the free-text reason without moving the Kanban column.
func (s *Service) SetStatusReasonOnly(ctx context.Context, taskID domain.TaskID, statusReason string) error {
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	t.StatusReason = strings.TrimSpace(statusReason)
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// Dispatch sends work to the execution plane when the task is assigned (MC dispatch gate).
func (s *Service) Dispatch(ctx context.Context, taskID domain.TaskID, estimatedCost float64) error {
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	if t.Status != domain.StatusAssigned {
		return fmt.Errorf("%w: dispatch requires status assigned (got %s)", domain.ErrInvalidTransition, t.Status)
	}
	if !t.PlanApproved {
		return fmt.Errorf("%w: plan not approved", domain.ErrInvalidTransition)
	}
	if err := s.Budget.AssertWithinBudget(ctx, t.ProductID, estimatedCost); err != nil {
		return err
	}
	ref, err := s.Gateway.DispatchTask(ctx, *t)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrGateway, err)
	}
	t.Status = domain.StatusInProgress
	t.ExternalRef = ref
	t.UpdatedAt = s.Clock.Now()
	if err := s.Tasks.Save(ctx, t); err != nil {
		return err
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ports.LiveActivityEvent{
			Type:      "task_dispatched",
			Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
			ProductID: string(t.ProductID),
			TaskID:    string(t.ID),
			Data: map[string]any{
				"external_ref": ref,
			},
		})
	}
	return nil
}

// RecordCheckpoint persists crash-recovery state from the gateway stream.
func (s *Service) RecordCheckpoint(ctx context.Context, taskID domain.TaskID, payload string) error {
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	switch t.Status {
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusConvoyActive:
	default:
		return fmt.Errorf("%w: checkpoint not allowed in %s", domain.ErrInvalidTransition, t.Status)
	}
	if err := s.Checkpt.Save(ctx, taskID, payload); err != nil {
		return err
	}
	t.Checkpoint = payload
	t.Status = domain.StatusInProgress
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}

// Complete marks the task finished (e.g. after agent-completion webhook).
func (s *Service) Complete(ctx context.Context, taskID domain.TaskID) error {
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return err
	}
	switch t.Status {
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusReview:
	default:
		return fmt.Errorf("%w: complete from %s", domain.ErrInvalidTransition, t.Status)
	}
	t.Status = domain.StatusDone
	t.StatusReason = ""
	t.UpdatedAt = s.Clock.Now()
	return s.Tasks.Save(ctx, t)
}
