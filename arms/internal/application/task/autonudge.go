package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// AutoStallNudgeSettings gates periodic auto-nudges for stalled tasks (Phase 1 of #83).
// Zero value disables the feature (Enabled false).
type AutoStallNudgeSettings struct {
	Enabled        bool
	StaleThreshold time.Duration
	Cooldown       time.Duration
	MaxPerDay      int // 0 means no daily cap
}

// AutoNudgeResult describes the outcome of [Service.AutoNudgeStallIfDue].
type AutoNudgeResult struct {
	Nudged       bool
	Reassigned   bool   // true when [Service.tryAutoReassignIfDue] re-dispatched to another execution agent (#107)
	ReassignTo   string // new execution_agents id when Reassigned
	StallReason  string // no_heartbeat | heartbeat_stale when stalled
	SkipReason   string // disabled | no_agent_health | not_stalled | cooldown | max_per_day | reassign_* | no_alternate_agent | …
}

// AutoNudgeStallIfDue applies auto-nudge policy for one task: stalled heartbeat only, cooldown, optional daily cap.
func (s *Service) AutoNudgeStallIfDue(ctx context.Context, t *domain.Task) (AutoNudgeResult, error) {
	var out AutoNudgeResult
	if t == nil {
		return out, nil
	}
	if !s.AutoStallNudge.Enabled {
		out.SkipReason = "disabled"
		return out, nil
	}
	if s.AgentHealth == nil {
		out.SkipReason = "no_agent_health"
		return out, nil
	}
	now := s.Clock.Now()
	row, err := s.AgentHealth.ByTask(ctx, t.ID)
	if err != nil {
		return out, err
	}
	stalled, stallReason := StalledTaskState(now, s.AutoStallNudge.StaleThreshold, t, row)
	if !stalled {
		out.SkipReason = "not_stalled"
		return out, nil
	}
	if ok, newAg, _, rerr := s.tryAutoReassignIfDue(ctx, t, stallReason); rerr != nil {
		return out, rerr
	} else if ok {
		out.Reassigned = true
		out.ReassignTo = newAg
		out.StallReason = stallReason
		return out, nil
	}
	detail := ""
	if row != nil {
		detail = row.DetailJSON
	}
	if last := lastAutoStallNudgeAt(detail); !last.IsZero() && now.Sub(last) < s.AutoStallNudge.Cooldown {
		out.StallReason = stallReason
		out.SkipReason = "cooldown"
		return out, nil
	}
	if s.AutoStallNudge.MaxPerDay > 0 {
		n := countAutoStallNudgesSince(detail, now.Add(-24*time.Hour))
		if n >= s.AutoStallNudge.MaxPerDay {
			out.StallReason = stallReason
			out.SkipReason = "max_per_day"
			return out, nil
		}
	}
	note := fmt.Sprintf("auto: %s (system nudge)", stallReason)
	if err := s.nudgeStall(ctx, t.ID, note, true); err != nil {
		return out, err
	}
	out.Nudged = true
	out.StallReason = stallReason
	return out, nil
}

// RunAutoStallNudgeSweep scans all products and attempts auto-nudge per task. No-op when disabled or AgentHealth unset.
func (s *Service) RunAutoStallNudgeSweep(ctx context.Context, products ports.ProductRepository) error {
	if products == nil {
		return nil
	}
	if !s.AutoStallNudge.Enabled || s.AgentHealth == nil {
		return nil
	}
	prods, err := products.ListAll(ctx)
	if err != nil {
		return err
	}
	for i := range prods {
		tasks, err := s.Tasks.ListByProduct(ctx, prods[i].ID)
		if err != nil {
			continue
		}
		for j := range tasks {
			if _, err := s.AutoNudgeStallIfDue(ctx, &tasks[j]); err != nil {
				slog.Debug("stall autonudge task", "task_id", string(tasks[j].ID), "err", err)
			}
		}
	}
	return nil
}

func lastAutoStallNudgeAt(detailJSON string) time.Time {
	var m map[string]any
	if strings.TrimSpace(detailJSON) == "" || !json.Valid([]byte(detailJSON)) {
		return time.Time{}
	}
	if err := json.Unmarshal([]byte(detailJSON), &m); err != nil {
		return time.Time{}
	}
	raw, ok := m["stall_nudges"].([]any)
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
		t := parseStallNudgeAt(atStr)
		if !t.IsZero() && t.After(latest) {
			latest = t
		}
	}
	return latest
}

func countAutoStallNudgesSince(detailJSON string, since time.Time) int {
	var m map[string]any
	if strings.TrimSpace(detailJSON) == "" || !json.Valid([]byte(detailJSON)) {
		return 0
	}
	if err := json.Unmarshal([]byte(detailJSON), &m); err != nil {
		return 0
	}
	raw, ok := m["stall_nudges"].([]any)
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
		t := parseStallNudgeAt(atStr)
		if !t.IsZero() && !t.Before(since) {
			n++
		}
	}
	return n
}

func parseStallNudgeAt(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
