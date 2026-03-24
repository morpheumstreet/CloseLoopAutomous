package shipping

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v66/github"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// GitHubPRMerger merges pull requests via GitHub REST (same auth stack as [GitHubPublisher]).
type GitHubPRMerger struct {
	client *github.Client
}

// NewGitHubPRMerger builds a merger client; token and apiBaseURL follow [NewGitHubPublisher].
func NewGitHubPRMerger(token, apiBaseURL string) (*GitHubPRMerger, error) {
	p, err := NewGitHubPublisher(token, apiBaseURL)
	if err != nil {
		return nil, err
	}
	return &GitHubPRMerger{client: p.client}, nil
}

var (
	_ ports.PullRequestMerger          = (*GitHubPRMerger)(nil)
	_ ports.PullRequestMergeGateChecker = (*GitHubPRMerger)(nil)
)

// CheckMergeGates enforces review + GitHub mergeable_state before unattended merge.
func (g *GitHubPRMerger) CheckMergeGates(ctx context.Context, owner, repo string, prNumber int, gates domain.MergeExecutionGates) error {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" || prNumber <= 0 {
		return fmt.Errorf("%w: merge gate input", domain.ErrInvalidInput)
	}
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrShipping, err)
	}
	if pr.GetMerged() {
		return fmt.Errorf("%w: pull request already merged", domain.ErrInvalidInput)
	}
	if gates.RequireCleanMergeable {
		if pr.Mergeable != nil && !*pr.Mergeable {
			return domain.ErrMergeGatesNotMet
		}
		ms := strings.ToLower(strings.TrimSpace(pr.GetMergeableState()))
		if ms != "clean" {
			return domain.ErrMergeGatesNotMet
		}
	}
	if gates.RequireApprovedReview {
		reviews, _, err := g.client.PullRequests.ListReviews(ctx, owner, repo, prNumber, nil)
		if err != nil {
			return fmt.Errorf("%w: list reviews: %v", domain.ErrShipping, err)
		}
		latest := make(map[int64]string)
		for _, rv := range reviews {
			if rv == nil || rv.User == nil {
				continue
			}
			latest[rv.User.GetID()] = strings.TrimSpace(rv.GetState())
		}
		ok := false
		for _, st := range latest {
			if strings.EqualFold(st, "APPROVED") {
				ok = true
				break
			}
		}
		if !ok {
			return domain.ErrMergeGatesNotMet
		}
	}
	return nil
}

// MergePullRequest merges an open PR using merge|squash|rebase.
func (g *GitHubPRMerger) MergePullRequest(ctx context.Context, owner, repo string, prNumber int, mergeMethod string) (domain.MergeShipResult, error) {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" || prNumber <= 0 {
		return domain.MergeShipResult{State: domain.MergeShipFailed, ErrorMessage: "owner, repo, and pr number required"}, fmt.Errorf("%w: merge input", domain.ErrInvalidInput)
	}
	method := domain.NormalizeMergeMethod(mergeMethod)
	msg := ""
	opts := &github.PullRequestOptions{
		MergeMethod: method,
	}
	res, resp, err := g.client.PullRequests.Merge(ctx, owner, repo, prNumber, msg, opts)
	if err != nil {
		out, _ := githubMergeErrToResult(resp, err)
		if out.State == domain.MergeShipConflict {
			return out, fmt.Errorf("%w: %s", domain.ErrMergeConflict, out.ErrorMessage)
		}
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return out, errors.Join(
				domain.ErrShippingNonRetryable,
				fmt.Errorf("%w: unauthorized (check token scopes: repo)", domain.ErrShipping),
			)
		}
		return out, fmt.Errorf("%w: %s", domain.ErrShipping, out.ErrorMessage)
	}
	if res != nil && res.SHA != nil && strings.TrimSpace(*res.SHA) != "" {
		return domain.MergeShipResult{State: domain.MergeShipMerged, MergedSHA: strings.TrimSpace(*res.SHA)}, nil
	}
	// Merged with empty SHA — fetch PR for merge_commit_sha
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return domain.MergeShipResult{State: domain.MergeShipMerged, ErrorMessage: "merge ok but could not read SHA: " + err.Error()}, nil
	}
	sha := ""
	if pr != nil && pr.MergeCommitSHA != nil {
		sha = strings.TrimSpace(*pr.MergeCommitSHA)
	}
	return domain.MergeShipResult{State: domain.MergeShipMerged, MergedSHA: sha}, nil
}

func githubMergeErrToResult(resp *github.Response, err error) (domain.MergeShipResult, error) {
	msg := err.Error()
	st := domain.MergeShipFailed
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusConflict, http.StatusUnprocessableEntity:
			st = domain.MergeShipConflict
		case http.StatusMethodNotAllowed:
			st = domain.MergeShipConflict
		}
	}
	if strings.Contains(strings.ToLower(msg), "merge conflict") || strings.Contains(strings.ToLower(msg), "not mergeable") {
		st = domain.MergeShipConflict
	}
	return domain.MergeShipResult{State: st, ErrorMessage: msg}, err
}

// NewPullRequestMergerFromConfig returns nil when token is empty.
func NewPullRequestMergerFromConfig(token, apiBaseURL string) ports.PullRequestMerger {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	m, err := NewGitHubPRMerger(token, apiBaseURL)
	if err != nil {
		return nil
	}
	return m
}
