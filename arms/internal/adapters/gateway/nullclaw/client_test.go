package nullclaw

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

func TestJoinA2AURL(t *testing.T) {
	if got := joinA2AURL("http://127.0.0.1:3000"); got != "http://127.0.0.1:3000/a2a" {
		t.Fatalf("got %q", got)
	}
	if got := joinA2AURL("http://127.0.0.1:3000/"); got != "http://127.0.0.1:3000/a2a" {
		t.Fatalf("got %q", got)
	}
	if got := joinA2AURL("http://127.0.0.1:3000/a2a"); got != "http://127.0.0.1:3000/a2a" {
		t.Fatalf("got %q", got)
	}
}

func TestClient_DispatchTaskWithSession(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a2a" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"x","result":{"id":"task-abc"}}`))
	}))
	defer srv.Close()

	c := New(Options{
		BaseURL: srv.URL,
		Token:   "tok",
		Timeout: 5 * time.Second,
	})
	ref, err := c.DispatchTaskWithSession(context.Background(), domain.Task{
		ID: "t1", ProductID: "p1", IdeaID: "i1", Status: domain.StatusPlanning, Spec: "do it",
	}, "ctx-1")
	if err != nil {
		t.Fatal(err)
	}
	if ref != "task-abc" {
		t.Fatalf("ref %q", ref)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("auth %q", gotAuth)
	}
	if gotBody["method"] != "message/send" {
		t.Fatalf("method %+v", gotBody["method"])
	}
	params, _ := gotBody["params"].(map[string]any)
	msg, _ := params["message"].(map[string]any)
	if msg["contextId"] != "ctx-1" {
		t.Fatalf("contextId %+v", msg["contextId"])
	}
	parts, _ := msg["parts"].([]any)
	p0, _ := parts[0].(map[string]any)
	txt, _ := p0["text"].(string)
	if !strings.Contains(txt, "ARMS TASK DISPATCH") || !strings.Contains(txt, "t1") {
		t.Fatalf("unexpected body: %s", txt)
	}
}

func TestClient_JSONRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"bad"}}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, Timeout: time.Second})
	_, err := c.DispatchTaskWithSession(context.Background(), domain.Task{ID: "t"}, "s")
	if err == nil || !strings.Contains(err.Error(), "bad") {
		t.Fatalf("err %v", err)
	}
}
