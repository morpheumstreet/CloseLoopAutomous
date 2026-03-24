package openclaw

import (
	"fmt"
	"strings"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// TaskDispatchMarkdown builds the chat.send body for a task; knowledgeBlock is appended when non-empty.
func TaskDispatchMarkdown(t domain.Task, knowledgeBlock string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**ARMS TASK DISPATCH**\n\n")
	fmt.Fprintf(&b, "**Task ID:** %s\n", t.ID)
	fmt.Fprintf(&b, "**Product ID:** %s\n", t.ProductID)
	fmt.Fprintf(&b, "**Idea ID:** %s\n", t.IdeaID)
	fmt.Fprintf(&b, "**Status:** %s\n", t.Status.String())
	if strings.TrimSpace(t.Spec) != "" {
		fmt.Fprintf(&b, "\n**Specification:**\n%s\n", t.Spec)
	}
	if strings.TrimSpace(t.Checkpoint) != "" {
		fmt.Fprintf(&b, "\n**Checkpoint:**\n%s\n", t.Checkpoint)
	}
	if strings.TrimSpace(t.ExternalRef) != "" {
		fmt.Fprintf(&b, "\n**Previous external ref:** %s\n", t.ExternalRef)
	}
	appendKnowledgeSection(&b, knowledgeBlock)
	return b.String()
}

// SubtaskDispatchMarkdown builds the chat.send body for a convoy subtask.
func SubtaskDispatchMarkdown(parent domain.TaskID, sub domain.Subtask, knowledgeBlock string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**ARMS CONVOY SUBTASK**\n\n")
	fmt.Fprintf(&b, "**Parent task ID:** %s\n", parent)
	fmt.Fprintf(&b, "**Subtask ID:** %s\n", sub.ID)
	fmt.Fprintf(&b, "**Agent role:** %s\n", sub.AgentRole)
	if len(sub.DependsOn) > 0 {
		deps := make([]string, len(sub.DependsOn))
		for i := range sub.DependsOn {
			deps[i] = string(sub.DependsOn[i])
		}
		fmt.Fprintf(&b, "**Depends on:** %s\n", strings.Join(deps, ", "))
	}
	if strings.TrimSpace(sub.Title) != "" {
		fmt.Fprintf(&b, "\n**Title:** %s\n", sub.Title)
	}
	appendKnowledgeSection(&b, knowledgeBlock)
	return b.String()
}

func appendKnowledgeSection(b *strings.Builder, knowledgeBlock string) {
	kb := strings.TrimSpace(knowledgeBlock)
	if kb == "" {
		return
	}
	fmt.Fprintf(b, "\n**Product knowledge (retrieval hints):**\n%s\n", kb)
}
