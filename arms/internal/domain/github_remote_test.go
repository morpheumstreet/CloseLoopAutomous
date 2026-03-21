package domain

import "testing"

func TestParseGitHubRepoURL(t *testing.T) {
	tests := []struct {
		in         string
		wantOwner  string
		wantRepo   string
		wantErr    bool
	}{
		{"https://github.com/acme/widget", "acme", "widget", false},
		{"https://github.com/acme/widget.git", "acme", "widget", false},
		{"git@github.com:acme/widget.git", "acme", "widget", false},
		{"acme/widget", "acme", "widget", false},
		{"", "", "", true},
		{"https://gitlab.com/a/b", "", "", true},
	}
	for _, tt := range tests {
		o, r, err := ParseGitHubRepoURL(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("%q: want error", tt.in)
			}
			continue
		}
		if err != nil || o != tt.wantOwner || r != tt.wantRepo {
			t.Fatalf("%q: got %q %q err %v", tt.in, o, r, err)
		}
	}
}

func TestParseGitHubLikeOwnerRepo_GHES(t *testing.T) {
	o, r, err := ParseGitHubLikeOwnerRepo("https://github.example.com/myorg/myrepo")
	if err != nil || o != "myorg" || r != "myrepo" {
		t.Fatalf("got %q %q err %v", o, r, err)
	}
}
