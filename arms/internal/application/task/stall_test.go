package task

import (
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestStalledTaskState(t *testing.T) {
	now := time.Unix(2000, 0)
	thresh := 5 * time.Minute
	t.Run("planning not stalled", func(t *testing.T) {
		ta := &domain.Task{Status: domain.StatusPlanning}
		st, r := StalledTaskState(now, thresh, ta, nil)
		if st || r != "" {
			t.Fatalf("got stalled=%v reason=%q", st, r)
		}
	})
	t.Run("no heartbeat", func(t *testing.T) {
		ta := &domain.Task{Status: domain.StatusInProgress}
		st, r := StalledTaskState(now, thresh, ta, nil)
		if !st || r != "no_heartbeat" {
			t.Fatalf("got stalled=%v reason=%q", st, r)
		}
	})
	t.Run("fresh heartbeat", func(t *testing.T) {
		ta := &domain.Task{Status: domain.StatusInProgress}
		h := &domain.TaskAgentHealth{LastHeartbeatAt: now.Add(-1 * time.Minute)}
		st, r := StalledTaskState(now, thresh, ta, h)
		if st {
			t.Fatalf("unexpected stall reason=%q", r)
		}
	})
	t.Run("stale heartbeat", func(t *testing.T) {
		ta := &domain.Task{Status: domain.StatusInProgress}
		h := &domain.TaskAgentHealth{LastHeartbeatAt: now.Add(-10 * time.Minute)}
		st, r := StalledTaskState(now, thresh, ta, h)
		if !st || r != "heartbeat_stale" {
			t.Fatalf("got stalled=%v reason=%q", st, r)
		}
	})
}
