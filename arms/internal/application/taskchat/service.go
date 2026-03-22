package taskchat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service is per-task operator/agent chat + product-scoped queued notes.
type Service struct {
	Chat     ports.TaskChatRepository
	Tasks    ports.TaskRepository
	Products ports.ProductRepository
	Clock    ports.Clock
	IDs      ports.IdentityGenerator
	Events   ports.LiveActivityPublisher
}

// Append adds a chat line on a task. queue=true marks an operator note pending on the product queue.
func (s *Service) Append(ctx context.Context, taskID domain.TaskID, body string, author string, queue bool) (*domain.TaskChatMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("%w: body required", domain.ErrInvalidInput)
	}
	t, err := s.Tasks.ByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := s.Products.ByID(ctx, t.ProductID); err != nil {
		return nil, err
	}
	now := s.Clock.Now()
	msg := &domain.TaskChatMessage{
		ID:           s.IDs.NewTaskChatMessageID(),
		ProductID:    t.ProductID,
		TaskID:       taskID,
		Author:       domain.NormalizeTaskChatAuthor(author),
		Body:         body,
		QueuePending: queue,
		CreatedAt:    now,
	}
	if err := s.Chat.Append(ctx, msg); err != nil {
		return nil, err
	}
	if s.Events != nil {
		_ = s.Events.Publish(ctx, ports.LiveActivityEvent{
			Type:      "task_chat_message",
			Ts:        now.UTC().Format(time.RFC3339Nano),
			ProductID: string(t.ProductID),
			TaskID:    string(taskID),
			Data: map[string]any{
				"message_id":    msg.ID,
				"author":        msg.Author,
				"queue_pending": msg.QueuePending,
				"body_preview":  trimPreview(body, 200),
			},
		})
	}
	return msg, nil
}

func trimPreview(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ListByTask returns chronological chat for a task (oldest first among the last `limit` rows).
func (s *Service) ListByTask(ctx context.Context, taskID domain.TaskID, limit int) ([]domain.TaskChatMessage, error) {
	if _, err := s.Tasks.ByID(ctx, taskID); err != nil {
		return nil, err
	}
	return s.Chat.ListByTask(ctx, taskID, limit)
}

// ListQueue lists pending queued notes for a product (oldest first).
func (s *Service) ListQueue(ctx context.Context, productID domain.ProductID, limit int) ([]domain.TaskChatMessage, error) {
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.Chat.ListPendingQueueByProduct(ctx, productID, limit)
}

// AckQueue clears queue_pending for a message; productID must match the row.
func (s *Service) AckQueue(ctx context.Context, productID domain.ProductID, messageID string) error {
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return err
	}
	m, err := s.Chat.ByID(ctx, messageID)
	if err != nil {
		return err
	}
	if m.ProductID != productID {
		return fmt.Errorf("%w: message belongs to another product", domain.ErrInvalidInput)
	}
	if err := s.Chat.ClearQueuePending(ctx, messageID); err != nil {
		return err
	}
	if s.Events != nil {
		now := s.Clock.Now()
		_ = s.Events.Publish(ctx, ports.LiveActivityEvent{
			Type:      "task_chat_queue_ack",
			Ts:        now.UTC().Format(time.RFC3339Nano),
			ProductID: string(productID),
			TaskID:    string(m.TaskID),
			Data: map[string]any{
				"message_id": messageID,
			},
		})
	}
	return nil
}
