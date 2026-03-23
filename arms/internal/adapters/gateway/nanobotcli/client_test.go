package nanobotcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildAgentArgs(t *testing.T) {
	got := buildAgentArgs("cli:direct", "hello", "", "")
	want := []string{"agent", "-m", "hello", "--no-markdown", "-s", "cli:direct"}
	if len(got) != len(want) {
		t.Fatalf("len got %d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx %d: got %q want %q (full %v)", i, got[i], want[i], got)
		}
	}

	withOpt := buildAgentArgs("t:1", "x", "/cfg.json", "/ws")
	if len(withOpt) != 10 {
		t.Fatalf("expected 10 args, got %d: %v", len(withOpt), withOpt)
	}
	// agent -m x --no-markdown -s t:1 -c /cfg.json -w /ws
	if withOpt[6] != "-c" || withOpt[7] != "/cfg.json" || withOpt[8] != "-w" || withOpt[9] != "/ws" {
		t.Fatalf("unexpected optional args tail: %v", withOpt[6:])
	}
}

func TestExpandPath_home(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	got := expandPath("~/nanobot/ws")
	want := filepath.Join(home, "nanobot/ws")
	if got != want {
		t.Fatalf("expandPath(~/nanobot/ws) = %q want %q", got, want)
	}
}
