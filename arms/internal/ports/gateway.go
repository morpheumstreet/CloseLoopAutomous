package ports

import (
	"context"

	"github.com/closeloopautomous/arms/internal/domain"
)

// AgentGateway is the execution plane: arms pushes work to an external agent runtime and receives
// completion via webhooks or operator APIs. Implementations include in-process stubs, OpenClaw-class
// WebSocket gateways (including ZeroClaw, Clawlet, IronClaw), NullClaw HTTP A2A (/a2a), and other runtime adapters.
type AgentGateway interface {
	DispatchTask(ctx context.Context, task domain.Task) (externalRef string, err error)
	DispatchSubtask(ctx context.Context, parent domain.Task, sub domain.Subtask) (externalRef string, err error)
}
