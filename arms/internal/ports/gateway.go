package ports

import (
	"context"

	"github.com/closeloopautomous/arms/internal/domain"
)

// AgentGateway is the execution plane (OpenClaw-style): LLM + tools live behind this port.
type AgentGateway interface {
	DispatchTask(ctx context.Context, task domain.Task) (externalRef string, err error)
	DispatchSubtask(ctx context.Context, parent domain.Task, sub domain.Subtask) (externalRef string, err error)
}
