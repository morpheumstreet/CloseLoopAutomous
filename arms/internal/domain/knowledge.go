package domain

import "time"

// KnowledgeEntry is a product-scoped knowledge snippet for retrieval at dispatch and operator CRUD (#90).
type KnowledgeEntry struct {
	ID           int64
	ProductID    ProductID
	TaskID       TaskID // optional; empty if not task-scoped
	Content      string
	MetadataJSON string // object JSON; "{}" when unset
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
