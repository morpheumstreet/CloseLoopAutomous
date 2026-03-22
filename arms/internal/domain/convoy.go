package domain

import "time"

// Convoy models a DAG of dependent subtasks (Convoy Mode).
type Convoy struct {
	ID        ConvoyID
	ProductID ProductID
	ParentID  TaskID
	Subtasks  []Subtask
	// MetadataJSON is opaque convoy-level graph/plan metadata (labels, UI hints); default "{}".
	MetadataJSON string
	CreatedAt    time.Time
}

type Subtask struct {
	ID        SubtaskID
	DependsOn []SubtaskID
	AgentRole string // e.g. builder, tester, reviewer
	Title     string
	// MetadataJSON is per-node DAG metadata (MC-style annotations); default "{}".
	MetadataJSON string
	// DagLayer is the longest dependency chain length to this node (roots = 0); computed on save/create.
	DagLayer int
	Dispatched   bool
	Completed    bool // set after agent reports done (webhook); gates dependents for dispatch
	ExternalRef  string
	LastCheckpoint string
	// DispatchAttempts counts failed Gateway.DispatchSubtask calls while not yet dispatched (for retry cap).
	DispatchAttempts int
}
