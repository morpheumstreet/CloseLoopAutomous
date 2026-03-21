package domain

import "time"

// Convoy models a DAG of dependent subtasks (Convoy Mode).
type Convoy struct {
	ID        ConvoyID
	ProductID ProductID
	ParentID  TaskID
	Subtasks  []Subtask
	CreatedAt time.Time
}

type Subtask struct {
	ID           SubtaskID
	DependsOn    []SubtaskID
	AgentRole    string // e.g. builder, tester, reviewer
	Dispatched   bool
	Completed    bool // set after agent reports done (webhook); gates dependents for dispatch
	ExternalRef  string
	LastCheckpoint string
}
