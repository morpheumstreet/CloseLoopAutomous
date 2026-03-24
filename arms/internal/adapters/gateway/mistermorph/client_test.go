package mistermorph

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestClient_DispatchTaskWithSession_waitDone(t *testing.T) {
	var polls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(submitTaskResponse{ID: "job-1", Status: "queued"})
	})
	mux.HandleFunc("/tasks/job-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		n := polls.Add(1)
		st := "running"
		if n >= 2 {
			st = "done"
		}
		_ = json.NewEncoder(w).Encode(taskInfo{ID: "job-1", Status: st})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(Options{
		BaseURL:      srv.URL,
		Token:        "t",
		Timeout:      5 * time.Second,
		PollInterval: 5 * time.Millisecond,
		HTTPClient:   srv.Client(),
	})
	ref, err := c.DispatchTaskWithSession(context.Background(), domain.Task{
		ID:        "tsk-1",
		ProductID: "p1",
		Spec:      "do the thing",
	}, "default")
	if err != nil {
		t.Fatal(err)
	}
	if want := "mistermorph:job-1"; ref != want {
		t.Fatalf("ref = %q want %q", ref, want)
	}
}

func TestClient_postSubmit_requiresToken(t *testing.T) {
	c := New(Options{BaseURL: "http://example.com", Timeout: time.Second})
	_, err := c.postSubmit(context.Background(), submitTaskRequest{Task: "x"})
	if err == nil || !strings.Contains(err.Error(), "gateway_token") {
		t.Fatalf("expected token error, got %v", err)
	}
}
