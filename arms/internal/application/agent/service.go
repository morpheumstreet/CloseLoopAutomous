package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// ErrRegistryDisabled is returned when the execution-agent registry port is not wired.
var ErrRegistryDisabled = errors.New("agent registry not configured")

// Service is the execution-agent registry + mailbox (not task heartbeats).
type Service struct {
	Registry ports.ExecutionAgentRegistry
	Mailbox  ports.AgentMailboxRepository
	Clock    ports.Clock
	IDs      ports.IdentityGenerator
}

// Register creates or updates a logical agent slot.
func (s *Service) Register(ctx context.Context, displayName string, productID domain.ProductID, source, externalRef string) (*domain.ExecutionAgent, error) {
	if s == nil || s.Registry == nil {
		return nil, ErrRegistryDisabled
	}
	id := s.IDs.NewExecutionAgentID()
	src := strings.TrimSpace(source)
	if src == "" {
		src = "manual"
	}
	a := &domain.ExecutionAgent{
		ID:          id,
		DisplayName: strings.TrimSpace(displayName),
		ProductID:   productID,
		Source:      src,
		ExternalRef: strings.TrimSpace(externalRef),
		CreatedAt:   s.Clock.Now(),
	}
	if err := s.Registry.Save(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// List returns newest agents first.
func (s *Service) List(ctx context.Context, limit int) ([]domain.ExecutionAgent, error) {
	if s == nil || s.Registry == nil {
		return nil, nil
	}
	return s.Registry.List(ctx, limit)
}

// PostMailbox appends a message; agent must exist.
func (s *Service) PostMailbox(ctx context.Context, agentID string, taskID domain.TaskID, body string) error {
	if s == nil || s.Mailbox == nil || s.Registry == nil {
		return ErrRegistryDisabled
	}
	if _, err := s.Registry.ByID(ctx, agentID); err != nil {
		return err
	}
	msg := strings.TrimSpace(body)
	if msg == "" {
		return fmt.Errorf("%w: body required", domain.ErrInvalidInput)
	}
	return s.Mailbox.Append(ctx, s.IDs.NewMailboxMessageID(), agentID, taskID, msg, s.Clock.Now())
}

// ListMailbox returns newest first.
func (s *Service) ListMailbox(ctx context.Context, agentID string, limit int) ([]domain.AgentMailboxMessage, error) {
	if s == nil || s.Mailbox == nil || s.Registry == nil {
		return nil, nil
	}
	if _, err := s.Registry.ByID(ctx, agentID); err != nil {
		return nil, err
	}
	return s.Mailbox.ListByAgent(ctx, agentID, limit)
}
