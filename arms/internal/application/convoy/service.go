package convoy

import (
	"context"
	"fmt"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service builds a convoy DAG and dispatches subtasks through the same gateway port as single tasks.
type Service struct {
	Convoys  ports.ConvoyRepository
	Tasks    ports.TaskRepository
	Products ports.ProductRepository
	Gateway  ports.AgentGateway
	Budget   ports.BudgetPolicy // optional; when set, enforces caps per subtask dispatch (like task.Dispatch)
	Clock    ports.Clock
	IDs      ports.IdentityGenerator
	Events   ports.LiveActivityPublisher // optional: SSE / outbox on dispatch + subtask completion
}

// Create attaches subtasks to a parent task (roles + dependencies only; no dispatch).
func (s *Service) Create(ctx context.Context, parent domain.TaskID, productID domain.ProductID, subtasks []domain.Subtask) (*domain.Convoy, error) {
	if _, err := s.Tasks.ByID(ctx, parent); err != nil {
		return nil, err
	}
	for i := range subtasks {
		if subtasks[i].ID == "" {
			subtasks[i].ID = s.IDs.NewSubtaskID()
		}
	}
	c := &domain.Convoy{
		ID:        s.IDs.NewConvoyID(),
		ProductID: productID,
		ParentID:  parent,
		Subtasks:  subtasks,
		CreatedAt: s.Clock.Now(),
	}
	if err := s.Convoys.Save(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// Get returns a convoy by id or ErrNotFound.
func (s *Service) Get(ctx context.Context, id domain.ConvoyID) (*domain.Convoy, error) {
	return s.Convoys.ByID(ctx, id)
}

// ListByProduct returns convoys for a product (newest first), or ErrNotFound if the product does not exist.
func (s *Service) ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Convoy, error) {
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.Convoys.ListByProduct(ctx, productID)
}

// DispatchReady dispatches subtasks whose dependencies are already completed (one wave).
// estimatedCostPerSubtask is passed to Budget.AssertWithinBudget once per subtask about to be dispatched (parity with POST /api/tasks/{id}/dispatch).
func (s *Service) DispatchReady(ctx context.Context, convoyID domain.ConvoyID, estimatedCostPerSubtask float64) error {
	c, err := s.Convoys.ByID(ctx, convoyID)
	if err != nil {
		return err
	}
	parent, err := s.Tasks.ByID(ctx, c.ParentID)
	if err != nil {
		return err
	}
	completed := make(map[domain.SubtaskID]bool, len(c.Subtasks))
	for i := range c.Subtasks {
		if c.Subtasks[i].Completed {
			completed[c.Subtasks[i].ID] = true
		}
	}
	for i := range c.Subtasks {
		st := &c.Subtasks[i]
		if st.Dispatched {
			continue
		}
		ready := true
		for _, dep := range st.DependsOn {
			if !completed[dep] {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}
		if s.Budget != nil {
			if err := s.Budget.AssertWithinBudget(ctx, c.ProductID, estimatedCostPerSubtask); err != nil {
				return err
			}
		}
		ref, err := s.Gateway.DispatchSubtask(ctx, parent.ID, *st)
		if err != nil {
			return fmt.Errorf("%w: subtask %s: %v", domain.ErrGateway, st.ID, err)
		}
		st.Dispatched = true
		st.ExternalRef = ref
		if s.Events != nil {
			_ = s.Events.Publish(ctx, ports.LiveActivityEvent{
				Type:      "convoy_subtask_dispatched",
				Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
				ProductID: string(c.ProductID),
				TaskID:    string(parent.ID),
				Data: map[string]any{
					"convoy_id":    string(c.ID),
					"subtask_id":   string(st.ID),
					"agent_role":   st.AgentRole,
					"external_ref": ref,
				},
			})
		}
	}
	return s.Convoys.Save(ctx, c)
}

// CompleteSubtask marks a dispatched subtask finished (typically via agent-completion webhook).
// parentTaskID must match the convoy's parent task. Idempotent if already completed.
func (s *Service) CompleteSubtask(ctx context.Context, convoyID domain.ConvoyID, subtaskID domain.SubtaskID, parentTaskID domain.TaskID) error {
	c, err := s.Convoys.ByID(ctx, convoyID)
	if err != nil {
		return err
	}
	if c.ParentID != parentTaskID {
		return fmt.Errorf("%w: task_id does not match convoy parent", domain.ErrInvalidInput)
	}
	idx := -1
	for i := range c.Subtasks {
		if c.Subtasks[i].ID == subtaskID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return domain.ErrNotFound
	}
	st := &c.Subtasks[idx]
	if !st.Dispatched {
		return fmt.Errorf("%w: subtask not dispatched yet", domain.ErrInvalidTransition)
	}
	if st.Completed {
		return nil
	}
	st.Completed = true
	if err := s.Convoys.Save(ctx, c); err != nil {
		return err
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ports.LiveActivityEvent{
			Type:      "convoy_subtask_completed",
			Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
			ProductID: string(c.ProductID),
			TaskID:    string(parentTaskID),
			Data: map[string]any{
				"convoy_id":  string(c.ID),
				"subtask_id": string(subtaskID),
				"agent_role": st.AgentRole,
			},
		})
	}
	return nil
}
