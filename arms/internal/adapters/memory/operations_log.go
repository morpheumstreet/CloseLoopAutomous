package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

const opsLogMax = 10000

type OperationsLogStore struct {
	mu   sync.Mutex
	next int64
	rows []domain.OperationLogEntry // newest at end
}

func NewOperationsLogStore() *OperationsLogStore {
	return &OperationsLogStore{}
}

var _ ports.OperationsLogRepository = (*OperationsLogStore)(nil)

func (s *OperationsLogStore) Append(_ context.Context, e domain.OperationLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	e.ID = s.next
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	} else {
		e.CreatedAt = e.CreatedAt.UTC()
	}
	s.rows = append(s.rows, e)
	if len(s.rows) > opsLogMax {
		s.rows = s.rows[len(s.rows)-opsLogMax:]
	}
	return nil
}

func (s *OperationsLogStore) List(_ context.Context, f ports.OperationsLogFilter) ([]domain.OperationLogEntry, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	wantAction := strings.TrimSpace(f.Action)
	wantRT := strings.TrimSpace(f.ResourceType)
	s.mu.Lock()
	defer s.mu.Unlock()
	var match []domain.OperationLogEntry
	for i := len(s.rows) - 1; i >= 0; i-- {
		e := s.rows[i]
		if f.ProductID != nil && *f.ProductID != "" && e.ProductID != *f.ProductID {
			continue
		}
		if wantAction != "" && e.Action != wantAction {
			continue
		}
		if wantRT != "" && e.ResourceType != wantRT {
			continue
		}
		if f.Since != nil && e.CreatedAt.Before(*f.Since) {
			continue
		}
		match = append(match, e)
		if len(match) >= limit {
			break
		}
	}
	return match, nil
}
