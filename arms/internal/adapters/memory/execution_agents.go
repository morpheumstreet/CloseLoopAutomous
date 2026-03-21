package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type ExecutionAgentStore struct {
	mu   sync.Mutex
	byID map[string]*domain.ExecutionAgent
}

func NewExecutionAgentStore() *ExecutionAgentStore {
	return &ExecutionAgentStore{byID: make(map[string]*domain.ExecutionAgent)}
}

var _ ports.ExecutionAgentRegistry = (*ExecutionAgentStore)(nil)

func (s *ExecutionAgentStore) Save(_ context.Context, a *domain.ExecutionAgent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *a
	s.byID[a.ID] = &cp
	return nil
}

func (s *ExecutionAgentStore) ByID(_ context.Context, id string) (*domain.ExecutionAgent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (s *ExecutionAgentStore) List(_ context.Context, limit int) ([]domain.ExecutionAgent, error) {
	if limit <= 0 {
		limit = 200
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.ExecutionAgent, 0, len(s.byID))
	for _, a := range s.byID {
		out = append(out, *a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type AgentMailboxStore struct {
	mu   sync.Mutex
	rows []domain.AgentMailboxMessage
}

func NewAgentMailboxStore() *AgentMailboxStore {
	return &AgentMailboxStore{}
}

var _ ports.AgentMailboxRepository = (*AgentMailboxStore)(nil)

func (s *AgentMailboxStore) Append(_ context.Context, id, agentID string, taskID domain.TaskID, body string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows = append(s.rows, domain.AgentMailboxMessage{
		ID: id, AgentID: agentID, TaskID: taskID, Body: body, CreatedAt: at.UTC(),
	})
	return nil
}

func (s *AgentMailboxStore) ListByAgent(_ context.Context, agentID string, limit int) ([]domain.AgentMailboxMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.AgentMailboxMessage
	for i := len(s.rows) - 1; i >= 0; i-- {
		if s.rows[i].AgentID == agentID {
			out = append(out, s.rows[i])
			if len(out) >= limit {
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
