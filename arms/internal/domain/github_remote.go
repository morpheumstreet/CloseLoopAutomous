package domain

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseGitHubRepoURL extracts owner and repo from common GitHub remote forms.
// Supports https://github.com/o/r, https://github.com/o/r.git, git@github.com:o/r.git, and "o/r".
func ParseGitHubRepoURL(raw string) (owner, repo string, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", fmt.Errorf("empty repo url")
	}
	if strings.Count(s, "/") == 1 && !strings.Contains(s, "://") && !strings.Contains(s, "@") {
		parts := strings.Split(s, "/")
		return strings.TrimSpace(parts[0]), strings.TrimSuffix(strings.TrimSpace(parts[1]), ".git"), nil
	}
	if strings.HasPrefix(s, "git@github.com:") {
		path := strings.TrimPrefix(s, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid ssh github remote")
		}
		return parts[0], parts[1], nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", "", err
	}
	if u.Host != "github.com" && u.Host != "www.github.com" {
		return "", "", fmt.Errorf("host %q is not github.com", u.Host)
	}
	path := strings.Trim(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("path must be owner/repo")
	}
	return parts[0], parts[1], nil
}

// ParseGitHubLikeOwnerRepo extracts owner/repo from https://host/o/r or git@host:o/r style URLs.
// Use when GitHub Enterprise host is not github.com but the path is still owner/repo.
func ParseGitHubLikeOwnerRepo(raw string) (owner, repo string, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", fmt.Errorf("empty repo url")
	}
	// Reuse strict parser when it applies.
	if o, r, e := ParseGitHubRepoURL(s); e == nil {
		return o, r, nil
	}
	u, err := url.Parse(s)
	if err == nil && u.Host != "" {
		path := strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/")
		parts := strings.Split(path, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], nil
		}
	}
	if strings.HasPrefix(s, "git@") {
		idx := strings.Index(s, ":")
		if idx > 0 {
			path := strings.TrimSuffix(strings.TrimSpace(s[idx+1:]), ".git")
			parts := strings.Split(path, "/")
			if len(parts) == 2 {
				return parts[0], parts[1], nil
			}
		}
	}
	return ParseGitHubRepoURL(s)
}
