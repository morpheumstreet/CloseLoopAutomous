package researchclaw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"0.3.3"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	h, v, err := Probe(context.Background(), srv.URL, "", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if h == nil || v == nil {
		t.Fatalf("want health and version maps")
	}
}
