package autopilot

import (
	"context"
	"fmt"
	"strings"
	"time"

	cronlib "github.com/robfig/cron/v3"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service runs research → ideation → swipe stages for a product.
type Service struct {
	Products       ports.ProductRepository
	Ideas          ports.IdeaRepository
	MaybePool      ports.MaybePoolRepository // optional; nil skips pool persistence
	Swipes         ports.SwipeHistoryRepository // optional; nil skips persisted swipe log
	Feedback       ports.ProductFeedbackRepository // optional; enriches preference aggregate when set
	ResearchCycles ports.ResearchCycleRepository // optional; append-only history on successful research
	Schedules      ports.ProductScheduleRepository // optional; nil = all products eligible for cadence ticks
	PrefModel      ports.PreferenceModelRepository // optional; used by preference recomputation
	Research       ports.ResearchPort
	Ideation       ports.IdeationPort
	Clock          ports.Clock
	Identities     ports.IdentityGenerator
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
	if err := s.Products.Save(ctx, p); err != nil {
		return err
	}
	if s.ResearchCycles != nil {
		_ = s.ResearchCycles.Append(ctx, s.Identities.NewResearchCycleID(), productID, summary, s.Clock.Now())
	}
	return nil
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
	researchIn := strings.TrimSpace(p.ResearchSummary)
	if s.PrefModel != nil {
		if mj, _, ok, err := s.PrefModel.Get(ctx, productID); err == nil && ok {
			if hints := preferenceHintsForIdeation(mj); hints != "" {
				if researchIn == "" {
					researchIn = hints
				} else {
					researchIn = researchIn + "\n\n## Operator-derived preference signals (bias new ideas toward these)\n" + hints
				}
			}
		}
	}
	drafts, err := s.Ideation.GenerateIdeas(ctx, *p, researchIn)
	if err != nil {
		return err
	}
	now := s.Clock.Now()
	var cycleID string
	if s.ResearchCycles != nil {
		if hist, herr := s.ResearchCycles.ListByProduct(ctx, productID, 1); herr == nil && len(hist) > 0 {
			cycleID = hist[0].ID
		}
	}
	for _, d := range drafts {
		rb := strings.TrimSpace(d.ResearchBacking)
		if rb == "" {
			rb = d.Reasoning
		}
		tags := append([]string(nil), d.Tags...)
		idea := domain.Idea{
			ID:                   s.Identities.NewIdeaID(),
			ProductID:            productID,
			Title:                d.Title,
			Description:          d.Description,
			Impact:               d.Impact,
			Feasibility:          d.Feasibility,
			Reasoning:            d.Reasoning,
			CreatedAt:            now,
			UpdatedAt:            now,
			ResearchCycleID:      cycleID,
			Category:             domain.NormalizeIdeaCategory(d.Category),
			ResearchBacking:      rb,
			Complexity:           domain.NormalizeIdeaComplexity(d.Complexity),
			EstimatedEffortHours: d.EstimatedEffortHours,
			CompetitiveAnalysis:  d.CompetitiveAnalysis,
			TargetUserSegment:    d.TargetUserSegment,
			RevenuePotential:     d.RevenuePotential,
			TechnicalApproach:    d.TechnicalApproach,
			Risks:                d.Risks,
			Tags:                 tags,
			Source:               domain.NormalizeIdeaSource("research"),
			SourceResearch:       d.SourceResearch,
			Status:               domain.IdeaStatusPending,
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
	domain.SyncIdeaStatusFromSwipe(idea, decision, now)
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
	s.refreshLearnedPreferenceModel(ctx, idea.ProductID)
	return nil
}

// ListSwipeHistory returns newest-first swipe audit rows for a product.
func (s *Service) ListSwipeHistory(ctx context.Context, productID domain.ProductID, limit int) ([]domain.SwipeHistoryEntry, error) {
	if s.Swipes == nil {
		return nil, nil
	}
	return s.Swipes.ListByProduct(ctx, productID, limit)
}

// ListResearchHistory returns newest-first research cycle snapshots (404 if product missing).
func (s *Service) ListResearchHistory(ctx context.Context, productID domain.ProductID, limit int) ([]domain.ResearchCycle, error) {
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	if s.ResearchCycles == nil {
		return []domain.ResearchCycle{}, nil
	}
	return s.ResearchCycles.ListByProduct(ctx, productID, limit)
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
	idea.Status = domain.IdeaStatusApproved
	idea.UpdatedAt = now
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
	s.refreshLearnedPreferenceModel(ctx, idea.ProductID)
	if p.Stage == domain.StageSwipe {
		p.Stage = domain.StagePlanning
		p.UpdatedAt = now
		return s.Products.Save(ctx, p)
	}
	return nil
}

// BatchReevaluateMaybePool records a batch re-evaluation for every idea in the maybe pool (MC-style).
// nextEvaluateDelaySec: nil or 0 → next_evaluate_at = now (due immediately); positive → now + seconds; -1 → clear next_evaluate_at.
func (s *Service) BatchReevaluateMaybePool(ctx context.Context, productID domain.ProductID, note string, nextEvaluateDelaySec *int) error {
	if s.MaybePool == nil {
		return domain.ErrNotConfigured
	}
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return err
	}
	now := s.Clock.Now().UTC()
	var next time.Time
	if nextEvaluateDelaySec != nil {
		v := *nextEvaluateDelaySec
		switch {
		case v < -1:
			return fmt.Errorf("%w: next_evaluate_delay_sec must be >= -1", domain.ErrInvalidInput)
		case v == -1:
			next = time.Time{}
		case v == 0:
			next = now
		default:
			next = now.Add(time.Duration(v) * time.Second)
		}
	} else {
		next = now
	}
	return s.MaybePool.ApplyBatchReevaluate(ctx, productID, note, next, now)
}

// GetProductSchedule returns the schedule row or defaults (enabled, empty spec) when no row exists.
func (s *Service) GetProductSchedule(ctx context.Context, productID domain.ProductID) (*domain.ProductSchedule, error) {
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return nil, err
	}
	if s.Schedules == nil {
		return &domain.ProductSchedule{ProductID: productID, Enabled: true, SpecJSON: "{}"}, nil
	}
	row, err := s.Schedules.Get(ctx, productID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &domain.ProductSchedule{ProductID: productID, Enabled: true, SpecJSON: "{}"}, nil
	}
	return row, nil
}

// SaveProductSchedule upserts a full schedule row (503 when store not wired).
func (s *Service) SaveProductSchedule(ctx context.Context, row *domain.ProductSchedule) (*domain.ProductSchedule, error) {
	if s.Schedules == nil {
		return nil, domain.ErrNotConfigured
	}
	if _, err := s.Products.ByID(ctx, row.ProductID); err != nil {
		return nil, err
	}
	if row.SpecJSON == "" {
		row.SpecJSON = "{}"
	}
	cexpr := strings.TrimSpace(row.CronExpr)
	if cexpr != "" {
		if _, err := cronlib.ParseStandard(cexpr); err != nil {
			return nil, fmt.Errorf("%w: invalid cron_expr", domain.ErrInvalidInput)
		}
		row.CronExpr = cexpr
	}
	if row.DelaySeconds < 0 {
		return nil, fmt.Errorf("%w: delay_seconds must be >= 0", domain.ErrInvalidInput)
	}
	row.UpdatedAt = s.Clock.Now()
	if err := s.Schedules.Upsert(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) scheduleAllowsAutopilot(ctx context.Context, pid domain.ProductID) bool {
	if s.Schedules == nil {
		return true
	}
	row, err := s.Schedules.Get(ctx, pid)
	if err != nil || row == nil {
		return true
	}
	return row.Enabled
}

// RecomputePreferenceModelFromSwipes aggregates swipe_history, product_feedback, and maybe-pool stats into preference_models (heuristic weights; not ML).
func (s *Service) RecomputePreferenceModelFromSwipes(ctx context.Context, productID domain.ProductID, limit int) (string, error) {
	return s.recomputePreferenceModel(ctx, productID, limit)
}

// TickScheduled runs due research/ideation steps for every product (best-effort; errors are skipped per product).
func (s *Service) TickScheduled(ctx context.Context, now time.Time) error {
	list, err := s.Products.ListAll(ctx)
	if err != nil {
		return err
	}
	for i := range list {
		_ = s.TickProduct(ctx, list[i].ID, now)
	}
	return nil
}

// TickProduct runs at most one due research step and then at most one due ideation step for a single product.
// Errors from RunResearch/RunIdeation are swallowed (same best-effort behavior as TickScheduled).
func (s *Service) TickProduct(ctx context.Context, productID domain.ProductID, now time.Time) error {
	p, err := s.Products.ByID(ctx, productID)
	if err != nil {
		return err
	}
	if !s.scheduleAllowsAutopilot(ctx, productID) {
		return nil
	}
	if p.ResearchCadenceSec > 0 && p.Stage == domain.StageResearch && cadenceDue(p.LastAutoResearchAt, now, p.ResearchCadenceSec) {
		if err := s.RunResearch(ctx, productID); err == nil {
			p2, err := s.Products.ByID(ctx, productID)
			if err == nil {
				p2.LastAutoResearchAt = now
				_ = s.Products.Save(ctx, p2)
			}
		}
	}
	p, err = s.Products.ByID(ctx, productID)
	if err != nil {
		return err
	}
	if !s.scheduleAllowsAutopilot(ctx, productID) {
		return nil
	}
	if p.IdeationCadenceSec > 0 && p.Stage == domain.StageIdeation && cadenceDue(p.LastAutoIdeationAt, now, p.IdeationCadenceSec) {
		if err := s.RunIdeation(ctx, productID); err == nil {
			p2, err := s.Products.ByID(ctx, productID)
			if err == nil {
				p2.LastAutoIdeationAt = now
				_ = s.Products.Save(ctx, p2)
			}
		}
	}
	return nil
}

// NextAutopilotEnqueueDelay is used by the Asynq per-product chain: how long to wait before the next cadence check,
// and whether this product should stay on the chain (research/ideation with positive cadence and schedule enabled).
func (s *Service) NextAutopilotEnqueueDelay(ctx context.Context, productID domain.ProductID, now time.Time) (delay time.Duration, keep bool, err error) {
	if !s.scheduleAllowsAutopilot(ctx, productID) {
		return 0, false, nil
	}
	p, err := s.Products.ByID(ctx, productID)
	if err != nil {
		return 0, false, err
	}
	switch p.Stage {
	case domain.StageResearch:
		if p.ResearchCadenceSec <= 0 {
			return 0, false, nil
		}
		return delayUntilCadenceFire(p.LastAutoResearchAt, now, p.ResearchCadenceSec), true, nil
	case domain.StageIdeation:
		if p.IdeationCadenceSec <= 0 {
			return 0, false, nil
		}
		return delayUntilCadenceFire(p.LastAutoIdeationAt, now, p.IdeationCadenceSec), true, nil
	default:
		return 0, false, nil
	}
}

func delayUntilCadenceFire(last time.Time, now time.Time, sec int) time.Duration {
	if sec <= 0 {
		return 0
	}
	if last.IsZero() {
		return 0
	}
	due := last.Add(time.Duration(sec) * time.Second)
	if !now.Before(due) {
		return 0
	}
	return due.Sub(now)
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
