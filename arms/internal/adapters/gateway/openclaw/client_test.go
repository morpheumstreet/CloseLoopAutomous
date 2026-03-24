package openclaw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestNativeHandshakeAndChatSend(t *testing.T) {
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			return
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			sess := context.Background()
			challenge, _ := json.Marshal(map[string]any{
				"type":    "event",
				"event":   "connect.challenge",
				"payload": map[string]any{"nonce": "abc"},
			})
			if err := c.Write(sess, websocket.MessageText, challenge); err != nil {
				t.Errorf("write challenge: %v", err)
				return
			}
			_, raw, err := c.Read(sess)
			if err != nil {
				t.Errorf("read connect: %v", err)
				return
			}
			var connReq map[string]any
			if err := json.Unmarshal(raw, &connReq); err != nil {
				t.Errorf("parse connect: %v", err)
				return
			}
			cid, _ := connReq["id"].(string)
			res, _ := json.Marshal(map[string]any{"type": "res", "id": cid, "ok": true})
			if err := c.Write(sess, websocket.MessageText, res); err != nil {
				t.Errorf("write connect res: %v", err)
				return
			}
			_, raw2, err := c.Read(sess)
			if err != nil {
				t.Errorf("read chat: %v", err)
				return
			}
			var chatReq map[string]any
			if err := json.Unmarshal(raw2, &chatReq); err != nil {
				t.Errorf("parse chat: %v", err)
				return
			}
			if chatReq["method"] != "chat.send" {
				t.Errorf("method %v", chatReq["method"])
				return
			}
			mid, _ := chatReq["id"].(string)
			out, _ := json.Marshal(map[string]any{
				"type": "res", "id": mid, "ok": true,
				"payload": map[string]any{"id": "msg-42"},
			})
			_ = c.Write(sess, websocket.MessageText, out)
		}()
		<-done
		_ = c.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	cl := New(Options{
		URL:        wsURL,
		Token:      "tok",
		SessionKey: "agent:main:test",
		Timeout:    5 * time.Second,
	})
	defer cl.Close()

	now := time.Unix(1, 0).UTC()
	ref, err := cl.DispatchTask(ctx, domain.Task{
		ID: "t1", ProductID: "p1", IdeaID: "i1", Spec: "do thing",
		Status: domain.StatusPlanning, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ref != "msg-42" {
		t.Fatalf("ref %q", ref)
	}
}

func TestReconnectAfterRPCFailure(t *testing.T) {
	ctx := context.Background()
	var wave atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			return
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			sess := context.Background()
			challenge, _ := json.Marshal(map[string]any{
				"type": "event", "event": "connect.challenge", "payload": map[string]any{},
			})
			_ = c.Write(sess, websocket.MessageText, challenge)
			_, raw, _ := c.Read(sess)
			var connReq map[string]any
			_ = json.Unmarshal(raw, &connReq)
			cid, _ := connReq["id"].(string)
			if wave.Add(1) == 1 {
				fail, _ := json.Marshal(map[string]any{
					"type": "res", "id": cid, "ok": false,
					"error": map[string]any{"message": "auth failed"},
				})
				_ = c.Write(sess, websocket.MessageText, fail)
				return
			}
			ok, _ := json.Marshal(map[string]any{"type": "res", "id": cid, "ok": true})
			_ = c.Write(sess, websocket.MessageText, ok)
			_, raw2, _ := c.Read(sess)
			var chat map[string]any
			_ = json.Unmarshal(raw2, &chat)
			mid, _ := chat["id"].(string)
			out, _ := json.Marshal(map[string]any{
				"type": "res", "id": mid, "ok": true,
				"payload": map[string]any{"id": "ok"},
			})
			_ = c.Write(sess, websocket.MessageText, out)
		}()
		<-done
		_ = c.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	cl := New(Options{URL: wsURL, SessionKey: "sk", Timeout: 5 * time.Second})
	defer cl.Close()
	now := time.Unix(1, 0).UTC()
	task := domain.Task{ID: "t", ProductID: "p", IdeaID: "i", Spec: "s", Status: domain.StatusPlanning, CreatedAt: now, UpdatedAt: now}
	if _, err := cl.DispatchTask(ctx, task); err == nil {
		t.Fatal("expected error")
	}
	ref, err := cl.DispatchTask(ctx, task)
	if err != nil {
		t.Fatal(err)
	}
	if ref != "ok" {
		t.Fatalf("ref %q", ref)
	}
}
