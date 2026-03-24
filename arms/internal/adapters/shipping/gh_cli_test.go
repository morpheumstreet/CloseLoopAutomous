package shipping

import (
	"encoding/json"
	"testing"
)

func TestGhStderrLooksLikeDuplicatePR(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"random failure", false},
		{"GraphQL: A pull request already exists for morpheumstreet:CloseLoopAutomous:feature-x", true},
		{"HTTP 422: already exists", true},
		{"Pull Request already open for this branch", true},
	}
	for _, tc := range cases {
		if got := ghStderrLooksLikeDuplicatePR(tc.in); got != tc.want {
			t.Errorf("ghStderrLooksLikeDuplicatePR(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestGhPRListItemJSON(t *testing.T) {
	raw := `[{"number":42,"url":"https://github.com/o/r/pull/42"}]`
	var items []ghPRListItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Number != 42 || items[0].URL == "" {
		t.Fatalf("got %+v", items)
	}
}
