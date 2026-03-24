package memory

import (
	"context"
	"sync"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type PreferenceModelStore struct {
	mu    sync.Mutex
	rows  map[domain.ProductID]struct {
		json string
		at   time.Time
	}
}

func NewPreferenceModelStore() *PreferenceModelStore {
	return &PreferenceModelStore{rows: make(map[domain.ProductID]struct {
		json string
		at   time.Time
	})}
}

var _ ports.PreferenceModelRepository = (*PreferenceModelStore)(nil)

func (s *PreferenceModelStore) Get(_ context.Context, productID domain.ProductID) (modelJSON string, updatedAt time.Time, ok bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.rows[productID]
	if !ok {
		return "", time.Time{}, false, nil
	}
	return v.json, v.at, true, nil
}

func (s *PreferenceModelStore) Upsert(_ context.Context, productID domain.ProductID, modelJSON string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[productID] = struct {
		json string
		at   time.Time
	}{json: modelJSON, at: at.UTC()}
	return nil
}
