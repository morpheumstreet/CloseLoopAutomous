package domain

import (
	"fmt"
	"strings"
	"time"
)

// TaskStatus matches Mission Control Kanban columns (snake_case) plus terminal / convoy helpers.
type TaskStatus string

const (
	StatusPlanning     TaskStatus = "planning"
	StatusInbox        TaskStatus = "inbox"
	StatusAssigned     TaskStatus = "assigned"
	StatusInProgress   TaskStatus = "in_progress"
	StatusTesting      TaskStatus = "testing"
	StatusReview       TaskStatus = "review"
	StatusDone         TaskStatus = "done"
	StatusFailed       TaskStatus = "failed"
	StatusConvoyActive TaskStatus = "convoy_active"
)

func (s TaskStatus) String() string { return string(s) }

// ParseTaskStatus normalizes MC-style input.
func ParseTaskStatus(v string) (TaskStatus, error) {
	t := TaskStatus(strings.ToLower(strings.TrimSpace(v)))
	switch t {
	case StatusPlanning, StatusInbox, StatusAssigned, StatusInProgress,
		StatusTesting, StatusReview, StatusDone, StatusFailed, StatusConvoyActive:
		return t, nil
	default:
		return "", fmt.Errorf("unknown task status %q", v)
	}
}

// AllowedKanbanTransition enforces a conservative subset of MC’s board moves (self allowed as no-op).
func AllowedKanbanTransition(from, to TaskStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case StatusPlanning:
		return to == StatusInbox
	case StatusInbox:
		return to == StatusAssigned || to == StatusPlanning || to == StatusConvoyActive
	case StatusAssigned:
		return to == StatusInProgress || to == StatusInbox || to == StatusFailed || to == StatusConvoyActive
	case StatusInProgress:
		return to == StatusTesting || to == StatusReview || to == StatusFailed || to == StatusAssigned || to == StatusConvoyActive
	case StatusTesting:
		return to == StatusReview || to == StatusInProgress || to == StatusFailed
	case StatusReview:
		return to == StatusDone || to == StatusTesting || to == StatusInProgress || to == StatusFailed
	case StatusDone, StatusFailed:
		return to == StatusPlanning || to == StatusInbox // retry / reopen
	case StatusConvoyActive:
		return to == StatusReview || to == StatusFailed || to == StatusInProgress || to == StatusInbox
	default:
		return false
	}
}

// TaskExpectsAgentHeartbeat is true when the task status should receive agent heartbeats (stall detection).
func TaskExpectsAgentHeartbeat(st TaskStatus) bool {
	switch st {
	case StatusInProgress, StatusTesting, StatusReview, StatusConvoyActive:
		return true
	default:
		return false
	}
}

// Task is orchestration state; execution happens in the external agent runtime (OpenClaw).
type Task struct {
	ID                 TaskID
	ProductID          ProductID
	IdeaID             IdeaID
	Spec               string // implementation / build spec (after planning)
	Status             TaskStatus
	StatusReason       string // free text when failed or blocked (MC-style)
	PlanApproved       bool   // planning gate cleared
	ClarificationsJSON string // optional Q&A JSON from planning (opaque to DB)
	Checkpoint         string
	ExternalRef        string
	SandboxPath        string // optional operator/CI path hint (worktree isolation metadata)
	WorktreePath       string
	// PullRequestURL / PullRequestNumber / PullRequestHeadBranch are set when a PR is opened (merge ship uses these).
	PullRequestURL       string
	PullRequestNumber    int
	PullRequestHeadBranch string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
