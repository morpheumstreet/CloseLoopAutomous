package ports

import (
	"context"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// AgentGateway is the execution plane: arms pushes work to an external agent runtime and receives
// completion via webhooks or operator APIs. Implementations include in-process stubs, OpenClaw-class
// WebSocket gateways (including ZeroClaw, Clawlet, IronClaw, NemoClaw), NullClaw HTTP A2A (/a2a), CoPaw Console JSON-RPC (/console/api), MetaClaw OpenAI HTTP (/v1/chat/completions), InkOS CLI, and other runtime adapters.
type AgentGateway interface {
	DispatchTask(ctx context.Context, task domain.Task) (externalRef string, err error)
	DispatchSubtask(ctx context.Context, parent domain.Task, sub domain.Subtask) (externalRef string, err error)
}
