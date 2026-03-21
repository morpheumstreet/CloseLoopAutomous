package shipping

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v66/github"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// GitHubPublisher creates pull requests via the GitHub REST API (github.com or GHES).
type GitHubPublisher struct {
	client *github.Client
}

// NewGitHubPublisher builds a client with a PAT or OAuth token. apiBaseURL is optional
// (e.g. https://github.example.com/api/v3/ for GitHub Enterprise).
func NewGitHubPublisher(token, apiBaseURL string) (*GitHubPublisher, error) {
	c := github.NewClient(nil).WithAuthToken(strings.TrimSpace(token))
	base := strings.TrimSpace(apiBaseURL)
	if base != "" {
		if !strings.HasSuffix(base, "/") {
			base += "/"
		}
		u, err := url.Parse(base)
		if err != nil {
			return nil, err
		}
		c.BaseURL = u
		c.UploadURL = u
	}
	return &GitHubPublisher{client: c}, nil
}

var _ ports.PullRequestPublisher = (*GitHubPublisher)(nil)

// CreatePullRequest opens a PR from HeadBranch into BaseBranch on Owner/Repo.
func (g *GitHubPublisher) CreatePullRequest(ctx context.Context, in ports.CreatePullRequestInput) (ports.CreatePullRequestResult, error) {
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
	pr := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(head),
		Base:  github.String(base),
		Body:  github.String(in.Body),
	}
	created, resp, err := g.client.PullRequests.Create(ctx, owner, repo, pr)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return ports.CreatePullRequestResult{}, fmt.Errorf("%w: unauthorized (check token scopes: repo)", domain.ErrShipping)
		}
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: %v", domain.ErrShipping, err)
	}
	if created == nil || created.HTMLURL == nil || strings.TrimSpace(*created.HTMLURL) == "" {
		return ports.CreatePullRequestResult{}, fmt.Errorf("%w: empty response", domain.ErrShipping)
	}
	n := 0
	if created.Number != nil {
		n = *created.Number
	}
	return ports.CreatePullRequestResult{HTMLURL: *created.HTMLURL, Number: n}, nil
}

// PublisherSettings selects how POST /api/tasks/{id}/pull-request opens a PR (REST PAT vs gh CLI).
type PublisherSettings struct {
	PRBackend  string // gh | cli | gh-cli | api | empty (empty ⇒ api when token set)
	APIToken   string
	APIBaseURL string
	GhPath     string
	GitHubHost string // GH_HOST when using gh against GitHub Enterprise
}

// NewPullRequestPublisher returns a publisher from settings:
//   - PRBackend gh/cli/gh-cli → [GhCLIPublisher] (uses `gh auth`, not APIToken)
//   - else when APIToken non-empty → [GitHubPublisher]
//   - else [PullRequestNoop]
func NewPullRequestPublisher(s PublisherSettings) ports.PullRequestPublisher {
	b := strings.ToLower(strings.TrimSpace(s.PRBackend))
	switch b {
	case "gh", "cli", "gh-cli":
		return NewGhCLIPublisher(s.GhPath, s.GitHubHost)
	case "api", "":
		if strings.TrimSpace(s.APIToken) == "" {
			return PullRequestNoop{}
		}
		g, err := NewGitHubPublisher(s.APIToken, s.APIBaseURL)
		if err != nil {
			return PullRequestNoop{}
		}
		return g
	default:
		return PullRequestNoop{}
	}
}
