package ai

import (
	"fmt"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
)

// ProductContextSnippet formats Mission Control–style profile fields for prompts.
// Real LLM adapters should embed similar context; stubs use it for deterministic tests.
func ProductContextSnippet(p domain.Product) string {
	var b strings.Builder
	if s := strings.TrimSpace(p.Description); s != "" {
		fmt.Fprintf(&b, "Description: %s\n", s)
	}
	if s := strings.TrimSpace(p.ProgramDocument); s != "" {
		fmt.Fprintf(&b, "Program:\n%s\n", s)
	}
	if s := strings.TrimSpace(p.RepoURL); s != "" {
		fmt.Fprintf(&b, "Repo: %s", s)
		if br := strings.TrimSpace(p.RepoBranch); br != "" {
			fmt.Fprintf(&b, " @ %s", br)
		}
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}
