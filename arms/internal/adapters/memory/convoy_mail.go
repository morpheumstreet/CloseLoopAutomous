package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

const convoyMailMax = 5000

type ConvoyMailStore struct {
	mu   sync.Mutex
	rows []domain.ConvoyMailMessage // append order
}

func NewConvoyMailStore() *ConvoyMailStore {
	return &ConvoyMailStore{}
}

var _ ports.ConvoyMailRepository = (*ConvoyMailStore)(nil)

func (s *ConvoyMailStore) Append(_ context.Context, convoyID domain.ConvoyID, msg domain.ConvoyMailDraft, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	from := msg.FromSubtaskID
	kind := domain.NormalizeConvoyMailKind(msg.Kind)
	m := domain.ConvoyMailMessage{
		ID:            uuid.NewString(),
		ConvoyID:      convoyID,
		SubtaskID:     from,
		FromSubtaskID: from,
		ToSubtaskID:   msg.ToSubtaskID,
		Kind:          kind,
		Body:          msg.Body,
		CreatedAt:     at.UTC(),
	}
	s.rows = append(s.rows, m)
	if len(s.rows) > convoyMailMax {
		s.rows = s.rows[len(s.rows)-convoyMailMax:]
	}
	return nil
}

func (s *ConvoyMailStore) ListByConvoy(_ context.Context, convoyID domain.ConvoyID, limit int) ([]domain.ConvoyMailMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.ConvoyMailMessage
	for i := len(s.rows) - 1; i >= 0 && len(out) < limit; i-- {
		if s.rows[i].ConvoyID == convoyID {
			out = append(out, s.rows[i])
		}
	}
	return out, nil
}
