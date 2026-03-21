package ports

import (
	"context"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

// ExecutionAgentRegistry persists registered agent slots (GET /api/agents registry).
type ExecutionAgentRegistry interface {
	Save(ctx context.Context, a *domain.ExecutionAgent) error
	ByID(ctx context.Context, id string) (*domain.ExecutionAgent, error)
	List(ctx context.Context, limit int) ([]domain.ExecutionAgent, error)
}

// AgentMailboxRepository is append-only mail per execution agent.
type AgentMailboxRepository interface {
	Append(ctx context.Context, id, agentID string, taskID domain.TaskID, body string, at time.Time) error
	ListByAgent(ctx context.Context, agentID string, limit int) ([]domain.AgentMailboxMessage, error)
}
