package domain

import "strings"

// AgentHealthBlocksConvoyDispatch returns true when the parent task's agent-health status
// should defer POST …/dispatch-ready (no subtasks dispatched this call; handler still 200).
func AgentHealthBlocksConvoyDispatch(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "error", "stalled", "failed", "dead", "offline":
		return true
	default:
		return false
	}
}
