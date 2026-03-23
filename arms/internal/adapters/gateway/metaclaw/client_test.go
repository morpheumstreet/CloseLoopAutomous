package metaclaw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/closeloopautomous/arms/internal/domain"
)

func TestChatCompletionsURL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"https://h.example", "https://h.example/v1/chat/completions"},
		{"https://h.example/", "https://h.example/v1/chat/completions"},
		{"https://h.example/v1", "https://h.example/v1/chat/completions"},
		{"https://h.example/v1/", "https://h.example/v1/chat/completions"},
		{"https://h.example/v1/chat/completions", "https://h.example/v1/chat/completions"},
		{"127.0.0.1:8765", "https://127.0.0.1:8765/v1/chat/completions"},
	}
	for _, tc := range tests {
		if got := chatCompletionsURL(tc.in); got != tc.want {
			t.Fatalf("chatCompletionsURL(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestClient_DispatchTaskWithSession(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "chatcmpl-test-1"})
	}))
	defer srv.Close()

	base := strings.TrimSuffix(srv.URL, "/")
	c := New(Options{
		BaseURL:       base,
		ModelOverride: "custom-model",
		HTTPClient:    srv.Client(),
	})
	task := domain.Task{
		ID:        "t1",
		ProductID: "p1",
		Spec:      "Hello",
	}
	ref, err := c.DispatchTaskWithSession(context.Background(), task, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if ref != "chatcmpl-test-1" {
		t.Fatalf("ref %q", ref)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/chat/completions" {
		t.Fatalf("request %s %s", gotMethod, gotPath)
	}
	if gotBody["model"] != "custom-model" {
		t.Fatalf("model %v", gotBody["model"])
	}
	if gotBody["user"] != "sess-1" {
		t.Fatalf("user %v", gotBody["user"])
	}
	msgs, _ := gotBody["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages %v", gotBody["messages"])
	}
}
