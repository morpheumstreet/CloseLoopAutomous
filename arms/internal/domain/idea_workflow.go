package domain

import (
	"strings"
	"time"
)

// MC-aligned idea lifecycle (mission-control ideas.status).
const (
	IdeaStatusPending  = "pending"
	IdeaStatusApproved = "approved"
	IdeaStatusRejected = "rejected"
	IdeaStatusMaybe    = "maybe"
	IdeaStatusBuilding = "building"
	IdeaStatusBuilt    = "built"
	IdeaStatusShipped  = "shipped"
)

// AllowedIdeaCategory: legacy MC product buckets plus ideation SOP buckets (Fishtank single-select).
var AllowedIdeaCategory = map[string]struct{}{
	// Legacy MC (existing rows / API)
	"feature": {}, "improvement": {}, "ux": {}, "performance": {}, "integration": {},
	"infrastructure": {}, "content": {}, "growth": {}, "monetization": {}, "operations": {}, "security": {},
	// Ideation buckets (slug → stored in ideas.category)
	"twitter_x_line": {}, "tiktok_reels_shorts": {}, "youtube_video": {}, "short_film": {},
	"documentation_video": {}, "interview_video": {}, "podcast_episode": {}, "newsletter_series": {},
	"research_discovery": {}, "investment_idea": {}, "prediction_forecasting": {}, "gamble_betting_systems": {},
	"gamble": {}, "casino": {},
	"engineered_software": {}, "nocode_lowcode_product": {},
	"blockchain_protocol": {}, "meme_coin": {}, "blockchain_smart_contract": {}, "crosschain": {},
	"iot_device": {}, "electronic_hardware": {},
	"robot_product": {}, "medical_device": {}, "drugs_pharma": {}, "biotech_gene_aerospace_regulated": {},
}

// AllowedIdeaComplexity: T-shirt sizes.
var AllowedIdeaComplexity = map[string]struct{}{
	"S": {}, "M": {}, "L": {}, "XL": {},
}

// AllowedIdeaSource aligns with MC ideas.source.
var AllowedIdeaSource = map[string]struct{}{
	"research": {}, "manual": {}, "resurfaced": {}, "feedback": {},
}

// NormalizeIdeaCategory returns a valid category or "feature".
func NormalizeIdeaCategory(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "feature"
	}
	if _, ok := AllowedIdeaCategory[strings.ToLower(s)]; ok {
		return strings.ToLower(s)
	}
	return "feature"
}

// NormalizeIdeaComplexity returns upper-case S/M/L/XL or empty when unknown/empty.
func NormalizeIdeaComplexity(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return ""
	}
	if _, ok := AllowedIdeaComplexity[s]; ok {
		return s
	}
	return ""
}

// NormalizeIdeaSource returns a valid source or "research".
func NormalizeIdeaSource(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "research"
	}
	if _, ok := AllowedIdeaSource[s]; ok {
		return s
	}
	return "research"
}

// SyncIdeaStatusFromSwipe sets Status, SwipedAt, UpdatedAt from a swipe decision (Decided/Decision must already match).
func SyncIdeaStatusFromSwipe(i *Idea, d SwipeDecision, now time.Time) {
	switch d {
	case DecisionPass:
		i.Status = IdeaStatusRejected
	case DecisionMaybe:
		i.Status = IdeaStatusMaybe
	case DecisionYes, DecisionNow:
		i.Status = IdeaStatusApproved
	default:
		i.Status = IdeaStatusPending
	}
	i.SwipedAt = now.UTC()
	i.UpdatedAt = now.UTC()
}

// NormalizeIdeaForSave fills defaults and keeps score columns aligned with Impact/Feasibility.
func NormalizeIdeaForSave(i *Idea, now time.Time) {
	now = now.UTC()
	i.Category = NormalizeIdeaCategory(i.Category)
	i.Complexity = NormalizeIdeaComplexity(i.Complexity)
	i.Source = NormalizeIdeaSource(i.Source)
	if i.Status == "" {
		if !i.Decided {
			i.Status = IdeaStatusPending
		} else {
			switch i.Decision {
			case DecisionPass:
				i.Status = IdeaStatusRejected
			case DecisionMaybe:
				i.Status = IdeaStatusMaybe
			case DecisionYes, DecisionNow:
				i.Status = IdeaStatusApproved
			default:
				i.Status = IdeaStatusPending
			}
		}
	}
	if i.ImpactScore == 0 && i.Impact != 0 {
		i.ImpactScore = i.Impact
	}
	if i.FeasibilityScore == 0 && i.Feasibility != 0 {
		i.FeasibilityScore = i.Feasibility
	}
	if i.Impact == 0 && i.ImpactScore != 0 {
		i.Impact = i.ImpactScore
	}
	if i.Feasibility == 0 && i.FeasibilityScore != 0 {
		i.Feasibility = i.FeasibilityScore
	}
	if i.UpdatedAt.IsZero() {
		i.UpdatedAt = i.CreatedAt.UTC()
		if i.UpdatedAt.IsZero() {
			i.UpdatedAt = now
		}
	}
	if i.Tags == nil {
		i.Tags = []string{}
	}
}
