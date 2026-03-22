package autopilot

import (
	"context"
	"fmt"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
)

// IdeaMetadataPatch updates MC-style idea fields (no swipe / status — use POST …/swipe).
type IdeaMetadataPatch struct {
	Category             *string
	ResearchBacking      *string
	ImpactScore          *float64
	FeasibilityScore     *float64
	Complexity           *string
	EstimatedEffortHours *float64
	CompetitiveAnalysis  *string
	TargetUserSegment    *string
	RevenuePotential     *string
	TechnicalApproach    *string
	Risks                *string
	Tags                 *[]string
	Source               *string
	SourceResearch       *string
	UserNotes            *string
	Reasoning            *string
	Description          *string
	Title                *string
}

// PatchIdeaMetadata applies non-swipe edits to an idea.
func (s *Service) PatchIdeaMetadata(ctx context.Context, ideaID domain.IdeaID, p IdeaMetadataPatch) error {
	idea, err := s.Ideas.ByID(ctx, ideaID)
	if err != nil {
		return err
	}
	if p.Title != nil {
		idea.Title = strings.TrimSpace(*p.Title)
		if idea.Title == "" {
			return fmt.Errorf("%w: title cannot be empty", domain.ErrInvalidInput)
		}
	}
	if p.Description != nil {
		idea.Description = strings.TrimSpace(*p.Description)
	}
	if p.Reasoning != nil {
		idea.Reasoning = strings.TrimSpace(*p.Reasoning)
	}
	if p.Category != nil {
		idea.Category = domain.NormalizeIdeaCategory(*p.Category)
	}
	if p.ResearchBacking != nil {
		idea.ResearchBacking = strings.TrimSpace(*p.ResearchBacking)
	}
	if p.ImpactScore != nil {
		idea.ImpactScore = *p.ImpactScore
		idea.Impact = *p.ImpactScore
	}
	if p.FeasibilityScore != nil {
		idea.FeasibilityScore = *p.FeasibilityScore
		idea.Feasibility = *p.FeasibilityScore
	}
	if p.Complexity != nil {
		c := strings.TrimSpace(*p.Complexity)
		if c != "" && domain.NormalizeIdeaComplexity(c) == "" {
			return fmt.Errorf("%w: complexity must be S, M, L, or XL", domain.ErrInvalidInput)
		}
		idea.Complexity = domain.NormalizeIdeaComplexity(c)
	}
	if p.EstimatedEffortHours != nil {
		if *p.EstimatedEffortHours < 0 {
			return fmt.Errorf("%w: estimated_effort_hours must be >= 0", domain.ErrInvalidInput)
		}
		idea.EstimatedEffortHours = *p.EstimatedEffortHours
	}
	if p.CompetitiveAnalysis != nil {
		idea.CompetitiveAnalysis = strings.TrimSpace(*p.CompetitiveAnalysis)
	}
	if p.TargetUserSegment != nil {
		idea.TargetUserSegment = strings.TrimSpace(*p.TargetUserSegment)
	}
	if p.RevenuePotential != nil {
		idea.RevenuePotential = strings.TrimSpace(*p.RevenuePotential)
	}
	if p.TechnicalApproach != nil {
		idea.TechnicalApproach = strings.TrimSpace(*p.TechnicalApproach)
	}
	if p.Risks != nil {
		idea.Risks = strings.TrimSpace(*p.Risks)
	}
	if p.Tags != nil {
		idea.Tags = append([]string(nil), (*p.Tags)...)
	}
	if p.Source != nil {
		idea.Source = domain.NormalizeIdeaSource(*p.Source)
	}
	if p.SourceResearch != nil {
		idea.SourceResearch = strings.TrimSpace(*p.SourceResearch)
	}
	if p.UserNotes != nil {
		idea.UserNotes = strings.TrimSpace(*p.UserNotes)
	}
	idea.UpdatedAt = s.Clock.Now()
	return s.Ideas.Save(ctx, idea)
}
