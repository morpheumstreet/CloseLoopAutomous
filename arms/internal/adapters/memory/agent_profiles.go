package memory

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type AgentProfileStore struct {
	mu   sync.Mutex
	byID map[string]domain.AgentIdentity
}

func NewAgentProfileStore() *AgentProfileStore {
	return &AgentProfileStore{byID: make(map[string]domain.AgentIdentity)}
}

var _ ports.AgentProfileRepository = (*AgentProfileStore)(nil)

func (s *AgentProfileStore) Upsert(_ context.Context, gatewayID string, ident *domain.AgentIdentity) error {
	if ident == nil {
		return nil
	}
	cp := *ident
	// Round-trip through JSON so we match SQLite behavior.
	b, err := json.Marshal(&cp)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &cp); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[cp.ID] = cp
	_ = gatewayID
	return nil
}

func (s *AgentProfileStore) ByID(_ context.Context, id string) (*domain.AgentIdentity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := v
	return &cp, nil
}

func (s *AgentProfileStore) List(_ context.Context, limit int) ([]domain.AgentIdentity, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	type row struct {
		id   string
		seen time.Time
	}
	var rows []row
	for id, ident := range s.byID {
		rows = append(rows, row{id: id, seen: ident.LastSeen})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].seen.After(rows[j].seen) })
	var out []domain.AgentIdentity
	for i := range rows {
		if len(out) >= limit {
			break
		}
		v := s.byID[rows[i].id]
		out = append(out, v)
	}
	return out, nil
}
