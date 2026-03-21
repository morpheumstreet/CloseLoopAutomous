package gateway

import (
	"context"
	"fmt"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Stub simulates OpenClaw dispatch without a real WebSocket connection.
type Stub struct {
	Seq int
}

func (s *Stub) DispatchTask(_ context.Context, task domain.Task) (string, error) {
	s.Seq++
	return fmt.Sprintf("gw-task-%s-%d", task.ID, s.Seq), nil
}

func (s *Stub) DispatchSubtask(_ context.Context, parent domain.TaskID, sub domain.Subtask) (string, error) {
	s.Seq++
	return fmt.Sprintf("gw-sub-%s-%s-%d", parent, sub.ID, s.Seq), nil
}

var _ ports.AgentGateway = (*Stub)(nil)
