package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

type ProductStore struct {
	mu   sync.RWMutex
	data map[domain.ProductID]*domain.Product
}

func NewProductStore() *ProductStore {
	return &ProductStore{data: make(map[domain.ProductID]*domain.Product)}
}

func (s *ProductStore) Save(_ context.Context, p *domain.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	s.data[p.ID] = &cp
	return nil
}

func (s *ProductStore) ByID(_ context.Context, id domain.ProductID) (*domain.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (s *ProductStore) ListAll(_ context.Context) ([]domain.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]domain.ProductID, 0, len(s.data))
	for id := range s.data {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return string(ids[i]) < string(ids[j]) })
	out := make([]domain.Product, 0, len(ids))
	for _, id := range ids {
		cp := *s.data[id]
		out = append(out, cp)
	}
	return out, nil
}

var _ ports.ProductRepository = (*ProductStore)(nil)

type MaybePoolStore struct {
	mu   sync.RWMutex
	rows map[domain.IdeaID]struct {
		ProductID domain.ProductID
		CreatedAt time.Time
	}
}

func NewMaybePoolStore() *MaybePoolStore {
	return &MaybePoolStore{rows: make(map[domain.IdeaID]struct {
		ProductID domain.ProductID
		CreatedAt time.Time
	})}
}

func (s *MaybePoolStore) Add(_ context.Context, ideaID domain.IdeaID, productID domain.ProductID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[ideaID] = struct {
		ProductID domain.ProductID
		CreatedAt time.Time
	}{ProductID: productID, CreatedAt: at}
	return nil
}

func (s *MaybePoolStore) Remove(_ context.Context, ideaID domain.IdeaID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rows, ideaID)
	return nil
}

func (s *MaybePoolStore) ListIdeaIDsByProduct(_ context.Context, productID domain.ProductID) ([]domain.IdeaID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var ids []domain.IdeaID
	for iid, e := range s.rows {
		if e.ProductID == productID {
			ids = append(ids, iid)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return string(ids[i]) < string(ids[j]) })
	return ids, nil
}

var _ ports.MaybePoolRepository = (*MaybePoolStore)(nil)

type IdeaStore struct {
	mu   sync.RWMutex
	data map[domain.IdeaID]*domain.Idea
}

func NewIdeaStore() *IdeaStore {
	return &IdeaStore{data: make(map[domain.IdeaID]*domain.Idea)}
}

func (s *IdeaStore) Save(_ context.Context, i *domain.Idea) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *i
	s.data[i.ID] = &cp
	return nil
}

func (s *IdeaStore) ByID(_ context.Context, id domain.IdeaID) (*domain.Idea, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i, ok := s.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *i
	return &cp, nil
}

func (s *IdeaStore) ListByProduct(_ context.Context, productID domain.ProductID) ([]domain.Idea, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Idea
	for _, i := range s.data {
		if i.ProductID == productID {
			out = append(out, *i)
		}
	}
	return out, nil
}

var _ ports.IdeaRepository = (*IdeaStore)(nil)

type TaskStore struct {
	mu   sync.RWMutex
	data map[domain.TaskID]*domain.Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{data: make(map[domain.TaskID]*domain.Task)}
}

func (s *TaskStore) Save(_ context.Context, t *domain.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *t
	s.data[t.ID] = &cp
	return nil
}

func (s *TaskStore) ByID(_ context.Context, id domain.TaskID) (*domain.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (s *TaskStore) ListByProduct(_ context.Context, productID domain.ProductID) ([]domain.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Task, 0)
	for _, t := range s.data {
		if t.ProductID != productID {
			continue
		}
		cp := *t
		out = append(out, cp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

var _ ports.TaskRepository = (*TaskStore)(nil)

type ConvoyStore struct {
	mu   sync.RWMutex
	data map[domain.ConvoyID]*domain.Convoy
}

func NewConvoyStore() *ConvoyStore {
	return &ConvoyStore{data: make(map[domain.ConvoyID]*domain.Convoy)}
}

func (s *ConvoyStore) Save(_ context.Context, c *domain.Convoy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *c
	cp.Subtasks = append([]domain.Subtask(nil), c.Subtasks...)
	s.data[c.ID] = &cp
	return nil
}

func (s *ConvoyStore) ByID(_ context.Context, id domain.ConvoyID) (*domain.Convoy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	cp.Subtasks = append([]domain.Subtask(nil), c.Subtasks...)
	return &cp, nil
}

func (s *ConvoyStore) ListByProduct(_ context.Context, productID domain.ProductID) ([]domain.Convoy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Convoy, 0)
	for _, c := range s.data {
		if c.ProductID != productID {
			continue
		}
		cp := *c
		cp.Subtasks = append([]domain.Subtask(nil), c.Subtasks...)
		out = append(out, cp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

var _ ports.ConvoyRepository = (*ConvoyStore)(nil)

type CostStore struct {
	mu   sync.RWMutex
	rows []domain.CostEvent
}

func NewCostStore() *CostStore {
	return &CostStore{}
}

func (s *CostStore) Append(_ context.Context, e domain.CostEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows = append(s.rows, e)
	return nil
}

func (s *CostStore) SumByProduct(_ context.Context, productID domain.ProductID) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sum float64
	for _, e := range s.rows {
		if e.ProductID == productID {
			sum += e.Amount
		}
	}
	return sum, nil
}

var _ ports.CostRepository = (*CostStore)(nil)

type CheckpointStore struct {
	mu   sync.RWMutex
	data map[domain.TaskID]string
}

func NewCheckpointStore() *CheckpointStore {
	return &CheckpointStore{data: make(map[domain.TaskID]string)}
}

func (s *CheckpointStore) Save(_ context.Context, taskID domain.TaskID, payload string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[taskID] = payload
	return nil
}

func (s *CheckpointStore) Load(_ context.Context, taskID domain.TaskID) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[taskID]
	if !ok {
		return "", domain.ErrNotFound
	}
	return v, nil
}

var _ ports.CheckpointRepository = (*CheckpointStore)(nil)
