package shipping

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

var ghPRNumberRE = regexp.MustCompile(`/pull/(\d+)`)

// GhCLIPublisher opens pull requests by shelling out to the GitHub CLI (`gh pr create`).
// Auth comes from `gh auth login` (or CI tokens gh understands), not from ARMS_GITHUB_TOKEN.
type GhCLIPublisher struct {
	// GhPath is the gh executable; empty means look up "gh" on PATH.
	GhPath string
	// EnterpriseHost, if set, is passed as GH_HOST (GitHub Enterprise Server hostname, no scheme).
	EnterpriseHost string
}

var _ ports.PullRequestPublisher = (*GhCLIPublisher)(nil)

// NewGhCLIPublisher returns a publisher that invokes gh. ghPath may be empty.
func NewGhCLIPublisher(ghPath, enterpriseHost string) *GhCLIPublisher {
	return &GhCLIPublisher{GhPath: strings.TrimSpace(ghPath), EnterpriseHost: strings.TrimSpace(enterpriseHost)}
}

// CreatePullRequest runs: gh pr create --repo owner/repo --base … --head … --title … --body-file …
func (g *GhCLIPublisher) CreatePullRequest(ctx context.Context, in ports.CreatePullRequestInput) (ports.CreatePullRequestResult, error) {
	owner := strings.TrimSpace(in.Owner)
	repo := strings.TrimSpace(in.Repo)
	if owner == "" || repo == "" {
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: owner and repo required", domain.ErrInvalidInput)
	}
	base := strings.TrimSpace(in.BaseBranch)
	if base == "" {
		base = "main"
	}
	head := strings.TrimSpace(in.HeadBranch)
	if head == "" {
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: head_branch required", domain.ErrInvalidInput)
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = fmt.Sprintf("Task %s", in.TaskID)
	}

	ghBin := strings.TrimSpace(g.GhPath)
	if ghBin == "" {
		ghBin = "gh"
	}

	dir, err := os.MkdirTemp("", "arms-gh-pr-*")
	if err != nil {
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: temp dir: %v", domain.ErrShipping, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	bodyPath := filepath.Join(dir, "body.md")
	if err := os.WriteFile(bodyPath, []byte(in.Body), 0o600); err != nil {
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: body file: %v", domain.ErrShipping, err)
	}

	args := []string{
		"pr", "create",
		"--repo", owner + "/" + repo,
		"--base", base,
		"--head", head,
		"--title", title,
		"--body-file", bodyPath,
	}

	cmd := exec.CommandContext(ctx, ghBin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1")
	if g.EnterpriseHost != "" {
		cmd.Env = append(cmd.Env, "GH_HOST="+g.EnterpriseHost)
	}

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: gh: %s", domain.ErrShipping, msg)
	}

	url := strings.TrimSpace(stdout.String())
	// gh may print extra lines; first non-empty line is typically the PR URL
	for _, line := range strings.Split(url, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && (strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://")) {
			return ports.CreatePullRequestResult{HTMLURL: line, Number: parsePRNumberFromURL(line)}, nil
		}
	}
	if url != "" {
		line := strings.TrimSpace(strings.Split(url, "\n")[0])
		return ports.CreatePullRequestResult{HTMLURL: line, Number: parsePRNumberFromURL(line)}, nil
	}
	return ports.CreatePullRequestResult{}, fmt.Errorf("%w: gh produced no PR URL in stdout", domain.ErrShipping)
}

func parsePRNumberFromURL(u string) int {
	m := ghPRNumberRE.FindStringSubmatch(u)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}
