package convoy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

const defaultMaxSubtaskDispatchAttempts = 5

// Service builds a convoy DAG and dispatches subtasks through the same gateway port as single tasks.
type Service struct {
	Convoys  ports.ConvoyRepository
	Tasks    ports.TaskRepository
	Products ports.ProductRepository
	Gateway  ports.AgentGateway
	Budget   ports.BudgetPolicy // optional; when set, enforces caps per subtask dispatch (like task.Dispatch)
	Health   ports.AgentHealthRepository // optional; parent task agent-health can defer dispatch-ready
	Mail     ports.ConvoyMailRepository  // optional; nil → mail routes return not configured
	// MaxSubtaskDispatchAttempts caps failed gateway calls per subtask before ErrGateway (0 → defaultMaxSubtaskDispatchAttempts).
	MaxSubtaskDispatchAttempts int
	Clock    ports.Clock
	IDs      ports.IdentityGenerator
	Events   ports.LiveActivityPublisher // optional: SSE / outbox on dispatch + subtask completion
}

func (s *Service) maxDispatchAttempts() int {
	if s.MaxSubtaskDispatchAttempts > 0 {
		return s.MaxSubtaskDispatchAttempts
	}
	return defaultMaxSubtaskDispatchAttempts
}

// Create attaches subtasks to a parent task (roles + dependencies only; no dispatch).
func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Convoy, error) {
	parent, err := s.Tasks.ByID(ctx, in.ParentTaskID)
	if err != nil {
		return nil, err
	}
	if err := ports.RequireActiveProduct(ctx, s.Products, parent.ProductID); err != nil {
		return nil, err
	}
	subtasks := in.Subtasks
	for i := range subtasks {
		if subtasks[i].ID == "" {
			subtasks[i].ID = s.IDs.NewSubtaskID()
		}
		mj, err := normalizeMetadataJSON(subtasks[i].MetadataJSON)
		if err != nil {
			return nil, fmt.Errorf("%w: subtask metadata_json must be a JSON object", domain.ErrInvalidInput)
		}
		subtasks[i].MetadataJSON = mj
	}
	if err := domain.ValidateConvoySubtasks(subtasks); err != nil {
		return nil, err
	}
	applyDagLayers(subtasks)
	meta, err := normalizeMetadataJSON(in.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("%w: metadata_json must be a JSON object", domain.ErrInvalidInput)
	}
	c := &domain.Convoy{
		ID:           s.IDs.NewConvoyID(),
		ProductID:    in.ProductID,
		ParentID:     in.ParentTaskID,
		Subtasks:     subtasks,
		MetadataJSON: meta,
		CreatedAt:    s.Clock.Now(),
	}
	if err := s.Convoys.Save(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// Get returns a convoy by id or ErrNotFound.
func (s *Service) Get(ctx context.Context, id domain.ConvoyID) (*domain.Convoy, error) {
	return s.Convoys.ByID(ctx, id)
}

// Graph returns MC-style DAG detail: stable topological order, explicit edges, and dag_layer buckets.
func (s *Service) Graph(ctx context.Context, id domain.ConvoyID) (*GraphDetail, error) {
	c, err := s.Convoys.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	order, err := StableTopologicalSubtaskOrder(c.Subtasks)
	if err != nil {
		return nil, fmt.Errorf("%w: convoy graph invalid (cycle or corrupt): %v", domain.ErrInvalidInput, err)
	}
	edgeCount := 0
	maxLayer := 0
	for i := range c.Subtasks {
		edgeCount += len(c.Subtasks[i].DependsOn)
		if c.Subtasks[i].DagLayer > maxLayer {
			maxLayer = c.Subtasks[i].DagLayer
		}
	}
	return &GraphDetail{
		ConvoyID:         string(id),
		TopologicalOrder: order,
		Edges:            SubtaskDependencyEdges(c.Subtasks),
		Layers:           SubtaskLayers(c.Subtasks),
		GraphSummary: map[string]any{
			"node_count": len(c.Subtasks),
			"edge_count": edgeCount,
			"max_depth":  maxLayer,
		},
	}, nil
}

// ListByProduct returns convoys for a product (newest first), or ErrNotFound if the product does not exist.
func (s *Service) ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Convoy, error) {
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.Convoys.ListByProduct(ctx, productID)
}

// DispatchReady dispatches subtasks whose dependencies are already completed (one wave).
// estimatedCostPerSubtask is passed to Budget.AssertWithinBudget once per subtask about to be dispatched (parity with POST /api/tasks/{id}/dispatch).
// The returned count is how many subtasks transitioned to dispatched in this call (0 when none ready, parent health blocks, or all already dispatched).
func (s *Service) DispatchReady(ctx context.Context, convoyID domain.ConvoyID, estimatedCostPerSubtask float64) (dispatched int, err error) {
	c, err := s.Convoys.ByID(ctx, convoyID)
	if err != nil {
		return 0, err
	}
	parent, err := s.Tasks.ByID(ctx, c.ParentID)
	if err != nil {
		return 0, err
	}
	if s.Health != nil {
		h, herr := s.Health.ByTask(ctx, parent.ID)
		if herr == nil && h != nil && domain.AgentHealthBlocksConvoyDispatch(h.Status) {
			return 0, nil
		}
	}
	completed := make(map[domain.SubtaskID]bool, len(c.Subtasks))
	for i := range c.Subtasks {
		if c.Subtasks[i].Completed {
			completed[c.Subtasks[i].ID] = true
		}
	}
	for i := range c.Subtasks {
		st := &c.Subtasks[i]
		if st.Dispatched {
			continue
		}
		ready := true
		for _, dep := range st.DependsOn {
			if !completed[dep] {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}
		if s.Budget != nil {
			if err := s.Budget.AssertWithinBudget(ctx, c.ProductID, estimatedCostPerSubtask); err != nil {
				return dispatched, err
			}
		}
		ref, err := s.Gateway.DispatchSubtask(ctx, *parent, *st)
		if err != nil {
			if errors.Is(err, domain.ErrNoDispatchTarget) {
				return dispatched, err
			}
			st.DispatchAttempts++
			if saveErr := s.Convoys.Save(ctx, c); saveErr != nil {
				return dispatched, saveErr
			}
			maxA := s.maxDispatchAttempts()
			if st.DispatchAttempts >= maxA {
				return dispatched, fmt.Errorf("%w: subtask %s after %d attempts: %v", domain.ErrGateway, st.ID, st.DispatchAttempts, err)
			}
			continue
		}
		st.DispatchAttempts = 0
		st.Dispatched = true
		st.ExternalRef = ref
		dispatched++
		if s.Events != nil {
			_ = s.Events.Publish(ctx, ports.LiveActivityEvent{
				Type:      "convoy_subtask_dispatched",
				Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
				ProductID: string(c.ProductID),
				TaskID:    string(parent.ID),
				Data: map[string]any{
					"convoy_id":    string(c.ID),
					"subtask_id":   string(st.ID),
					"agent_role":   st.AgentRole,
					"external_ref": ref,
				},
			})
		}
	}
	if err := s.Convoys.Save(ctx, c); err != nil {
		return dispatched, err
	}
	return dispatched, nil
}

// PostMail appends inter-subtask mail for a convoy.
func (s *Service) PostMail(ctx context.Context, convoyID domain.ConvoyID, msg domain.ConvoyMailDraft) error {
	if s.Mail == nil {
		return domain.ErrNotConfigured
	}
	msg.Body = strings.TrimSpace(msg.Body)
	if msg.Body == "" {
		return fmt.Errorf("%w: body required", domain.ErrInvalidInput)
	}
	if msg.FromSubtaskID == "" {
		return fmt.Errorf("%w: from subtask required", domain.ErrInvalidInput)
	}
	msg.Kind = domain.NormalizeConvoyMailKind(msg.Kind)
	c, err := s.Convoys.ByID(ctx, convoyID)
	if err != nil {
		return err
	}
	if !subtaskInConvoy(c, msg.FromSubtaskID) {
		return fmt.Errorf("%w: unknown from_subtask_id for convoy", domain.ErrInvalidInput)
	}
	if msg.ToSubtaskID != "" && !subtaskInConvoy(c, msg.ToSubtaskID) {
		return fmt.Errorf("%w: unknown to_subtask_id for convoy", domain.ErrInvalidInput)
	}
	return s.Mail.Append(ctx, convoyID, msg, s.Clock.Now())
}

func subtaskInConvoy(c *domain.Convoy, id domain.SubtaskID) bool {
	for i := range c.Subtasks {
		if c.Subtasks[i].ID == id {
			return true
		}
	}
	return false
}

// ListMail returns newest-first mail for a convoy.
func (s *Service) ListMail(ctx context.Context, convoyID domain.ConvoyID, limit int) ([]domain.ConvoyMailMessage, error) {
	if s.Mail == nil {
		return nil, domain.ErrNotConfigured
	}
	if _, err := s.Convoys.ByID(ctx, convoyID); err != nil {
		return nil, err
	}
	return s.Mail.ListByConvoy(ctx, convoyID, limit)
}

// CompleteSubtask marks a dispatched subtask finished (typically via agent-completion webhook).
// parentTaskID must match the convoy's parent task. Idempotent if already completed.
func (s *Service) CompleteSubtask(ctx context.Context, convoyID domain.ConvoyID, subtaskID domain.SubtaskID, parentTaskID domain.TaskID) error {
	c, err := s.Convoys.ByID(ctx, convoyID)
	if err != nil {
		return err
	}
	if c.ParentID != parentTaskID {
		return fmt.Errorf("%w: task_id does not match convoy parent", domain.ErrInvalidInput)
	}
	if err := ports.RequireActiveProduct(ctx, s.Products, c.ProductID); err != nil {
		return err
	}
	idx := -1
	for i := range c.Subtasks {
		if c.Subtasks[i].ID == subtaskID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return domain.ErrNotFound
	}
	st := &c.Subtasks[idx]
	if !st.Dispatched {
		return fmt.Errorf("%w: subtask not dispatched yet", domain.ErrInvalidTransition)
	}
	if st.Completed {
		return nil
	}
	st.Completed = true
	if err := s.Convoys.Save(ctx, c); err != nil {
		return err
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ports.LiveActivityEvent{
			Type:      "convoy_subtask_completed",
			Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
			ProductID: string(c.ProductID),
			TaskID:    string(parentTaskID),
			Data: map[string]any{
				"convoy_id":  string(c.ID),
				"subtask_id": string(subtaskID),
				"agent_role": st.AgentRole,
			},
		})
	}
	return nil
}

// GetByParentTask returns the convoy whose parent task id matches (Mission Control lookup key).
func (s *Service) GetByParentTask(ctx context.Context, parentID domain.TaskID) (*domain.Convoy, error) {
	return s.Convoys.ByParentTask(ctx, parentID)
}

// DeleteConvoy removes a convoy and its subtasks/edges (SQLite FK cascade).
func (s *Service) DeleteConvoy(ctx context.Context, id domain.ConvoyID) error {
	return s.Convoys.Delete(ctx, id)
}

// AppendSubtasks adds subtasks to the convoy for parentID and recomputes dag layers.
func (s *Service) AppendSubtasks(ctx context.Context, parentID domain.TaskID, extra []domain.Subtask) (*domain.Convoy, error) {
	c, err := s.Convoys.ByParentTask(ctx, parentID)
	if err != nil {
		return nil, err
	}
	parent, err := s.Tasks.ByID(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if err := ports.RequireActiveProduct(ctx, s.Products, parent.ProductID); err != nil {
		return nil, err
	}
	if c.ProductID != parent.ProductID {
		return nil, fmt.Errorf("%w: convoy product mismatch", domain.ErrInvalidInput)
	}
	for i := range extra {
		if extra[i].ID == "" {
			extra[i].ID = s.IDs.NewSubtaskID()
		}
		mj, err := normalizeMetadataJSON(extra[i].MetadataJSON)
		if err != nil {
			return nil, fmt.Errorf("%w: subtask metadata_json must be a JSON object", domain.ErrInvalidInput)
		}
		extra[i].MetadataJSON = mj
	}
	merged := append(append([]domain.Subtask(nil), c.Subtasks...), extra...)
	if err := domain.ValidateConvoySubtasks(merged); err != nil {
		return nil, err
	}
	applyDagLayers(merged)
	c.Subtasks = merged
	meta, err := MergeMCCompatIntoMetadata(c.MetadataJSON, MCCompatFields{UpdatedAt: s.Clock.Now()})
	if err != nil {
		return nil, err
	}
	c.MetadataJSON = meta
	if err := s.Convoys.Save(ctx, c); err != nil {
		return nil, err
	}
	return s.Convoys.ByID(ctx, c.ID)
}

// PatchMCCompat merges Mission Control–style mc_compat fields (e.g. status) into convoy metadata.
func (s *Service) PatchMCCompat(ctx context.Context, convoyID domain.ConvoyID, patch MCCompatFields) (*domain.Convoy, error) {
	c, err := s.Convoys.ByID(ctx, convoyID)
	if err != nil {
		return nil, err
	}
	if err := ports.RequireActiveProduct(ctx, s.Products, c.ProductID); err != nil {
		return nil, err
	}
	if patch.UpdatedAt.IsZero() {
		patch.UpdatedAt = s.Clock.Now()
	}
	meta, err := MergeMCCompatIntoMetadata(c.MetadataJSON, patch)
	if err != nil {
		return nil, err
	}
	c.MetadataJSON = meta
	if err := s.Convoys.Save(ctx, c); err != nil {
		return nil, err
	}
	return s.Convoys.ByID(ctx, c.ID)
}
