package picoclaw

import (
	"strings"
	"unicode/utf8"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func knowledgeQueryFromTask(t domain.Task) string {
	var b strings.Builder
	if strings.TrimSpace(string(t.IdeaID)) != "" {
		b.WriteString(string(t.IdeaID))
		b.WriteByte(' ')
	}
	s := strings.TrimSpace(t.Spec)
	if s == "" {
		return strings.TrimSpace(b.String())
	}
	if utf8.RuneCountInString(s) > 800 {
		s = string([]rune(s)[:800])
	}
	b.WriteString(s)
	return strings.TrimSpace(b.String())
}

func knowledgeQueryFromSubtask(parent domain.Task, sub domain.Subtask) string {
	var b strings.Builder
	if strings.TrimSpace(sub.AgentRole) != "" {
		b.WriteString(sub.AgentRole)
		b.WriteByte(' ')
	}
	if strings.TrimSpace(sub.Title) != "" {
		b.WriteString(sub.Title)
		b.WriteByte(' ')
	}
	s := strings.TrimSpace(parent.Spec)
	if utf8.RuneCountInString(s) > 600 {
		s = string([]rune(s)[:600])
	}
	b.WriteString(s)
	return strings.TrimSpace(b.String())
}
