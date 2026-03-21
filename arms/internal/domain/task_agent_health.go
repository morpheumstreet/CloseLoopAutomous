package domain

import "time"

// TaskAgentHealth is execution-plane liveness for a single task (OpenClaw / agent slot).
type TaskAgentHealth struct {
	TaskID          TaskID
	ProductID       ProductID
	Status          string // e.g. unknown, healthy, busy, error
	DetailJSON      string
	LastHeartbeatAt time.Time
}
