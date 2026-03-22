package task

import (
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

// StalledTaskState reports whether a task is stalled relative to agent health and a heartbeat staleness threshold.
// health may be nil (treated as no_heartbeat when the status expects heartbeats).
func StalledTaskState(now time.Time, staleThreshold time.Duration, t *domain.Task, health *domain.TaskAgentHealth) (stalled bool, reason string) {
	if t == nil || !domain.TaskExpectsAgentHeartbeat(t.Status) {
		return false, ""
	}
	if health == nil {
		return true, "no_heartbeat"
	}
	if now.Sub(health.LastHeartbeatAt) > staleThreshold {
		return true, "heartbeat_stale"
	}
	return false, ""
}
