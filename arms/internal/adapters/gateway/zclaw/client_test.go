package zclaw

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestJoinChatURL(t *testing.T) {
	if got := joinChatURL("http://127.0.0.1:8787"); got != "http://127.0.0.1:8787/api/chat" {
		t.Fatalf("got %q", got)
	}
	if got := joinChatURL("http://127.0.0.1:8787/"); got != "http://127.0.0.1:8787/api/chat" {
		t.Fatalf("got %q", got)
	}
	if got := joinChatURL("http://127.0.0.1:8787/api/chat"); got != "http://127.0.0.1:8787/api/chat" {
		t.Fatalf("got %q", got)
	}
}

func TestClient_DispatchTaskWithSession(t *testing.T) {
	var gotKey string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)
			return
		}
		gotKey = r.Header.Get("X-Zclaw-Key")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"reply":"ok","bridge_target":"/dev/cu.usbmodem1","elapsed_ms":42}`))
	}))
	defer srv.Close()

	c := New(Options{
		BaseURL: srv.URL,
		Token:   "secret",
		Timeout: 5 * time.Second,
	})
	ref, err := c.DispatchTaskWithSession(context.Background(), domain.Task{
		ID: "t1", ProductID: "p1", IdeaID: "i1", Status: domain.StatusPlanning, Spec: "do it",
	}, "default")
	if err != nil {
		t.Fatal(err)
	}
	if ref != "zclaw:42ms:/dev/cu.usbmodem1" {
		t.Fatalf("ref %q", ref)
	}
	if gotKey != "secret" {
		t.Fatalf("X-Zclaw-Key %q", gotKey)
	}
	msg, _ := gotBody["message"].(string)
	if !strings.Contains(msg, "ARMS TASK DISPATCH") || !strings.Contains(msg, "t1") {
		t.Fatalf("unexpected body: %s", msg)
	}
}

func TestClient_DispatchHTTPErrorJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Unauthorized"}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, Timeout: time.Second})
	_, err := c.DispatchTaskWithSession(context.Background(), domain.Task{ID: "t"}, "s")
	if err == nil || !strings.Contains(err.Error(), "Unauthorized") {
		t.Fatalf("err %v", err)
	}
}

func TestTruncateToMaxRunes(t *testing.T) {
	long := strings.Repeat("x", maxMessageRunes+10)
	if utf8.RuneCountInString(truncateToMaxRunes(long)) != maxMessageRunes {
		t.Fatal("truncate length")
	}
}
