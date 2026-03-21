package memory

import (
	"context"
	"sync"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

type SwipeHistoryStore struct {
	mu   sync.Mutex
	next int64
	rows []domain.SwipeHistoryEntry
}

func NewSwipeHistoryStore() *SwipeHistoryStore {
	return &SwipeHistoryStore{}
}

func (s *SwipeHistoryStore) Append(ctx context.Context, ideaID domain.IdeaID, productID domain.ProductID, decision string, at time.Time) error {
	_ = ctx
	if decision == "" {
		return domain.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	s.rows = append(s.rows, domain.SwipeHistoryEntry{
		ID:        s.next,
		IdeaID:    ideaID,
		ProductID: productID,
		Decision:  decision,
		CreatedAt: at.UTC(),
	})
	return nil
}

func (s *SwipeHistoryStore) ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.SwipeHistoryEntry, error) {
	_ = ctx
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.SwipeHistoryEntry
	for i := len(s.rows) - 1; i >= 0 && len(out) < limit; i-- {
		if s.rows[i].ProductID == productID {
			out = append(out, s.rows[i])
		}
	}
	return out, nil
}
