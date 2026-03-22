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

// AllowedIdeaCategory values align with Mission Control CHECK constraint.
var AllowedIdeaCategory = map[string]struct{}{
	"feature": {}, "improvement": {}, "ux": {}, "performance": {}, "integration": {},
	"infrastructure": {}, "content": {}, "growth": {}, "monetization": {}, "operations": {}, "security": {},
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
