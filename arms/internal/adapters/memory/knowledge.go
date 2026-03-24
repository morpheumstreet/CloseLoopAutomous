package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// KnowledgeStore is an in-memory knowledge repository (substring search for Search).
type KnowledgeStore struct {
	mu      sync.RWMutex
	nextID  int64
	entries []domain.KnowledgeEntry
}

func NewKnowledgeStore() *KnowledgeStore {
	return &KnowledgeStore{}
}

var _ ports.KnowledgeRepository = (*KnowledgeStore)(nil)

func (s *KnowledgeStore) Create(_ context.Context, e *domain.KnowledgeEntry) error {
	if e == nil {
		return domain.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	cp := *e
	cp.ID = s.nextID
	if strings.TrimSpace(cp.MetadataJSON) == "" {
		cp.MetadataJSON = "{}"
	}
	cp.TaskID = domain.TaskID(strings.TrimSpace(string(cp.TaskID)))
	s.entries = append(s.entries, cp)
	*e = cp
	return nil
}

func (s *KnowledgeStore) Update(_ context.Context, id int64, productID domain.ProductID, content string, metadataJSON string, at time.Time) error {
	meta := strings.TrimSpace(metadataJSON)
	if meta == "" {
		meta = "{}"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].ID == id && s.entries[i].ProductID == productID {
			s.entries[i].Content = content
			s.entries[i].MetadataJSON = meta
			s.entries[i].UpdatedAt = at.UTC()
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *KnowledgeStore) Delete(_ context.Context, id int64, productID domain.ProductID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].ID == id && s.entries[i].ProductID == productID {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *KnowledgeStore) ByID(_ context.Context, id int64, productID domain.ProductID) (*domain.KnowledgeEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.entries {
		if s.entries[i].ID == id && s.entries[i].ProductID == productID {
			cp := s.entries[i]
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (s *KnowledgeStore) ListByProduct(_ context.Context, productID domain.ProductID, limit int) ([]domain.KnowledgeEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.KnowledgeEntry
	for i := range s.entries {
		if s.entries[i].ProductID == productID {
			out = append(out, s.entries[i])
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *KnowledgeStore) Search(_ context.Context, productID domain.ProductID, ftsQuery string, limit int) ([]domain.KnowledgeEntry, error) {
	q := strings.TrimSpace(ftsQuery)
	if q == "" {
		return s.ListByProduct(context.Background(), productID, limit)
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	tokens := strings.Fields(strings.ToLower(q))
	s.mu.RLock()
	defer s.mu.RUnlock()
	type scored struct {
		e domain.KnowledgeEntry
		n int
	}
	var hits []scored
	for i := range s.entries {
		if s.entries[i].ProductID != productID {
			continue
		}
		low := strings.ToLower(s.entries[i].Content)
		n := 0
		for _, tok := range tokens {
			if tok != "" && strings.Contains(low, tok) {
				n++
			}
		}
		if n > 0 {
			hits = append(hits, scored{e: s.entries[i], n: n})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].n != hits[j].n {
			return hits[i].n > hits[j].n
		}
		return hits[i].e.UpdatedAt.After(hits[j].e.UpdatedAt)
	})
	out := make([]domain.KnowledgeEntry, 0, len(hits))
	for i := range hits {
		if len(out) >= limit {
			break
		}
		out = append(out, hits[i].e)
	}
	return out, nil
}
