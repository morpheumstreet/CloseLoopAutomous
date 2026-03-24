package feedback

import (
	"context"
	"fmt"
	"strings"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// Service stores product-scoped external feedback (Mission Control product_feedback).
type Service struct {
	Products ports.ProductRepository
	Feedback ports.ProductFeedbackRepository
	Ideas    ports.IdeaRepository // optional; when set, validates idea_id belongs to product
	Clock    ports.Clock
	IDs      ports.IdentityGenerator
}

// AppendInput is POST body for new feedback.
type AppendInput struct {
	Source     string
	Content    string
	CustomerID string
	Category   string
	Sentiment  string
	IdeaID     string
}

func (s *Service) Append(ctx context.Context, productID domain.ProductID, in AppendInput) (*domain.ProductFeedback, error) {
	if s.Feedback == nil {
		return nil, domain.ErrNotConfigured
	}
	src := strings.TrimSpace(in.Source)
	if src == "" {
		return nil, fmt.Errorf("%w: source is required", domain.ErrInvalidInput)
	}
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, fmt.Errorf("%w: content is required", domain.ErrInvalidInput)
	}
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	var ideaID domain.IdeaID
	if strings.TrimSpace(in.IdeaID) != "" {
		ideaID = domain.IdeaID(strings.TrimSpace(in.IdeaID))
		if s.Ideas != nil {
			idea, err := s.Ideas.ByID(ctx, ideaID)
			if err != nil {
				return nil, err
			}
			if idea.ProductID != productID {
				return nil, fmt.Errorf("%w: idea belongs to another product", domain.ErrInvalidInput)
			}
		}
	}
	now := s.Clock.Now()
	f := &domain.ProductFeedback{
		ID:         s.IDs.NewProductFeedbackID(),
		ProductID:  productID,
		Source:     src,
		Content:    content,
		CustomerID: strings.TrimSpace(in.CustomerID),
		Category:   strings.TrimSpace(in.Category),
		Sentiment:  domain.NormalizeFeedbackSentiment(in.Sentiment),
		IdeaID:     ideaID,
		CreatedAt:  now,
	}
	if err := s.Feedback.Append(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Service) ListByProduct(ctx context.Context, productID domain.ProductID, limit int) ([]domain.ProductFeedback, error) {
	if s.Feedback == nil {
		return nil, domain.ErrNotConfigured
	}
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.Feedback.ListByProduct(ctx, productID, limit)
}

func (s *Service) SetProcessed(ctx context.Context, id string, processed bool) error {
	if s.Feedback == nil {
		return domain.ErrNotConfigured
	}
	return s.Feedback.SetProcessed(ctx, id, processed)
}

// ByID returns one feedback row.
func (s *Service) ByID(ctx context.Context, id string) (*domain.ProductFeedback, error) {
	if s.Feedback == nil {
		return nil, domain.ErrNotConfigured
	}
	return s.Feedback.ByID(ctx, id)
}
