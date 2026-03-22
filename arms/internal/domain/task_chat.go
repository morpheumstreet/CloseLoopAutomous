package domain

import (
	"strings"
	"time"
)

// TaskChatMessage is one line in per-task operator/agent chat history.
type TaskChatMessage struct {
	ID           string
	ProductID    ProductID
	TaskID       TaskID
	Author       string // operator | agent | system
	Body         string
	QueuePending bool // operator queued note awaiting agent pickup
	CreatedAt    time.Time
}

// NormalizeTaskChatAuthor returns operator | agent | system.
func NormalizeTaskChatAuthor(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "agent", "system", "operator":
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return "operator"
	}
}
