package gateway

import (
	"context"
	"fmt"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// SimulationMockClaw simulates gateway dispatch without a real WebSocket connection
// (synthetic external_ref values for local dev and tests).
type SimulationMockClaw struct {
	Seq int
}

func (s *SimulationMockClaw) DispatchTask(_ context.Context, task domain.Task) (string, error) {
	s.Seq++
	return fmt.Sprintf("gw-task-%s-%d", task.ID, s.Seq), nil
}

func (s *SimulationMockClaw) DispatchSubtask(_ context.Context, parent domain.Task, sub domain.Subtask) (string, error) {
	s.Seq++
	return fmt.Sprintf("gw-sub-%s-%s-%d", parent.ID, sub.ID, s.Seq), nil
}

var _ ports.AgentGateway = (*SimulationMockClaw)(nil)
