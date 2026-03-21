package sqlite

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/application/livefeed"
	"github.com/closeloopautomous/arms/internal/ports"
)

func TestOutboxRelayToHub(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	ob := NewOutboxStore(db)
	hub := livefeed.NewHub()
	ch, unsub := hub.Subscribe()
	defer unsub()

	ev := ports.LiveActivityEvent{Type: "task_dispatched", Ts: time.Now().UTC().Format(time.RFC3339Nano), TaskID: "x"}
	b, _ := json.Marshal(ev)
	if err := ob.Append(ctx, b); err != nil {
		t.Fatal(err)
	}
	relayCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go livefeed.RunOutboxRelay(relayCtx, ob, hub, 20*time.Millisecond)
	select {
	case payload := <-ch:
		if string(payload) != string(b) {
			t.Fatalf("got %s want %s", payload, b)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
