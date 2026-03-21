package convoy

import (
	"context"
	"fmt"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service builds a convoy DAG and dispatches subtasks through the same gateway port as single tasks.
type Service struct {
	Convoys  ports.ConvoyRepository
	Tasks    ports.TaskRepository
	Products ports.ProductRepository
	Gateway  ports.AgentGateway
	Clock    ports.Clock
	IDs      ports.IdentityGenerator
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

// DispatchReady dispatches subtasks whose dependencies are already dispatched (one wave).
func (s *Service) DispatchReady(ctx context.Context, convoyID domain.ConvoyID) error {
	c, err := s.Convoys.ByID(ctx, convoyID)
	if err != nil {
		return err
	}
	parent, err := s.Tasks.ByID(ctx, c.ParentID)
	if err != nil {
		return err
	}
	dispatched := make(map[domain.SubtaskID]bool, len(c.Subtasks))
	for i := range c.Subtasks {
		if c.Subtasks[i].Dispatched {
			dispatched[c.Subtasks[i].ID] = true
		}
	}
	for i := range c.Subtasks {
		st := &c.Subtasks[i]
		if st.Dispatched {
			continue
		}
		ready := true
		for _, dep := range st.DependsOn {
			if !dispatched[dep] {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}
		ref, err := s.Gateway.DispatchSubtask(ctx, parent.ID, *st)
		if err != nil {
			return fmt.Errorf("%w: subtask %s: %v", domain.ErrGateway, st.ID, err)
		}
		st.Dispatched = true
		st.ExternalRef = ref
		dispatched[st.ID] = true
	}
	return s.Convoys.Save(ctx, c)
}
