package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// AutoStallReassignSettings gates auto re-dispatch to another execution agent on stall (#107).
// Requires AutoStallNudge sweep (same tick), Redis + worker, AgentHealth, ExecutionAgent registry, and a real gateway.
type AutoStallReassignSettings struct {
	Enabled   bool
	Cooldown  time.Duration
	MaxPerDay int // 0 = no cap
}

func (s *Service) pickLeastLoadedExecutionAgent(ctx context.Context, productID domain.ProductID, excludeID string) (string, error) {
	if s.ExecAgents == nil || s.Tasks == nil {
		return "", nil
	}
	agents, err := s.ExecAgents.ListByProduct(ctx, productID, 500)
	if err != nil || len(agents) == 0 {
		return "", err
	}
	ex := strings.TrimSpace(excludeID)
	var candidates []domain.ExecutionAgent
	for _, a := range agents {
		if a.ID == ex {
			continue
		}
		candidates = append(candidates, a)
	}
	if len(candidates) == 0 {
		return "", nil
	}
	best := candidates[0].ID
	bestLoad := int(^uint(0) >> 1)
	for _, a := range candidates {
		load, err := s.Tasks.CountByExecutionAgent(ctx, productID, a.ID)
		if err != nil {
			return "", err
		}
		if load < bestLoad {
			bestLoad = load
			best = a.ID
		}
	}
	return best, nil
}

// tryAutoReassignIfDue attempts gateway re-dispatch to another execution agent. Caller already verified stall.
func (s *Service) tryAutoReassignIfDue(ctx context.Context, t *domain.Task, stallReason string) (reassigned bool, newAgentID string, skipReason string, err error) {
	if t == nil {
		return false, "", "", nil
	}
	if !s.AutoStallReassign.Enabled {
		return false, "", "reassign_disabled", nil
	}
	if s.ExecAgents == nil || s.Gateway == nil || s.Tasks == nil {
		return false, "", "reassign_not_configured", nil
	}
	if t.Status == domain.StatusConvoyActive {
		return false, "", "convoy_active", nil
	}
	switch t.Status {
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusReview:
	default:
		return false, "", "not_eligible_status", nil
	}
	if s.AgentHealth == nil {
		return false, "", "no_agent_health", nil
	}
	now := s.Clock.Now()
	var detail string
	if row, herr := s.AgentHealth.ByTask(ctx, t.ID); herr == nil && row != nil {
		detail = row.DetailJSON
	}
	if last := lastAutoReassignAt(detail); !last.IsZero() && now.Sub(last) < s.AutoStallReassign.Cooldown {
		return false, "", "reassign_cooldown", nil
	}
	if s.AutoStallReassign.MaxPerDay > 0 {
		n := countAutoReassignsSince(detail, now.Add(-24 * time.Hour))
		if n >= s.AutoStallReassign.MaxPerDay {
			return false, "", "reassign_max_per_day", nil
		}
	}
	newID, err := s.pickLeastLoadedExecutionAgent(ctx, t.ProductID, strings.TrimSpace(t.CurrentExecutionAgentID))
	if err != nil {
		return false, "", "", err
	}
	if newID == "" {
		return false, "", "no_alternate_agent", nil
	}
	if err := s.redispatchStalledTask(ctx, t.ID, newID, stallReason); err != nil {
		return false, "", "", err
	}
	return true, newID, "", nil
}

func (s *Service) redispatchStalledTask(ctx context.Context, taskID domain.TaskID, newAgentID, stallReason string) error {
	t, err := s.taskWithActiveProduct(ctx, taskID)
	if err != nil {
		return err
	}
	switch t.Status {
	case domain.StatusInProgress, domain.StatusTesting, domain.StatusReview:
	default:
		return fmt.Errorf("%w: reassign only for in_progress, testing, review (got %s)", domain.ErrInvalidTransition, t.Status)
	}
	if err := s.Budget.AssertWithinBudget(ctx, t.ProductID, 0); err != nil {
		return err
	}
	t.CurrentExecutionAgentID = newAgentID
	ref, err := s.Gateway.DispatchTask(ctx, *t)
	if err != nil {
		if errors.Is(err, domain.ErrNoDispatchTarget) {
			return err
		}
		return fmt.Errorf("%w: %v", domain.ErrGateway, err)
	}
	now := s.Clock.Now()
	note := fmt.Sprintf("auto: reassign to %s (%s)", newAgentID, stallReason)
	line := fmt.Sprintf("[stall_reassign %s] %s", now.UTC().Format(time.RFC3339Nano), note)
	reason := strings.TrimSpace(t.StatusReason)
	if reason != "" {
		reason = line + "; " + reason
	} else {
		reason = line
	}
	t.ExternalRef = ref
	t.StatusReason = reason
	t.UpdatedAt = now

	var prevDetail string
	if s.AgentHealth != nil {
		if h, herr := s.AgentHealth.ByTask(ctx, taskID); herr == nil && h != nil {
			prevDetail = h.DetailJSON
		}
	}
	healthDetail := mergeStallReassignDetail(prevDetail, newAgentID, note, now)

	ev := ports.LiveActivityEvent{
		Type:      "task_execution_reassigned",
		Ts:        now.UTC().Format(time.RFC3339Nano),
		ProductID: string(t.ProductID),
		TaskID:    string(taskID),
		Data: map[string]any{
			"execution_agent_id": newAgentID,
			"external_ref":       ref,
			"stall_reason":       stallReason,
			"source":             "auto",
		},
	}

	if s.LiveTX != nil {
		if err := s.LiveTX.SaveTaskWithEvent(ctx, t, ev); err != nil {
			return err
		}
		if s.AgentHealth != nil {
			st := string(t.Status)
			_ = s.AgentHealth.UpsertHeartbeat(ctx, taskID, t.ProductID, st, healthDetail, now)
		}
		return nil
	}
	if err := s.Tasks.Save(ctx, t); err != nil {
		return err
	}
	if s.AgentHealth != nil {
		st := string(t.Status)
		_ = s.AgentHealth.UpsertHeartbeat(ctx, taskID, t.ProductID, st, healthDetail, now)
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ev)
	}
	return nil
}

func mergeStallReassignDetail(existingJSON, agentID, note string, at time.Time) string {
	var m map[string]any
	if strings.TrimSpace(existingJSON) != "" && json.Valid([]byte(existingJSON)) {
		_ = json.Unmarshal([]byte(existingJSON), &m)
	}
	if m == nil {
		m = map[string]any{}
	}
	var arr []any
	if raw, ok := m["stall_reassigns"].([]any); ok {
		arr = raw
	}
	entry := map[string]any{
		"at":        at.UTC().Format(time.RFC3339Nano),
		"agent_id":  agentID,
		"note":      note,
	}
	arr = append(arr, entry)
	m["stall_reassigns"] = arr
	b, err := json.Marshal(m)
	if err != nil {
		return `{"stall_reassigns":[]}`
	}
	return string(b)
}

func lastAutoReassignAt(detailJSON string) time.Time {
	var m map[string]any
	if strings.TrimSpace(detailJSON) == "" || !json.Valid([]byte(detailJSON)) {
		return time.Time{}
	}
	if err := json.Unmarshal([]byte(detailJSON), &m); err != nil {
		return time.Time{}
	}
	raw, ok := m["stall_reassigns"].([]any)
	if !ok {
		return time.Time{}
	}
	var latest time.Time
	for _, e := range raw {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		note, _ := em["note"].(string)
		if !strings.HasPrefix(strings.TrimSpace(note), "auto:") {
			continue
		}
		atStr, _ := em["at"].(string)
		tm := parseStallNudgeAt(atStr)
		if !tm.IsZero() && tm.After(latest) {
			latest = tm
		}
	}
	return latest
}

func countAutoReassignsSince(detailJSON string, since time.Time) int {
	var m map[string]any
	if strings.TrimSpace(detailJSON) == "" || !json.Valid([]byte(detailJSON)) {
		return 0
	}
	if err := json.Unmarshal([]byte(detailJSON), &m); err != nil {
		return 0
	}
	raw, ok := m["stall_reassigns"].([]any)
	if !ok {
		return 0
	}
	n := 0
	for _, e := range raw {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		note, _ := em["note"].(string)
		if !strings.HasPrefix(strings.TrimSpace(note), "auto:") {
			continue
		}
		atStr, _ := em["at"].(string)
		tm := parseStallNudgeAt(atStr)
		if !tm.IsZero() && !tm.Before(since) {
			n++
		}
	}
	return n
}
