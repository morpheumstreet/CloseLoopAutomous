package domain

import (
	"encoding/json"
	"strings"
)

// MergeShipState is persisted on workspace_merge_queue after a ship attempt.
type MergeShipState string

const (
	MergeShipNone     MergeShipState = ""          // never attempted / legacy row
	MergeShipMerged   MergeShipState = "merged"    // Git or local merge succeeded
	MergeShipSkipped  MergeShipState = "skipped"   // break-glass: advance queue without forge merge
	MergeShipConflict MergeShipState = "conflict"  // merge conflict (local or GitHub)
	MergeShipFailed   MergeShipState = "failed"    // other failure
)

// MergeShipResult is the outcome of a forge or local git merge attempt.
type MergeShipResult struct {
	State         MergeShipState
	MergedSHA     string
	ErrorMessage  string
	ConflictFiles []string
}

// MergePolicy is optional per-product JSON (merge_policy_json).
type MergePolicy struct {
	MergeMethod string `json:"merge_method"` // merge | squash | rebase (default merge)
	// MergeBackendOverride: when set, overrides process ARMS_MERGE_BACKEND for this product (github|local|noop).
	MergeBackendOverride string `json:"merge_backend,omitempty"`
}

// ParseMergePolicy unmarshals product.MergePolicyJSON; invalid JSON yields defaults.
func ParseMergePolicy(jsonStr string) MergePolicy {
	s := strings.TrimSpace(jsonStr)
	if s == "" {
		return MergePolicy{MergeMethod: "merge"}
	}
	var p MergePolicy
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return MergePolicy{MergeMethod: "merge"}
	}
	if strings.TrimSpace(p.MergeMethod) == "" {
		p.MergeMethod = "merge"
	}
	p.MergeMethod = strings.ToLower(strings.TrimSpace(p.MergeMethod))
	return p
}

// NormalizeMergeMethod returns a GitHub API merge_method string.
func NormalizeMergeMethod(m string) string {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case "squash":
		return "squash"
	case "rebase":
		return "rebase"
	default:
		return "merge"
	}
}
