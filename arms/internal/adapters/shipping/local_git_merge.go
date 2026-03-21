package shipping

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// LocalGitMerger runs git fetch/checkout/merge inside a clone ([ports.WorktreeMerger]).
type LocalGitMerger struct{}

// NewLocalGitMerger is a stateless adapter (safe for concurrent use).
func NewLocalGitMerger() *LocalGitMerger { return &LocalGitMerger{} }

var _ ports.WorktreeMerger = (*LocalGitMerger)(nil)

func (LocalGitMerger) MergeBranches(ctx context.Context, in ports.WorktreeMergeInput) (domain.MergeShipResult, error) {
	root := filepath.Clean(strings.TrimSpace(in.RepoRoot))
	if root == "" || root == "." {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: "repo root required"}, fmt.Errorf("%w: repo root", domain.ErrInvalidInput)
	}
	base := strings.TrimSpace(in.BaseBranch)
	head := strings.TrimSpace(in.HeadBranch)
	if base == "" || head == "" {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: "base and head branch required"}, fmt.Errorf("%w: branches", domain.ErrInvalidInput)
	}
	gitBin := strings.TrimSpace(in.GitBin)
	if gitBin == "" {
		gitBin = "git"
	}
	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, gitBin, args...)
		cmd.Dir = root
		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		err := cmd.Run()
		if err != nil {
			msg := strings.TrimSpace(errb.String())
			if msg == "" {
				msg = err.Error()
			}
			return outb.String(), fmt.Errorf("%s", msg)
		}
		return outb.String(), nil
	}
	if _, err := run("fetch", "origin"); err != nil {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: err.Error()}, err
	}
	if _, err := run("checkout", base); err != nil {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: err.Error()}, err
	}
	mergeRef := head
	if !strings.Contains(head, "/") {
		mergeRef = "origin/" + head
	}
	if _, err := run("merge", mergeRef, "--no-edit"); err != nil {
		un, uerr := run("diff", "--name-only", "--diff-filter=U")
		files := []string{}
		if uerr == nil {
			for _, line := range strings.Split(strings.TrimSpace(un), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					files = append(files, line)
				}
			}
		}
		return domain.MergeShipResult{State: domain.MergeShipConflict, ErrorMessage: err.Error(), ConflictFiles: files}, domain.ErrMergeConflict
	}
	shaOut, err := run("rev-parse", "HEAD")
	if err != nil {
		return domain.MergeShipResult{State: domain.MergeShipMerged, ErrorMessage: "merged but rev-parse failed: " + err.Error()}, nil
	}
	return domain.MergeShipResult{State: domain.MergeShipMerged, MergedSHA: strings.TrimSpace(shaOut)}, nil
}
