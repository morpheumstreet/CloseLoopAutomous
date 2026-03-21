package httpapi

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

// AuthMiddleware enforces Bearer MC_API_TOKEN and/or HTTP Basic users from ARMS_ACL when configured.
// Bearer (when MC_API_TOKEN is set) always has full access. Basic users get admin or read (GET/HEAD only) per role.
func AuthMiddleware(cfg Config, next http.Handler) http.Handler {
	if !authConfigured(cfg) {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if localhostMCBypass(cfg, r) {
			next.ServeHTTP(w, r)
			return
		}
		admin, ok := authenticate(cfg, r)
		if !ok {
			if len(cfg.ACLUsers) > 0 {
				w.Header().Set("WWW-Authenticate", `Basic realm="arms"`)
			}
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid credentials")
			return
		}
		if !admin && !readOnlyMethod(r.Method) {
			writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions for this role")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authConfigured(cfg Config) bool {
	return cfg.MCAPIToken != "" || len(cfg.ACLUsers) > 0
}

// localhostMCBypass matches legacy behavior: same-origin-style access to localhost when MC_API_TOKEN is set and ARMS_ALLOW_SAME_ORIGIN is on.
func localhostMCBypass(cfg Config, r *http.Request) bool {
	if !cfg.AllowLocalhost || cfg.MCAPIToken == "" {
		return false
	}
	host := strings.Split(r.Host, ":")[0]
	return host == "localhost" || host == "127.0.0.1"
}

func readOnlyMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func authenticate(cfg Config, r *http.Request) (admin bool, ok bool) {
	if cfg.MCAPIToken != "" && bearerOrQueryTokenOK(r, cfg) {
		return true, true
	}
	if len(cfg.ACLUsers) == 0 {
		return false, false
	}
	user, pass, hasBasic := r.BasicAuth()
	if hasBasic {
		return aclVerify(cfg, user, pass)
	}
	return false, false
}

func bearerOrQueryTokenOK(r *http.Request, cfg Config) bool {
	h := r.Header.Get("Authorization")
	const p = "Bearer "
	if strings.HasPrefix(h, p) && strings.TrimSpace(h[len(p):]) == cfg.MCAPIToken {
		return true
	}
	q := r.URL.Query().Get("token")
	if q == cfg.MCAPIToken {
		return true
	}
	return false
}

func aclVerify(cfg Config, user, pass string) (admin bool, ok bool) {
	for _, u := range cfg.ACLUsers {
		if u.UserID != user {
			continue
		}
		if !secureStringEqual(pass, u.Password) {
			return false, false
		}
		switch u.Role {
		case "admin":
			return true, true
		case "read":
			return false, true
		default:
			return false, false
		}
	}
	return false, false
}

func secureStringEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// SSEQueryToken wraps a handler and requires ?token= (MC_API_TOKEN) or ?basic= (base64 of user:password for ARMS_ACL) when auth is required.
func SSEQueryToken(cfg Config, next http.Handler) http.Handler {
	if !authConfigured(cfg) {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if localhostSSEBypass(cfg, r) {
			next.ServeHTTP(w, r)
			return
		}
		if sseAuthOK(r, cfg) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token query param")
	})
}

func localhostSSEBypass(cfg Config, r *http.Request) bool {
	if !cfg.AllowLocalhost || cfg.MCAPIToken == "" {
		return false
	}
	host := strings.Split(r.Host, ":")[0]
	return host == "localhost" || host == "127.0.0.1"
}

func sseAuthOK(r *http.Request, cfg Config) bool {
	q := r.URL.Query()
	if cfg.MCAPIToken != "" && q.Get("token") == cfg.MCAPIToken {
		return true
	}
	b64 := strings.TrimSpace(q.Get("basic"))
	if b64 != "" && len(cfg.ACLUsers) > 0 {
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return false
		}
		user, pass, ok := parseUserPassColon(string(raw))
		if !ok {
			return false
		}
		_, ok = aclVerify(cfg, user, pass)
		return ok
	}
	if cfg.MCAPIToken != "" {
		return false
	}
	if len(cfg.ACLUsers) > 0 {
		return false
	}
	return true
}

func parseUserPassColon(s string) (user, pass string, ok bool) {
	i := strings.IndexByte(s, ':')
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}
