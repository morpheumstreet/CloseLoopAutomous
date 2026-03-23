package mimiclaw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/closeloopautomous/arms/internal/domain"
)

func TestClient_MessageFrameBearer(t *testing.T) {
	ctx := context.Background()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")
		sess := context.Background()
		_, raw, err := c.Read(sess)
		if err != nil {
			return
		}
		var msg wireOutbound
		if json.Unmarshal(raw, &msg) != nil {
			t.Error("unmarshal body")
			return
		}
		if msg.Type != typeMessage {
			t.Errorf("type %q", msg.Type)
			return
		}
		if msg.ChatID != "arms-1" {
			t.Errorf("chat_id %q", msg.ChatID)
			return
		}
		if !strings.Contains(msg.Content, "ARMS TASK DISPATCH") {
			t.Errorf("content missing dispatch header: %q", msg.Content)
			return
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	cl := New(Options{URL: wsURL, Token: "secret", Timeout: 5 * time.Second})
	defer cl.Close()

	now := time.Unix(1, 0).UTC()
	ref, err := cl.DispatchTaskWithSession(ctx, domain.Task{
		ID: "t1", ProductID: "p1", IdeaID: "i1", Spec: "work",
		Status: domain.StatusAssigned, PlanApproved: true, CreatedAt: now, UpdatedAt: now,
	}, "arms-1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(ref) == "" {
		t.Fatal("empty ref")
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization %q", gotAuth)
	}
}

func TestClient_RequiresSession(t *testing.T) {
	cl := New(Options{URL: "ws://localhost:9/x", Timeout: time.Second})
	defer cl.Close()
	_, err := cl.DispatchTaskWithSession(context.Background(), domain.Task{ID: "t"}, "  ")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTruncateChatID(t *testing.T) {
	long := strings.Repeat("a", 40)
	got := truncateChatID(long)
	if len([]rune(got)) != maxChatIDRunes {
		t.Fatalf("len %d want %d", len([]rune(got)), maxChatIDRunes)
	}
}
