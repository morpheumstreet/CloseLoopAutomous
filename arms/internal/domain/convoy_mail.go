package domain

import (
	"strings"
	"time"
)

// ConvoyMailMessage is an append-only inter-subtask message on a convoy.
type ConvoyMailMessage struct {
	ID        string
	ConvoyID  ConvoyID
	SubtaskID SubtaskID // legacy: same as FromSubtaskID when reading older rows
	FromSubtaskID SubtaskID
	ToSubtaskID   SubtaskID // empty = convoy-wide / broadcast
	Kind          string    // note | handoff | blocker
	Body          string
	CreatedAt     time.Time
}

// ConvoyMailDraft is input for Append (HTTP maps subtask_id → FromSubtaskID).
type ConvoyMailDraft struct {
	FromSubtaskID SubtaskID
	ToSubtaskID   SubtaskID
	Kind          string
	Body          string
}

// NormalizeConvoyMailKind returns note | handoff | blocker.
func NormalizeConvoyMailKind(k string) string {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "handoff", "blocker", "note":
		return strings.ToLower(strings.TrimSpace(k))
	default:
		return "note"
	}
}
