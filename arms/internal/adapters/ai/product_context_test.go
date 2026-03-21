package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/closeloopautomous/arms/internal/domain"
)

func TestProductContextSnippet(t *testing.T) {
	p := domain.Product{
		ID:              "p1",
		Description:     "  short blurb  ",
		ProgramDocument: "Build the widget.",
		RepoURL:         "https://git.example/r",
		RepoBranch:      "main",
	}
	s := ProductContextSnippet(p)
	if !strings.Contains(s, "short blurb") || !strings.Contains(s, "Build the widget") || !strings.Contains(s, "git.example") || !strings.Contains(s, "@ main") {
		t.Fatalf("snippet: %q", s)
	}
	if ProductContextSnippet(domain.Product{ID: "x"}) != "" {
		t.Fatal("want empty")
	}
}

func TestResearchStubIncludesProgram(t *testing.T) {
	var stub ResearchStub
	sum, err := stub.RunResearch(context.Background(), domain.Product{
		ID:              "p1",
		ProgramDocument: "Ship MVP by Q2.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sum, "Ship MVP") {
		t.Fatalf("summary should embed program: %q", sum)
	}
}
