package httpapi

import (
	"net/http"
	"strings"
)

// AuthMiddleware enforces Bearer MC_API_TOKEN when configured.
func AuthMiddleware(cfg Config, next http.Handler) http.Handler {
	if cfg.MCAPIToken == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tokenOK(r, cfg) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid bearer token")
	})
}

func tokenOK(r *http.Request, cfg Config) bool {
	h := r.Header.Get("Authorization")
	const p = "Bearer "
	if strings.HasPrefix(h, p) && strings.TrimSpace(h[len(p):]) == cfg.MCAPIToken {
		return true
	}
	// SSE / EventSource cannot set headers — allow query param (MC-style for browser).
	q := r.URL.Query().Get("token")
	if q == cfg.MCAPIToken {
		return true
	}
	if cfg.AllowLocalhost {
		host := strings.Split(r.Host, ":")[0]
		if host == "localhost" || host == "127.0.0.1" {
			return true
		}
	}
	return false
}

// SSEQueryToken wraps a handler and requires ?token= when MC_API_TOKEN is set (unless same-origin bypass).
func SSEQueryToken(cfg Config, next http.Handler) http.Handler {
	if cfg.MCAPIToken == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != cfg.MCAPIToken {
			if cfg.AllowLocalhost {
				host := strings.Split(r.Host, ":")[0]
				if host == "localhost" || host == "127.0.0.1" {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token query param")
			return
		}
		next.ServeHTTP(w, r)
	})
}
