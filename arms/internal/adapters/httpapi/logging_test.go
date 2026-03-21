package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDPropagates(t *testing.T) {
	const clientRID = "client-trace-abc"
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != clientRID {
			t.Errorf("context request_id %q want %q", got, clientRID)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(RequestIDMiddleware(inner))
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set(RequestIDHeader, clientRID)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.Header.Get(RequestIDHeader) != clientRID {
		t.Fatalf("response %s=%q", RequestIDHeader, res.Header.Get(RequestIDHeader))
	}
}

func TestRequestIDGenerated(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Error("expected generated request id in context")
		}
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	RequestIDMiddleware(inner).ServeHTTP(rec, req)
	if rec.Header().Get(RequestIDHeader) == "" {
		t.Fatal("missing response X-Request-ID")
	}
}
