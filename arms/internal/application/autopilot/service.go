package autopilot

import (
	"context"
	"fmt"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service runs research → ideation → swipe stages for a product.
type Service struct {
	Products   ports.ProductRepository
	Ideas      ports.IdeaRepository
	MaybePool  ports.MaybePoolRepository // optional; nil skips pool persistence
	Swipes     ports.SwipeHistoryRepository // optional; nil skips persisted swipe log
	Research   ports.ResearchPort
	Ideation   ports.IdeationPort
	Clock      ports.Clock
	Identities ports.IdentityGenerator
}

func (s *Service) RunResearch(ctx context.Context, productID domain.ProductID) error {
	p, err := s.Products.ByID(ctx, productID)
	if err != nil {
		return err
	}
	if p.Stage != domain.StageResearch {
		return fmt.Errorf("%w: product stage is %s", domain.ErrInvalidTransition, p.Stage.String())
	}
	summary, err := s.Research.RunResearch(ctx, *p)
	if err != nil {
		return err
	}
	p.ResearchSummary = summary
	p.Stage = domain.StageIdeation
	p.UpdatedAt = s.Clock.Now()
	return s.Products.Save(ctx, p)
}

// RunIdeation expects product at ideation stage; uses stored research summary from the product.
func (s *Service) RunIdeation(ctx context.Context, productID domain.ProductID) error {
	p, err := s.Products.ByID(ctx, productID)
	if err != nil {
		return err
	}
	if p.Stage != domain.StageIdeation {
		return fmt.Errorf("%w: product stage is %s", domain.ErrInvalidTransition, p.Stage.String())
	}
	drafts, err := s.Ideation.GenerateIdeas(ctx, *p, p.ResearchSummary)
	if err != nil {
		return err
	}
	now := s.Clock.Now()
	for _, d := range drafts {
		idea := domain.Idea{
			ID:          s.Identities.NewIdeaID(),
			ProductID:   productID,
			Title:       d.Title,
			Description: d.Description,
			Impact:      d.Impact,
			Feasibility: d.Feasibility,
			Reasoning:   d.Reasoning,
			CreatedAt:   now,
		}
		if err := s.Ideas.Save(ctx, &idea); err != nil {
			return err
		}
	}
	p.Stage = domain.StageSwipe
	p.UpdatedAt = now
	return s.Products.Save(ctx, p)
}

// SubmitSwipe records the human decision; approved ideas move the product toward planning.
// "Maybe" adds the idea to the maybe pool; other decisions remove it from the pool if present.
func (s *Service) SubmitSwipe(ctx context.Context, ideaID domain.IdeaID, decision domain.SwipeDecision) error {
	idea, err := s.Ideas.ByID(ctx, ideaID)
	if err != nil {
		return err
	}
	if idea.Decided {
		return domain.ErrConflict
	}
	p, err := s.Products.ByID(ctx, idea.ProductID)
	if err != nil {
		return err
	}
	now := s.Clock.Now()
	pref, err := appendSwipePreferenceJSON(p.PreferenceModelJSON, ideaID, decision, now)
	if err != nil {
		return err
	}
	p.PreferenceModelJSON = pref

	idea.Decided = true
	idea.Decision = decision
	if err := s.Ideas.Save(ctx, idea); err != nil {
		return err
	}
	if s.MaybePool != nil {
		if decision == domain.DecisionMaybe {
			if err := s.MaybePool.Add(ctx, ideaID, idea.ProductID, now); err != nil {
				return err
			}
		} else {
			if err := s.MaybePool.Remove(ctx, ideaID); err != nil {
				return err
			}
		}
	}
	if decision.Approved() && p.Stage == domain.StageSwipe {
		p.Stage = domain.StagePlanning
	}
	p.UpdatedAt = now
	if err := s.Products.Save(ctx, p); err != nil {
		return err
	}
	if s.Swipes != nil {
		key := domain.SwipeDecisionKey(decision)
		if key == "" {
			return fmt.Errorf("%w: unknown swipe decision", domain.ErrInvalidInput)
		}
		if err := s.Swipes.Append(ctx, ideaID, idea.ProductID, key, now); err != nil {
			return err
		}
	}
	return nil
}

// ListSwipeHistory returns newest-first swipe audit rows for a product.
func (s *Service) ListSwipeHistory(ctx context.Context, productID domain.ProductID, limit int) ([]domain.SwipeHistoryEntry, error) {
	if s.Swipes == nil {
		return nil, nil
	}
	return s.Swipes.ListByProduct(ctx, productID, limit)
}

// PromoteMaybe turns a "maybe" swipe into an approval (yes) and advances the product when still in swipe.
func (s *Service) PromoteMaybe(ctx context.Context, ideaID domain.IdeaID) error {
	idea, err := s.Ideas.ByID(ctx, ideaID)
	if err != nil {
		return err
	}
	if !idea.Decided || idea.Decision != domain.DecisionMaybe {
		return fmt.Errorf("%w: idea must be decided as maybe", domain.ErrInvalidInput)
	}
	p, err := s.Products.ByID(ctx, idea.ProductID)
	if err != nil {
		return err
	}
	now := s.Clock.Now()
	idea.Decision = domain.DecisionYes
	if err := s.Ideas.Save(ctx, idea); err != nil {
		return err
	}
	if s.MaybePool != nil {
		if err := s.MaybePool.Remove(ctx, ideaID); err != nil {
			return err
		}
	}
	if s.Swipes != nil {
		if err := s.Swipes.Append(ctx, ideaID, idea.ProductID, domain.SwipeDecisionKey(domain.DecisionYes), now); err != nil {
			return err
		}
	}
	if p.Stage == domain.StageSwipe {
		p.Stage = domain.StagePlanning
		p.UpdatedAt = now
		return s.Products.Save(ctx, p)
	}
	return nil
}

// TickScheduled runs due research/ideation steps from cadence fields (best-effort; errors are skipped per product).
func (s *Service) TickScheduled(ctx context.Context, now time.Time) error {
	list, err := s.Products.ListAll(ctx)
	if err != nil {
		return err
	}
	for i := range list {
		p := &list[i]
		if p.ResearchCadenceSec <= 0 || p.Stage != domain.StageResearch {
			continue
		}
		if !cadenceDue(p.LastAutoResearchAt, now, p.ResearchCadenceSec) {
			continue
		}
		if err := s.RunResearch(ctx, p.ID); err != nil {
			continue
		}
		p2, err := s.Products.ByID(ctx, p.ID)
		if err != nil {
			continue
		}
		p2.LastAutoResearchAt = now
		_ = s.Products.Save(ctx, p2)
	}
	list, err = s.Products.ListAll(ctx)
	if err != nil {
		return err
	}
	for i := range list {
		p := &list[i]
		if p.IdeationCadenceSec <= 0 || p.Stage != domain.StageIdeation {
			continue
		}
		if !cadenceDue(p.LastAutoIdeationAt, now, p.IdeationCadenceSec) {
			continue
		}
		if err := s.RunIdeation(ctx, p.ID); err != nil {
			continue
		}
		p2, err := s.Products.ByID(ctx, p.ID)
		if err != nil {
			continue
		}
		p2.LastAutoIdeationAt = now
		_ = s.Products.Save(ctx, p2)
	}
	return nil
}

func cadenceDue(last time.Time, now time.Time, sec int) bool {
	if sec <= 0 {
		return false
	}
	if last.IsZero() {
		return true
	}
	return now.Sub(last) >= time.Duration(sec)*time.Second
}
