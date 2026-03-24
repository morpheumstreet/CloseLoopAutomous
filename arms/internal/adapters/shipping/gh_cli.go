package shipping

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
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
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr == "" {
			stderrStr = err.Error()
		}
		if ghStderrLooksLikeDuplicatePR(stderrStr) {
			recovered, rerr := g.listOpenPRByHead(ctx, ghBin, owner, repo, head)
			if rerr != nil {
				return ports.CreatePullRequestResult{}, rerr
			}
			if strings.TrimSpace(recovered.HTMLURL) != "" {
				return recovered, nil
			}
		}
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: gh: %s", domain.ErrShipping, stderrStr)
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

func ghStderrLooksLikeDuplicatePR(s string) bool {
	low := strings.ToLower(s)
	return strings.Contains(low, "already exists") ||
		strings.Contains(low, "pull request already") ||
		strings.Contains(low, "a pull request already")
}

type ghPRListItem struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// listOpenPRByHead runs `gh pr list` for owner:branch (same convention as REST duplicate recovery).
func (g *GhCLIPublisher) listOpenPRByHead(ctx context.Context, ghBin, owner, repo, head string) (ports.CreatePullRequestResult, error) {
	headRef := strings.TrimSpace(owner) + ":" + strings.TrimSpace(head)
	args := []string{
		"pr", "list",
		"--repo", owner + "/" + repo,
		"--head", headRef,
		"--state", "open",
		"--json", "number,url",
		"--limit", "1",
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
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: gh pr list: %s", domain.ErrShipping, msg)
	}
	var items []ghPRListItem
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: gh pr list json: %v", domain.ErrShipping, err)
	}
	if len(items) == 0 || strings.TrimSpace(items[0].URL) == "" {
		return ports.CreatePullRequestResult{}, nil
	}
	u := strings.TrimSpace(items[0].URL)
	return ports.CreatePullRequestResult{HTMLURL: u, Number: items[0].Number}, nil
}
