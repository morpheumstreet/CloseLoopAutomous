package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type AgentHealthStore struct {
	mu   sync.RWMutex
	rows map[domain.TaskID]domain.TaskAgentHealth
}

func NewAgentHealthStore() *AgentHealthStore {
	return &AgentHealthStore{rows: make(map[domain.TaskID]domain.TaskAgentHealth)}
}

var _ ports.AgentHealthRepository = (*AgentHealthStore)(nil)

func (s *AgentHealthStore) UpsertHeartbeat(_ context.Context, taskID domain.TaskID, productID domain.ProductID, status, detailJSON string, at time.Time) error {
	if detailJSON == "" {
		detailJSON = "{}"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[taskID] = domain.TaskAgentHealth{
		TaskID: taskID, ProductID: productID, Status: status, DetailJSON: detailJSON,
		LastHeartbeatAt: at.UTC(),
	}
	return nil
}

func (s *AgentHealthStore) ByTask(_ context.Context, taskID domain.TaskID) (*domain.TaskAgentHealth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.rows[taskID]
	if !ok {
		return nil, nil
	}
	cp := h
	return &cp, nil
}

func (s *AgentHealthStore) ListByProduct(_ context.Context, productID domain.ProductID, limit int) ([]domain.TaskAgentHealth, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.TaskAgentHealth
	for _, h := range s.rows {
		if h.ProductID == productID {
			out = append(out, h)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastHeartbeatAt.After(out[j].LastHeartbeatAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *AgentHealthStore) ListRecent(_ context.Context, limit int) ([]domain.TaskAgentHealth, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.TaskAgentHealth, 0, len(s.rows))
	for _, h := range s.rows {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastHeartbeatAt.After(out[j].LastHeartbeatAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
