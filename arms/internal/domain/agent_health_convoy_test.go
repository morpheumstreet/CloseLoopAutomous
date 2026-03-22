package domain

import "testing"

func TestAgentHealthBlocksConvoyDispatch(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"", false},
		{"healthy", false},
		{"busy", false},
		{"completed", false},
		{"unknown", false},
		{"ERROR", true},
		{"stalled", true},
		{"Failed", true},
		{"dead", true},
		{"offline", true},
	} {
		if got := AgentHealthBlocksConvoyDispatch(tc.in); got != tc.want {
			t.Fatalf("AgentHealthBlocksConvoyDispatch(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
