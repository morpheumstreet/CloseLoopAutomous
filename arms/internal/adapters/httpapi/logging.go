package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ctxKey int

const requestIDCtxKey ctxKey = 1

// RequestIDHeader is the standard correlation header (in/out).
const RequestIDHeader = "X-Request-ID"

// RequestIDFromContext returns the request id if the request passed through RequestIDMiddleware.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(requestIDCtxKey).(string)
	return v
}

// RequestIDMiddleware ensures every response has X-Request-ID (reuses client header when present and non-empty).
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := strings.TrimSpace(r.Header.Get(RequestIDHeader))
		if rid == "" {
			rid = uuid.NewString()
		}
		w.Header().Set(RequestIDHeader, rid)
		ctx := context.WithValue(r.Context(), requestIDCtxKey, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wrote {
		s.status = code
		s.wrote = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wrote {
		s.status = http.StatusOK
		s.wrote = true
	}
	return s.ResponseWriter.Write(b)
}

// AccessLogMiddleware logs one structured line per request (method, path, status, duration, request_id).
func AccessLogMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		rid := RequestIDFromContext(r.Context())
		log.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", rid,
		)
	})
}
