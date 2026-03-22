package domain

import "testing"

func TestNormalizeIdeaCategory(t *testing.T) {
	if got := NormalizeIdeaCategory(""); got != "feature" {
		t.Fatalf("empty: %q", got)
	}
	if got := NormalizeIdeaCategory("UX"); got != "ux" {
		t.Fatalf("ux: %q", got)
	}
	if got := NormalizeIdeaCategory("not-a-real-cat"); got != "feature" {
		t.Fatalf("unknown -> feature: %q", got)
	}
}
