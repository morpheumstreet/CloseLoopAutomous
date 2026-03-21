package product

import (
	"context"
	"fmt"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service registers products at the start of the pipeline.
type Service struct {
	Products ports.ProductRepository
	Clock    ports.Clock
	IDs      ports.IdentityGenerator
}

// RegistrationInput creates a product; optional strings extend the MC-style profile.
type RegistrationInput struct {
	Name            string
	WorkspaceID     string
	RepoURL         string
	RepoClonePath   string
	RepoBranch      string
	Description     string
	ProgramDocument string
	SettingsJSON    string
	IconURL         string

	ResearchCadenceSec  *int
	IdeationCadenceSec  *int
	AutomationTier      string // empty → supervised
	AutoDispatchEnabled *bool
}

func (s *Service) Register(ctx context.Context, in RegistrationInput) (*domain.Product, error) {
	tier, err := domain.ParseAutomationTier(in.AutomationTier)
	if err != nil {
		return nil, err
	}
	if in.ResearchCadenceSec != nil && *in.ResearchCadenceSec < 0 {
		return nil, fmt.Errorf("%w: research_cadence_sec must be >= 0", domain.ErrInvalidInput)
	}
	if in.IdeationCadenceSec != nil && *in.IdeationCadenceSec < 0 {
		return nil, fmt.Errorf("%w: ideation_cadence_sec must be >= 0", domain.ErrInvalidInput)
	}
	now := s.Clock.Now()
	p := &domain.Product{
		ID:                  s.IDs.NewProductID(),
		Name:                in.Name,
		Stage:               domain.StageResearch,
		WorkspaceID:         in.WorkspaceID,
		RepoURL:             strings.TrimSpace(in.RepoURL),
		RepoClonePath:       strings.TrimSpace(in.RepoClonePath),
		RepoBranch:          strings.TrimSpace(in.RepoBranch),
		Description:         strings.TrimSpace(in.Description),
		ProgramDocument:     strings.TrimSpace(in.ProgramDocument),
		SettingsJSON:        in.SettingsJSON,
		IconURL:             strings.TrimSpace(in.IconURL),
		AutomationTier:      tier,
		UpdatedAt:           now,
	}
	if in.ResearchCadenceSec != nil {
		p.ResearchCadenceSec = *in.ResearchCadenceSec
	}
	if in.IdeationCadenceSec != nil {
		p.IdeationCadenceSec = *in.IdeationCadenceSec
	}
	if in.AutoDispatchEnabled != nil {
		p.AutoDispatchEnabled = *in.AutoDispatchEnabled
	}
	if err := s.Products.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// MetadataPatch updates product profile fields; only non-nil pointers are applied.
type MetadataPatch struct {
	Name            *string
	RepoURL         *string
	RepoClonePath   *string
	RepoBranch      *string
	Description     *string
	ProgramDocument *string
	SettingsJSON    *string
	IconURL         *string

	ResearchCadenceSec  *int
	IdeationCadenceSec  *int
	AutomationTier      *string
	AutoDispatchEnabled *bool
}

// PatchMetadata applies partial updates to product metadata (not pipeline stage).
func (s *Service) PatchMetadata(ctx context.Context, id domain.ProductID, patch MetadataPatch) (*domain.Product, error) {
	p, err := s.Products.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if patch.Name != nil {
		v := strings.TrimSpace(*patch.Name)
		if v == "" {
			return nil, domain.ErrInvalidInput
		}
		p.Name = v
	}
	if patch.RepoURL != nil {
		p.RepoURL = strings.TrimSpace(*patch.RepoURL)
	}
	if patch.RepoClonePath != nil {
		p.RepoClonePath = strings.TrimSpace(*patch.RepoClonePath)
	}
	if patch.RepoBranch != nil {
		p.RepoBranch = strings.TrimSpace(*patch.RepoBranch)
	}
	if patch.Description != nil {
		p.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.ProgramDocument != nil {
		p.ProgramDocument = strings.TrimSpace(*patch.ProgramDocument)
	}
	if patch.SettingsJSON != nil {
		p.SettingsJSON = *patch.SettingsJSON
	}
	if patch.IconURL != nil {
		p.IconURL = strings.TrimSpace(*patch.IconURL)
	}
	if patch.ResearchCadenceSec != nil {
		if *patch.ResearchCadenceSec < 0 {
			return nil, fmt.Errorf("%w: research_cadence_sec must be >= 0", domain.ErrInvalidInput)
		}
		p.ResearchCadenceSec = *patch.ResearchCadenceSec
	}
	if patch.IdeationCadenceSec != nil {
		if *patch.IdeationCadenceSec < 0 {
			return nil, fmt.Errorf("%w: ideation_cadence_sec must be >= 0", domain.ErrInvalidInput)
		}
		p.IdeationCadenceSec = *patch.IdeationCadenceSec
	}
	if patch.AutomationTier != nil {
		tier, err := domain.ParseAutomationTier(*patch.AutomationTier)
		if err != nil {
			return nil, err
		}
		p.AutomationTier = tier
	}
	if patch.AutoDispatchEnabled != nil {
		p.AutoDispatchEnabled = *patch.AutoDispatchEnabled
	}
	p.UpdatedAt = s.Clock.Now()
	if err := s.Products.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
