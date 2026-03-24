package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type ResearchCycleStore struct {
	mu   sync.Mutex
	rows []domain.ResearchCycle
}

func NewResearchCycleStore() *ResearchCycleStore {
	return &ResearchCycleStore{}
}

var _ ports.ResearchCycleRepository = (*ResearchCycleStore)(nil)

func (s *ResearchCycleStore) Append(_ context.Context, id string, productID domain.ProductID, summary string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows = append(s.rows, domain.ResearchCycle{
		ID: id, ProductID: productID, SummarySnapshot: summary, CreatedAt: at.UTC(),
	})
	return nil
}

func (s *ResearchCycleStore) ListByProduct(_ context.Context, productID domain.ProductID, limit int) ([]domain.ResearchCycle, error) {
	if limit <= 0 {
		limit = 50
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var match []domain.ResearchCycle
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
