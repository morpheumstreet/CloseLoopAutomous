package workspace

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// PrepareGitWorktree runs `git -C mainRepo worktree add worktreeAbs branch`.
func PrepareGitWorktree(ctx context.Context, gitBin, mainRepoAbs, worktreeAbs, branch string) error {
	if gitBin == "" {
		gitBin = "git"
	}
	cmd := exec.CommandContext(ctx, gitBin, "-C", mainRepoAbs, "worktree", "add", worktreeAbs, branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// EnsurePathUnderRoot checks that worktree is the same path or a subdirectory of root (after Abs + Clean).
func EnsurePathUnderRoot(root, worktree string) (cleanWorktree string, err error) {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", err
	}
	wtAbs, err := filepath.Abs(filepath.Clean(worktree))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, wtAbs)
	if err != nil {
		return "", fmt.Errorf("path under root: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("worktree path escapes workspace root")
	}
	return wtAbs, nil
}
