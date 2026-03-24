package knowledge

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

const maxAutoIngestRunes = 8000

func ingestEnabled(s *Service) bool {
	return s != nil && s.AutoIngest && s.Repo != nil
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// IngestFromSwipe records a swipe outcome as product knowledge (best-effort; errors are ignored by callers).
func (s *Service) IngestFromSwipe(ctx context.Context, idea *domain.Idea, decision domain.SwipeDecision) error {
	if !ingestEnabled(s) || idea == nil {
		return nil
	}
	key := domain.SwipeDecisionKey(decision)
	title := strings.TrimSpace(idea.Title)
	if title == "" {
		title = "(untitled idea)"
	}
	desc := strings.TrimSpace(idea.Description)
	var b strings.Builder
	b.WriteString("Operator swipe: ")
	b.WriteString(key)
	b.WriteString(" on idea \"")
	b.WriteString(title)
	b.WriteString("\"")
	if desc != "" {
		b.WriteString("\n\n")
		b.WriteString(desc)
	}
	content := truncateRunes(strings.TrimSpace(b.String()), maxAutoIngestRunes)
	if content == "" {
		return nil
	}
	meta := map[string]any{
		"ingest":   "auto",
		"source":   "swipe",
		"idea_id":  string(idea.ID),
		"decision": key,
	}
	_, err := s.Create(ctx, idea.ProductID, content, idea.LinkedTaskID, meta)
	return err
}

// IngestFromProductFeedback records external feedback as knowledge (best-effort).
func (s *Service) IngestFromProductFeedback(ctx context.Context, f *domain.ProductFeedback) error {
	if !ingestEnabled(s) || f == nil {
		return nil
	}
	content := truncateRunes(strings.TrimSpace(f.Content), maxAutoIngestRunes)
	if content == "" {
		return nil
	}
	meta := map[string]any{
		"ingest":      "auto",
		"source":      "product_feedback",
		"feedback_id": f.ID,
		"fb_source":   f.Source,
	}
	if strings.TrimSpace(f.Category) != "" {
		meta["category"] = strings.TrimSpace(f.Category)
	}
	if strings.TrimSpace(f.Sentiment) != "" {
		meta["sentiment"] = f.Sentiment
	}
	if f.IdeaID != "" {
		meta["idea_id"] = string(f.IdeaID)
	}
	var tid domain.TaskID
	_, err := s.Create(ctx, f.ProductID, content, tid, meta)
	return err
}

// IngestFromTaskCompletion records task done as knowledge (best-effort). summary overrides auto text when non-empty.
func (s *Service) IngestFromTaskCompletion(ctx context.Context, t *domain.Task, source, summary string) error {
	if !ingestEnabled(s) || t == nil {
		return nil
	}
	summary = strings.TrimSpace(summary)
	var content string
	if summary != "" {
		content = truncateRunes(summary, maxAutoIngestRunes)
	} else {
		var b strings.Builder
		b.WriteString("Task ")
		b.WriteString(string(t.ID))
		b.WriteString(" completed (")
		b.WriteString(source)
		b.WriteString(").")
		spec := strings.TrimSpace(t.Spec)
		if spec != "" {
			b.WriteString("\n\nSpec excerpt:\n")
			b.WriteString(truncateRunes(spec, 2000))
		}
		if strings.TrimSpace(t.StatusReason) != "" && t.Status == domain.StatusFailed {
			b.WriteString("\n\nStatus reason: ")
			b.WriteString(strings.TrimSpace(t.StatusReason))
		}
		content = truncateRunes(strings.TrimSpace(b.String()), maxAutoIngestRunes)
	}
	if content == "" {
		return nil
	}
	meta := map[string]any{
		"ingest":   "auto",
		"source":   "task_completion",
		"task_id":  string(t.ID),
		"done_via": source,
	}
	if t.IdeaID != "" {
		meta["idea_id"] = string(t.IdeaID)
	}
	_, err := s.Create(ctx, t.ProductID, content, t.ID, meta)
	return err
}

// IngestFromConvoySubtask records convoy subtask completion as knowledge (best-effort).
func (s *Service) IngestFromConvoySubtask(ctx context.Context, parent *domain.Task, convoyID domain.ConvoyID, st *domain.Subtask) error {
	if !ingestEnabled(s) || parent == nil || st == nil {
		return nil
	}
	title := strings.TrimSpace(st.Title)
	if title == "" {
		title = string(st.ID)
	}
	var b strings.Builder
	b.WriteString("Convoy subtask completed: ")
	b.WriteString(title)
	b.WriteString(" (role ")
	b.WriteString(strings.TrimSpace(st.AgentRole))
	b.WriteString(") for parent task ")
	b.WriteString(string(parent.ID))
	b.WriteString(".")
	if strings.TrimSpace(st.ExternalRef) != "" {
		b.WriteString("\nExternal ref: ")
		b.WriteString(strings.TrimSpace(st.ExternalRef))
	}
	content := truncateRunes(strings.TrimSpace(b.String()), maxAutoIngestRunes)
	meta := map[string]any{
		"ingest":     "auto",
		"source":     "convoy_subtask",
		"convoy_id":  string(convoyID),
		"subtask_id": string(st.ID),
		"agent_role": strings.TrimSpace(st.AgentRole),
		"parent_id":  string(parent.ID),
	}
	_, err := s.Create(ctx, parent.ProductID, content, parent.ID, meta)
	return err
}
