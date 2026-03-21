package cost

import (
	"context"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service records spend events for observability and budget inputs.
type Service struct {
	Costs  ports.CostRepository
	Caps   ports.CostCapRepository
	Clock  ports.Clock
	IDs    ports.IdentityGenerator
	Events ports.LiveActivityPublisher // optional live activity
	LiveTX ports.LiveActivityTX        // optional: SQLite cost row + outbox in one transaction
}

func (s *Service) Record(ctx context.Context, productID domain.ProductID, taskID domain.TaskID, amount float64, note, agent, model string) error {
	e := domain.CostEvent{
		ID:        s.IDs.NewCostEventID(),
		ProductID: productID,
		TaskID:    taskID,
		Amount:    amount,
		Note:      note,
		Agent:     agent,
		Model:     model,
		At:        s.Clock.Now(),
	}
	ev := ports.LiveActivityEvent{
		Type:      "cost_recorded",
		Ts:        s.Clock.Now().UTC().Format(time.RFC3339Nano),
		ProductID: string(productID),
		TaskID:    string(taskID),
		Data: map[string]any{
			"amount": amount,
			"agent":  agent,
			"model":  model,
		},
	}
	if s.LiveTX != nil {
		return s.LiveTX.AppendCostWithEvent(ctx, e, ev)
	}
	if err := s.Costs.Append(ctx, e); err != nil {
		return err
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ev)
	}
	return nil
}

// Breakdown returns cost events in a window plus simple aggregates.
func (s *Service) Breakdown(ctx context.Context, productID domain.ProductID, from, to time.Time) (map[string]any, error) {
	events, err := s.Costs.ListByProductBetween(ctx, productID, from, to)
	if err != nil {
		return nil, err
	}
	byAgent := make(map[string]float64)
	byModel := make(map[string]float64)
	var total float64
	list := make([]map[string]any, 0, len(events))
	for _, e := range events {
		total += e.Amount
		ak := e.Agent
		if ak == "" {
			ak = "(none)"
		}
		mk := e.Model
		if mk == "" {
			mk = "(none)"
		}
		byAgent[ak] += e.Amount
		byModel[mk] += e.Amount
		list = append(list, map[string]any{
			"id": e.ID, "task_id": string(e.TaskID), "amount": e.Amount,
			"note": e.Note, "agent": e.Agent, "model": e.Model,
			"at": e.At.UTC().Format(time.RFC3339Nano),
		})
	}
	out := map[string]any{
		"product_id": string(productID),
		"total":      total,
		"events":     list,
		"by_agent":   byAgent,
		"by_model":   byModel,
	}
	if !from.IsZero() {
		out["from"] = from.UTC().Format(time.RFC3339Nano)
	}
	if !to.IsZero() {
		out["to"] = to.UTC().Format(time.RFC3339Nano)
	}
	return out, nil
}

// PatchCaps merges non-nil patch fields into stored caps (upserts row). Use negative values to clear a limit.
func (s *Service) PatchCaps(ctx context.Context, productID domain.ProductID, daily, monthly, cumulative *float64) error {
	cur, err := s.Caps.Get(ctx, productID)
	if err != nil && err != domain.ErrNotFound {
		return err
	}
	var next domain.ProductCostCaps
	next.ProductID = productID
	if cur != nil {
		next.DailyCap = cur.DailyCap
		next.MonthlyCap = cur.MonthlyCap
		next.CumulativeCap = cur.CumulativeCap
	}
	applyCapPatch := func(patch *float64, target **float64) {
		if patch == nil {
			return
		}
		if *patch < 0 {
			*target = nil
			return
		}
		v := *patch
		*target = &v
	}
	applyCapPatch(daily, &next.DailyCap)
	applyCapPatch(monthly, &next.MonthlyCap)
	applyCapPatch(cumulative, &next.CumulativeCap)
	return s.Caps.Upsert(ctx, &next)
}
