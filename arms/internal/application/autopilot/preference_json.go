package autopilot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

func appendSwipePreferenceJSON(raw string, ideaID domain.IdeaID, dec domain.SwipeDecision, at time.Time) (string, error) {
	var events []map[string]any
	raw = strings.TrimSpace(raw)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &events); err != nil {
			return raw, fmt.Errorf("%w: preference_model_json must be a JSON array", domain.ErrInvalidInput)
		}
	}
	if events == nil {
		events = []map[string]any{}
	}
	events = append(events, map[string]any{
		"idea_id":  string(ideaID),
		"decision": domain.SwipeDecisionKey(dec),
		"at":       at.Format(time.RFC3339Nano),
	})
	b, err := json.Marshal(events)
	if err != nil {
		return raw, err
	}
	return string(b), nil
}
