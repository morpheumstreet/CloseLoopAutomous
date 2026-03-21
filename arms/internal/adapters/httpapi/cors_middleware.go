package httpapi

import "net/http"

// CORSMiddleware enables browser calls from a separate origin (e.g. Fishtank on :3000 to arms on :8080).
// When allowOrigin is empty, next is returned unchanged.
func CORSMiddleware(allowOrigin string, next http.Handler) http.Handler {
	if allowOrigin == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
