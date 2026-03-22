package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type TaskChatStore struct {
	mu   sync.RWMutex
	rows []domain.TaskChatMessage
}

func NewTaskChatStore() *TaskChatStore {
	return &TaskChatStore{}
}

var _ ports.TaskChatRepository = (*TaskChatStore)(nil)

func (s *TaskChatStore) Append(_ context.Context, m *domain.TaskChatMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *m
	s.rows = append(s.rows, cp)
	return nil
}

func (s *TaskChatStore) ByID(_ context.Context, id string) (*domain.TaskChatMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.rows {
		if s.rows[i].ID == id {
			cp := s.rows[i]
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (s *TaskChatStore) ListByTask(_ context.Context, taskID domain.TaskID, limit int) ([]domain.TaskChatMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.RLock()
	var match []domain.TaskChatMessage
	for i := range s.rows {
		if s.rows[i].TaskID == taskID {
			match = append(match, s.rows[i])
		}
	}
	s.mu.RUnlock()
	sort.Slice(match, func(i, j int) bool {
		if match[i].CreatedAt.Equal(match[j].CreatedAt) {
			return match[i].ID < match[j].ID
		}
		return match[i].CreatedAt.Before(match[j].CreatedAt)
	})
	if len(match) > limit {
		match = match[len(match)-limit:]
	}
	return match, nil
}

func (s *TaskChatStore) ListPendingQueueByProduct(_ context.Context, productID domain.ProductID, limit int) ([]domain.TaskChatMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.RLock()
	var match []domain.TaskChatMessage
	for i := range s.rows {
		if s.rows[i].ProductID == productID && s.rows[i].QueuePending {
			match = append(match, s.rows[i])
		}
	}
	s.mu.RUnlock()
	sort.Slice(match, func(i, j int) bool {
		if match[i].CreatedAt.Equal(match[j].CreatedAt) {
			return match[i].ID < match[j].ID
		}
		return match[i].CreatedAt.Before(match[j].CreatedAt)
	})
	if len(match) > limit {
		match = match[:limit]
	}
	return match, nil
}

func (s *TaskChatStore) ClearQueuePending(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.rows {
		if s.rows[i].ID == id {
			if !s.rows[i].QueuePending {
				return domain.ErrNotFound
			}
			s.rows[i].QueuePending = false
			return nil
		}
	}
	return domain.ErrNotFound
}
