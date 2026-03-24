package autopilot

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

const (
	preferenceAggregateVersion     = 1
	defaultPreferenceRecomputeLimit = 500
)

func swipeDecisionAffinityWeight(decision string) float64 {
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "yes", "now":
		return 2.0
	case "maybe":
		return 0.25
	case "pass":
		return -1.5
	default:
		return 0
	}
}

func feedbackSentimentAffinityWeight(sentiment string) float64 {
	switch domain.NormalizeFeedbackSentiment(sentiment) {
	case "positive":
		return 0.6
	case "negative":
		return -0.6
	case "mixed":
		return 0.15
	default:
		return 0
	}
}

func addCount(m map[string]int, key string, delta int) {
	if key == "" {
		return
	}
	m[key] += delta
}

func mergeAffinity(dst map[string]float64, key string, w float64) {
	k := strings.TrimSpace(key)
	if k == "" {
		return
	}
	dst[k] += w
}

// recomputePreferenceModel aggregates swipe_history, product_feedback, and maybe-pool stats into preference_models.
func (s *Service) recomputePreferenceModel(ctx context.Context, productID domain.ProductID, limit int) (string, error) {
	if s.PrefModel == nil {
		return "", fmt.Errorf("%w: preference_models store not configured", domain.ErrInvalidInput)
	}
	if s.Swipes == nil && s.Feedback == nil && s.MaybePool == nil {
		return "", fmt.Errorf("%w: no preference data sources (swipes, feedback, maybe pool)", domain.ErrInvalidInput)
	}
	if _, err := s.Products.ByID(ctx, productID); err != nil {
		return "", err
	}
	if limit <= 0 || limit > 5000 {
		limit = defaultPreferenceRecomputeLimit
	}

	decisionCounts := make(map[string]int)
	categoryAffinity := make(map[string]float64)
	var swipeSample int

	if s.Swipes != nil {
		swipes, err := s.Swipes.ListByProduct(ctx, productID, limit)
		if err != nil {
			return "", err
		}
		swipeSample = len(swipes)
		for i := range swipes {
			d := swipes[i].Decision
			addCount(decisionCounts, d, 1)
			w := swipeDecisionAffinityWeight(d)
			if w == 0 {
				continue
			}
			idea, err := s.Ideas.ByID(ctx, swipes[i].IdeaID)
			if err != nil || idea == nil {
				continue
			}
			cat := domain.NormalizeIdeaCategory(idea.Category)
			mergeAffinity(categoryAffinity, cat, w)
		}
	}

	sentimentCounts := make(map[string]int)
	feedbackCategoryCounts := make(map[string]int)
	var feedbackSample int

	if s.Feedback != nil {
		fb, err := s.Feedback.ListByProduct(ctx, productID, limit)
		if err != nil {
			return "", err
		}
		feedbackSample = len(fb)
		for i := range fb {
			addCount(sentimentCounts, domain.NormalizeFeedbackSentiment(fb[i].Sentiment), 1)
			fcat := domain.NormalizeIdeaCategory(fb[i].Category)
			addCount(feedbackCategoryCounts, fcat, 1)
			mergeAffinity(categoryAffinity, fcat, feedbackSentimentAffinityWeight(fb[i].Sentiment))
		}
	}

	maybeSection := map[string]any{}
	if s.MaybePool != nil {
		entries, err := s.MaybePool.ListEntriesByProduct(ctx, productID)
		if err != nil {
			return "", err
		}
		var totalEval int
		var geOne int
		for i := range entries {
			totalEval += entries[i].EvaluationCount
			if entries[i].EvaluationCount >= 1 {
				geOne++
			}
		}
		survival := 0.0
		if len(entries) > 0 {
			survival = float64(geOne) / float64(len(entries))
		}
		maybeSection = map[string]any{
			"current_size":       len(entries),
			"total_evaluations":  totalEval,
			"entries_reeval_once": geOne,
			"reeval_survival_rate": math.Round(survival*1000) / 1000,
		}
	}

	now := s.Clock.Now().UTC()
	payload := map[string]any{
		"version":        preferenceAggregateVersion,
		"source":         "preference_aggregate_v1",
		"generated_at":   now.Format(time.RFC3339Nano),
		"category_affinity": categoryAffinity,
	}
	if swipeSample > 0 || len(decisionCounts) > 0 {
		payload["swipe"] = map[string]any{
			"decision_counts": decisionCounts,
			"sample_size":     swipeSample,
		}
	}
	if feedbackSample > 0 || len(sentimentCounts) > 0 {
		payload["feedback"] = map[string]any{
			"sentiment_counts": sentimentCounts,
			"category_counts":  feedbackCategoryCounts,
			"sample_size":      feedbackSample,
		}
	}
	if len(maybeSection) > 0 {
		payload["maybe_pool"] = maybeSection
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	jsonStr := string(b)
	if err := s.PrefModel.Upsert(ctx, productID, jsonStr, now); err != nil {
		return "", err
	}
	return jsonStr, nil
}

// preferenceHintsForIdeation turns stored model_json into a short prompt appendix for ideation.
func preferenceHintsForIdeation(modelJSON string) string {
	raw := strings.TrimSpace(modelJSON)
	if raw == "" || raw == "{}" {
		return ""
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return ""
	}
	var lines []string

	if v, ok := root["version"].(float64); ok && int(v) == preferenceAggregateVersion {
		if aff, ok := root["category_affinity"].(map[string]any); ok && len(aff) > 0 {
			type kv struct {
				k string
				v float64
			}
			var pairs []kv
			for k, val := range aff {
				f, _ := val.(float64)
				if math.Abs(f) < 0.05 {
					continue
				}
				pairs = append(pairs, kv{k: k, v: f})
			}
			sort.Slice(pairs, func(i, j int) bool {
				if pairs[i].v == pairs[j].v {
					return pairs[i].k < pairs[j].k
				}
				return math.Abs(pairs[i].v) > math.Abs(pairs[j].v)
			})
			if len(pairs) > 0 {
				lines = append(lines, "Idea categories to lean toward (relative affinity, higher = more valued by operators and feedback):")
				for i := 0; i < len(pairs) && i < 8; i++ {
					lines = append(lines, fmt.Sprintf("- %s: %.2f", pairs[i].k, pairs[i].v))
				}
			}
		}
		if mp, ok := root["maybe_pool"].(map[string]any); ok {
			if sz, ok := mp["current_size"].(float64); ok && int(sz) > 0 {
				lines = append(lines, fmt.Sprintf("Maybe pool: %d deferred ideas; re-eval survival rate %.3f (ideas that survived at least one batch re-check / pool size).",
					int(sz), toFloat(mp["reeval_survival_rate"])))
			}
		}
		if fb, ok := root["feedback"].(map[string]any); ok {
			if sc, ok := fb["sentiment_counts"].(map[string]any); ok && len(sc) > 0 {
				lines = append(lines, "Recent external feedback sentiment mix: "+compactMapCounts(sc))
			}
		}
	} else if src, _ := root["source"].(string); src == "swipe_history_aggregate" {
		if counts, ok := root["counts"].(map[string]any); ok {
			lines = append(lines, "Legacy swipe decision counts: "+compactMapCounts(counts))
		}
		if n, ok := root["sample_size"].(float64); ok {
			lines = append(lines, fmt.Sprintf("(from last %d swipe rows)", int(n)))
		}
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func toFloat(x any) float64 {
	f, _ := x.(float64)
	return f
}

func compactMapCounts(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		v := m[k]
		switch t := v.(type) {
		case float64:
			parts = append(parts, fmt.Sprintf("%s=%.0f", k, t))
		default:
			parts = append(parts, fmt.Sprintf("%s=%v", k, t))
		}
	}
	return strings.Join(parts, ", ")
}

func (s *Service) refreshLearnedPreferenceModel(ctx context.Context, productID domain.ProductID) {
	if s.PrefModel == nil {
		return
	}
	if s.Swipes == nil && s.Feedback == nil && s.MaybePool == nil {
		return
	}
	_, _ = s.recomputePreferenceModel(ctx, productID, defaultPreferenceRecomputeLimit)
}

// RefreshPreferenceModelBestEffort recomputes the learned preference payload after external events (HTTP hooks, swipes).
func (s *Service) RefreshPreferenceModelBestEffort(ctx context.Context, productID domain.ProductID) {
	s.refreshLearnedPreferenceModel(ctx, productID)
}
