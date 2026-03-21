package domain

import "time"

// CheckpointHistoryEntry is one persisted checkpoint revision for restore APIs.
type CheckpointHistoryEntry struct {
	ID        int64
	TaskID    TaskID
	Payload   string
	CreatedAt time.Time
}
