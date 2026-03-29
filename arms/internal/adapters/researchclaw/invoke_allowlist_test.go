package researchclaw

import (
	"net/http"
	"testing"
)

func TestAllowedInvokePath(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/api/health", true},
		{http.MethodGet, "/api/runs", true},
		{http.MethodGet, "/api/runs/abc-1", true},
		{http.MethodGet, "/api/runs/abc/metrics", true},
		{http.MethodGet, "/api/runs/../x", false},
		{http.MethodGet, "https://evil/api/health", false},
		{http.MethodPost, "/api/pipeline/start", true},
		{http.MethodPost, "/api/pipeline/stop", true},
		{http.MethodPost, "/api/voice/transcribe", false},
		{"PATCH", "/api/health", false},
	}
	for _, tt := range tests {
		if got := AllowedInvokePath(tt.method, tt.path); got != tt.want {
			t.Errorf("AllowedInvokePath(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
		}
	}
}
