package convoy

import (
	"encoding/json"
	"strings"
	"time"
)

const mcCompatKey = "mc_compat"

// MCCompatFields is optional Mission Control–oriented metadata merged into convoy.metadata_json under mc_compat.
type MCCompatFields struct {
	Name              string
	Strategy          string
	Status            string
	DecompositionSpec string
	UpdatedAt         time.Time
}

// MergeMCCompatIntoMetadata merges patch into the mc_compat object; other top-level metadata keys are preserved.
func MergeMCCompatIntoMetadata(existingJSON string, patch MCCompatFields) (string, error) {
	s := strings.TrimSpace(existingJSON)
	if s == "" {
		s = "{}"
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &root); err != nil {
		return "", err
	}
	var compat map[string]any
	if raw, ok := root[mcCompatKey]; ok && len(raw) > 0 && string(raw) != "null" {
		_ = json.Unmarshal(raw, &compat)
	}
	if compat == nil {
		compat = make(map[string]any)
	}
	if patch.Name != "" {
		compat["name"] = patch.Name
	}
	if patch.Strategy != "" {
		compat["strategy"] = patch.Strategy
	}
	if patch.Status != "" {
		compat["status"] = patch.Status
	}
	if patch.DecompositionSpec != "" {
		compat["decomposition_spec"] = patch.DecompositionSpec
	}
	at := patch.UpdatedAt
	if at.IsZero() {
		at = time.Now().UTC()
	}
	compat["updated_at"] = at.UTC().Format(time.RFC3339Nano)
	b, err := json.Marshal(compat)
	if err != nil {
		return "", err
	}
	root[mcCompatKey] = b
	out, err := json.Marshal(root)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// MCCompatFromMetadata returns mc_compat fields if present (best-effort).
func MCCompatFromMetadata(metadataJSON string) (name, strategy, status, decompositionSpec, updatedAt string) {
	s := strings.TrimSpace(metadataJSON)
	if s == "" {
		return "", "", "", "", ""
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &root); err != nil {
		return "", "", "", "", ""
	}
	raw, ok := root[mcCompatKey]
	if !ok || len(raw) == 0 {
		return "", "", "", "", ""
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", "", "", "", ""
	}
	if v, ok := m["name"].(string); ok {
		name = v
	}
	if v, ok := m["strategy"].(string); ok {
		strategy = v
	}
	if v, ok := m["status"].(string); ok {
		status = v
	}
	if v, ok := m["decomposition_spec"].(string); ok {
		decompositionSpec = v
	}
	if v, ok := m["updated_at"].(string); ok {
		updatedAt = v
	}
	return name, strategy, status, decompositionSpec, updatedAt
}

// MCConvoyDispatchAllowed is false when mc_compat.status is a non-active pause-like value.
func MCConvoyDispatchAllowed(metadataJSON string) bool {
	_, _, st, _, _ := MCCompatFromMetadata(metadataJSON)
	switch strings.ToLower(strings.TrimSpace(st)) {
	case "", "active":
		return true
	case "paused", "cancelled", "canceled", "done", "failed":
		return false
	default:
		return true
	}
}
