package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

type ResearchHubStore struct {
	mu   sync.Mutex
	byID map[string]*domain.ResearchHub
}

func NewResearchHubStore() *ResearchHubStore {
	return &ResearchHubStore{byID: make(map[string]*domain.ResearchHub)}
}

var _ ports.ResearchHubRegistry = (*ResearchHubStore)(nil)

func (s *ResearchHubStore) Save(_ context.Context, h *domain.ResearchHub) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *h
	s.byID[h.ID] = &cp
	return nil
}

func (s *ResearchHubStore) ByID(_ context.Context, id string) (*domain.ResearchHub, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (s *ResearchHubStore) List(_ context.Context, limit int) ([]domain.ResearchHub, error) {
	if limit <= 0 {
		limit = 200
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.ResearchHub, 0, len(s.byID))
	for _, e := range s.byID {
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *ResearchHubStore) Update(_ context.Context, h *domain.ResearchHub) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[h.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *h
	s.byID[h.ID] = &cp
	return nil
}

func (s *ResearchHubStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.byID, id)
	return nil
}

type ResearchSystemSettingsStore struct {
	mu sync.Mutex
	v  domain.ResearchSystemSettings
}

func NewResearchSystemSettingsStore() *ResearchSystemSettingsStore {
	return &ResearchSystemSettingsStore{}
}

var _ ports.ResearchSystemSettingsRepository = (*ResearchSystemSettingsStore)(nil)

func (s *ResearchSystemSettingsStore) Get(_ context.Context) (domain.ResearchSystemSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.v, nil
}

func (s *ResearchSystemSettingsStore) Upsert(_ context.Context, st domain.ResearchSystemSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.v = st
	return nil
}
