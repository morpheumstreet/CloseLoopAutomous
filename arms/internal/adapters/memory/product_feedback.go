package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type ProductFeedbackStore struct {
	mu   sync.RWMutex
	rows []domain.ProductFeedback
}

func NewProductFeedbackStore() *ProductFeedbackStore {
	return &ProductFeedbackStore{}
}

var _ ports.ProductFeedbackRepository = (*ProductFeedbackStore)(nil)

func (s *ProductFeedbackStore) Append(_ context.Context, f *domain.ProductFeedback) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *f
	s.rows = append(s.rows, cp)
	return nil
}

func (s *ProductFeedbackStore) ListByProduct(_ context.Context, productID domain.ProductID, limit int) ([]domain.ProductFeedback, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var match []domain.ProductFeedback
	for i := len(s.rows) - 1; i >= 0; i-- {
		if s.rows[i].ProductID == productID {
			match = append(match, s.rows[i])
			if len(match) >= limit {
				break
			}
		}
	}
	sort.Slice(match, func(i, j int) bool { return match[i].CreatedAt.After(match[j].CreatedAt) })
	return match, nil
}

func (s *ProductFeedbackStore) ByID(_ context.Context, id string) (*domain.ProductFeedback, error) {
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

func (s *ProductFeedbackStore) SetProcessed(_ context.Context, id string, processed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.rows {
		if s.rows[i].ID == id {
			s.rows[i].Processed = processed
			return nil
		}
	}
	return domain.ErrNotFound
}
