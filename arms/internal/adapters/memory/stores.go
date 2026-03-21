package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
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

func (s *TaskStore) TryComplete(_ context.Context, taskID domain.TaskID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.data[taskID]
	if !ok {
		return domain.ErrNotFound
	}
	switch t.Status {
	case domain.StatusDone:
		return nil
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusReview:
		cp := *t
		cp.Status = domain.StatusDone
		cp.StatusReason = ""
		cp.UpdatedAt = at.UTC()
		s.data[taskID] = &cp
		return nil
	default:
		return fmt.Errorf("%w: complete from %s", domain.ErrInvalidTransition, t.Status)
	}
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

func (s *CostStore) SumByProductSince(_ context.Context, productID domain.ProductID, since time.Time) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sum float64
	for _, e := range s.rows {
		if e.ProductID != productID {
			continue
		}
		if !since.IsZero() && e.At.Before(since) {
			continue
		}
		sum += e.Amount
	}
	return sum, nil
}

func (s *CostStore) ListByProductBetween(_ context.Context, productID domain.ProductID, from, to time.Time) ([]domain.CostEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.CostEvent
	for _, e := range s.rows {
		if e.ProductID != productID {
			continue
		}
		if !from.IsZero() && e.At.Before(from) {
			continue
		}
		if !to.IsZero() && e.At.After(to) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

var _ ports.CostRepository = (*CostStore)(nil)

type checkpointHist struct {
	id        int64
	task      domain.TaskID
	payload   string
	createdAt time.Time
}

type CheckpointStore struct {
	mu       sync.RWMutex
	latest   map[domain.TaskID]string
	history  []checkpointHist
	nextHist int64
}

func NewCheckpointStore() *CheckpointStore {
	return &CheckpointStore{latest: make(map[domain.TaskID]string)}
}

func (s *CheckpointStore) Save(_ context.Context, taskID domain.TaskID, payload string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextHist++
	s.history = append(s.history, checkpointHist{
		id: s.nextHist, task: taskID, payload: payload, createdAt: time.Now().UTC(),
	})
	s.latest[taskID] = payload
	return nil
}

func (s *CheckpointStore) Load(_ context.Context, taskID domain.TaskID) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.latest[taskID]
	if !ok {
		return "", domain.ErrNotFound
	}
	return v, nil
}

func (s *CheckpointStore) ListHistory(_ context.Context, taskID domain.TaskID, limit int) ([]domain.CheckpointHistoryEntry, error) {
	if limit < 1 {
		limit = 50
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.CheckpointHistoryEntry
	for i := len(s.history) - 1; i >= 0 && len(out) < limit; i-- {
		h := s.history[i]
		if h.task != taskID {
			continue
		}
		out = append(out, domain.CheckpointHistoryEntry{
			ID: h.id, TaskID: h.task, Payload: h.payload, CreatedAt: h.createdAt,
		})
	}
	return out, nil
}

func (s *CheckpointStore) HistoryByID(_ context.Context, id int64) (*domain.CheckpointHistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.history {
		if s.history[i].id == id {
			h := s.history[i]
			return &domain.CheckpointHistoryEntry{
				ID: h.id, TaskID: h.task, Payload: h.payload, CreatedAt: h.createdAt,
			}, nil
		}
	}
	return nil, domain.ErrNotFound
}

var _ ports.CheckpointRepository = (*CheckpointStore)(nil)

// —— cost caps ——

type CostCapStore struct {
	mu   sync.RWMutex
	data map[domain.ProductID]*domain.ProductCostCaps
}

func NewCostCapStore() *CostCapStore {
	return &CostCapStore{data: make(map[domain.ProductID]*domain.ProductCostCaps)}
}

func (s *CostCapStore) Get(_ context.Context, productID domain.ProductID) (*domain.ProductCostCaps, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.data[productID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (s *CostCapStore) Upsert(_ context.Context, caps *domain.ProductCostCaps) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *caps
	s.data[caps.ProductID] = &cp
	return nil
}

var _ ports.CostCapRepository = (*CostCapStore)(nil)

// —— workspace ——

const memWorkspacePortMin, memWorkspacePortMax = 4200, 4299

type mergeQueueRow struct {
	id                int64
	productID         domain.ProductID
	taskID            domain.TaskID
	status            string
	createdAt         time.Time
	completedAt       time.Time
	leaseOwner        string
	leaseExpiresAt    time.Time
	mergeShipState    string
	mergedSHA         string
	mergeError        string
	conflictFilesJSON string
}

type WorkspaceStore struct {
	mu       sync.Mutex
	ports    map[int]domain.AllocatedPort
	mq       []mergeQueueRow
	nextMQID int64
}

func NewWorkspaceStore() *WorkspaceStore {
	return &WorkspaceStore{ports: make(map[int]domain.AllocatedPort)}
}

func (s *WorkspaceStore) Allocate(ctx context.Context, productID domain.ProductID, taskID domain.TaskID, at time.Time) (int, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	for p := memWorkspacePortMin; p <= memWorkspacePortMax; p++ {
		if _, taken := s.ports[p]; taken {
			continue
		}
		s.ports[p] = domain.AllocatedPort{
			Port: p, ProductID: productID, TaskID: taskID, AllocatedAt: at,
		}
		return p, nil
	}
	return 0, domain.ErrNotFound
}

func (s *WorkspaceStore) Release(ctx context.Context, port int) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.ports[port]; !ok {
		return domain.ErrNotFound
	}
	delete(s.ports, port)
	return nil
}

func (s *WorkspaceStore) ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.AllocatedPort, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.AllocatedPort
	for _, a := range s.ports {
		if a.ProductID == productID {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Port < out[j].Port })
	return out, nil
}

func (s *WorkspaceStore) ListAll(ctx context.Context) ([]domain.AllocatedPort, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.AllocatedPort, 0, len(s.ports))
	for _, a := range s.ports {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Port < out[j].Port })
	return out, nil
}

func (s *WorkspaceStore) CountPending(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for i := range s.mq {
		if s.mq[i].status == "pending" {
			n++
		}
	}
	return n, nil
}

func (s *WorkspaceStore) Enqueue(_ context.Context, productID domain.ProductID, taskID domain.TaskID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.mq {
		if s.mq[i].taskID == taskID && s.mq[i].status == "pending" {
			return domain.ErrConflict
		}
	}
	s.nextMQID++
	s.mq = append(s.mq, mergeQueueRow{
		id:        s.nextMQID,
		productID: productID,
		taskID:    taskID,
		status:    "pending",
		createdAt: at.UTC(),
	})
	return nil
}

func (s *WorkspaceStore) ListPendingByProduct(_ context.Context, productID domain.ProductID, limit int) ([]domain.MergeQueueEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.MergeQueueEntry
	for i := range s.mq {
		row := &s.mq[i]
		if row.productID == productID && row.status == "pending" {
			out = append(out, domain.MergeQueueEntry{
				ID:                row.id,
				ProductID:         row.productID,
				TaskID:            row.taskID,
				Status:            row.status,
				CreatedAt:         row.createdAt,
				LeaseOwner:        row.leaseOwner,
				LeaseExpiresAt:    row.leaseExpiresAt,
				MergeShipState:    domain.MergeShipState(row.mergeShipState),
				MergedSHA:         row.mergedSHA,
				MergeError:        row.mergeError,
				ConflictFilesJSON: row.conflictFilesJSON,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *WorkspaceStore) CompletePendingForTask(_ context.Context, taskID domain.TaskID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var productID domain.ProductID
	var myIdx = -1
	for i := range s.mq {
		if s.mq[i].taskID == taskID && s.mq[i].status == "pending" {
			myIdx = i
			productID = s.mq[i].productID
			break
		}
	}
	if myIdx < 0 {
		return domain.ErrNotFound
	}
	var headID int64
	var haveHead bool
	for i := range s.mq {
		row := &s.mq[i]
		if row.productID == productID && row.status == "pending" {
			if !haveHead || row.id < headID {
				headID = row.id
				haveHead = true
			}
		}
	}
	if !haveHead || s.mq[myIdx].id != headID {
		return domain.ErrNotMergeQueueHead
	}
	s.mq[myIdx].status = "done"
	s.mq[myIdx].completedAt = time.Now().UTC()
	s.mq[myIdx].leaseOwner = ""
	s.mq[myIdx].leaseExpiresAt = time.Time{}
	return nil
}

func (s *WorkspaceStore) ReserveHeadForShip(_ context.Context, taskID domain.TaskID, leaseOwner string, leaseExpires time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var productID domain.ProductID
	var myIdx = -1
	for i := range s.mq {
		if s.mq[i].taskID == taskID && s.mq[i].status == "pending" {
			myIdx = i
			productID = s.mq[i].productID
			break
		}
	}
	if myIdx < 0 {
		return 0, domain.ErrNotFound
	}
	var headID int64
	var haveHead bool
	for i := range s.mq {
		row := &s.mq[i]
		if row.productID == productID && row.status == "pending" {
			if !haveHead || row.id < headID {
				headID = row.id
				haveHead = true
			}
		}
	}
	if !haveHead || s.mq[myIdx].id != headID {
		return 0, domain.ErrNotMergeQueueHead
	}
	now := time.Now().UTC()
	row := &s.mq[myIdx]
	if strings.TrimSpace(row.leaseOwner) != "" && !row.leaseExpiresAt.IsZero() && row.leaseExpiresAt.After(now) {
		return 0, domain.ErrMergeShipBusy
	}
	row.leaseOwner = strings.TrimSpace(leaseOwner)
	row.leaseExpiresAt = leaseExpires.UTC()
	return row.id, nil
}

func (s *WorkspaceStore) FinishShip(_ context.Context, rowID int64, leaseOwner string, result domain.MergeShipResult, shipOpErr error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.mq {
		if s.mq[i].id == rowID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return domain.ErrNotFound
	}
	row := &s.mq[idx]
	if strings.TrimSpace(row.leaseOwner) != strings.TrimSpace(leaseOwner) {
		return domain.ErrMergeShipBusy
	}
	r := result
	if errors.Is(shipOpErr, domain.ErrMergeConflict) && r.State == domain.MergeShipNone {
		r.State = domain.MergeShipConflict
		if strings.TrimSpace(r.ErrorMessage) == "" && shipOpErr != nil {
			r.ErrorMessage = shipOpErr.Error()
		}
	}
	if shipOpErr != nil && r.State == domain.MergeShipNone {
		r.State = domain.MergeShipFailed
		if strings.TrimSpace(r.ErrorMessage) == "" {
			r.ErrorMessage = shipOpErr.Error()
		}
	}
	cfj, _ := json.Marshal(r.ConflictFiles)
	if len(cfj) == 0 {
		cfj = []byte("[]")
	}
	row.mergeShipState = string(r.State)
	row.mergedSHA = strings.TrimSpace(r.MergedSHA)
	row.mergeError = strings.TrimSpace(r.ErrorMessage)
	row.conflictFilesJSON = string(cfj)
	row.leaseOwner = ""
	row.leaseExpiresAt = time.Time{}
	now := time.Now().UTC()
	switch r.State {
	case domain.MergeShipMerged, domain.MergeShipSkipped:
		row.status = "done"
		row.completedAt = now
	}
	return nil
}

func (s *WorkspaceStore) ReleaseShipLease(_ context.Context, rowID int64, leaseOwner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.mq {
		if s.mq[i].id == rowID && strings.TrimSpace(s.mq[i].leaseOwner) == strings.TrimSpace(leaseOwner) {
			s.mq[i].leaseOwner = ""
			s.mq[i].leaseExpiresAt = time.Time{}
			break
		}
	}
	return nil
}

var (
	_ ports.WorkspacePortRepository       = (*WorkspaceStore)(nil)
	_ ports.WorkspaceMergeQueueRepository = (*WorkspaceStore)(nil)
)
