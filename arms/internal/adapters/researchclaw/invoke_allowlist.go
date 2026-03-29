package researchclaw

import (
	"net/http"
	"strings"
)

// AllowedInvokePath reports whether method+path is permitted for POST /api/research-hubs/{id}/invoke.
// Paths mirror ResearchClaw OpenAPI (e.g. videodecodetg/openapi.yaml): only /api/*, no traversal.
func AllowedInvokePath(method, path string) bool {
	m := strings.ToUpper(strings.TrimSpace(method))
	p := strings.TrimSpace(path)
	if m != http.MethodGet && m != http.MethodPost {
		return false
	}
	if p == "" || !strings.HasPrefix(p, "/api/") || strings.Contains(p, "..") {
		return false
	}
	if strings.ContainsAny(p, "\r\n\t") {
		return false
	}
	switch m {
	case http.MethodGet:
		exact := []string{
			"/api/health",
			"/api/version",
			"/api/config",
			"/api/pipeline/status",
			"/api/pipeline/stages",
			"/api/runs",
			"/api/projects",
		}
		for _, e := range exact {
			if p == e {
				return true
			}
		}
		rest, ok := strings.CutPrefix(p, "/api/runs/")
		if !ok || rest == "" {
			return false
		}
		parts := strings.Split(rest, "/")
		switch len(parts) {
		case 1:
			return parts[0] != "" && parts[0] != "metrics"
		case 2:
			return parts[0] != "" && parts[1] == "metrics"
		default:
			return false
		}
	case http.MethodPost:
		switch p {
		case "/api/pipeline/start", "/api/pipeline/stop":
			return true
		default:
			return false
		}
	default:
		return false
	}
}
