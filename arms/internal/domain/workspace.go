package domain

import "time"

// AllocatedPort is a reserved dev-server port in the MC-style range 4200–4299.
type AllocatedPort struct {
	Port        int
	ProductID   ProductID
	TaskID      TaskID
	AllocatedAt time.Time
}

// MergeQueueEntry is a pending or historical merge-queue row (serialized merge lane).
type MergeQueueEntry struct {
	ID        int64
	ProductID ProductID
	TaskID    TaskID
	Status    string // e.g. pending
	CreatedAt time.Time
}
