package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type GatewayEndpointStore struct {
	mu   sync.Mutex
	byID map[string]*domain.GatewayEndpoint
}

func NewGatewayEndpointStore() *GatewayEndpointStore {
	return &GatewayEndpointStore{byID: make(map[string]*domain.GatewayEndpoint)}
}

var _ ports.GatewayEndpointRegistry = (*GatewayEndpointStore)(nil)

func (s *GatewayEndpointStore) Save(_ context.Context, e *domain.GatewayEndpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *e
	s.byID[e.ID] = &cp
	return nil
}

func (s *GatewayEndpointStore) ByID(_ context.Context, id string) (*domain.GatewayEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (s *GatewayEndpointStore) List(_ context.Context, limit int) ([]domain.GatewayEndpoint, error) {
	if limit <= 0 {
		limit = 200
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.GatewayEndpoint, 0, len(s.byID))
	for _, e := range s.byID {
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *GatewayEndpointStore) Update(_ context.Context, e *domain.GatewayEndpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[e.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *e
	s.byID[e.ID] = &cp
	return nil
}

func (s *GatewayEndpointStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.byID, id)
	return nil
}
