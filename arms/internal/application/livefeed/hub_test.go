package livefeed

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

func TestHubPublishSubscribe(t *testing.T) {
	h := NewHub()
	ch, unsub := h.Subscribe()
	defer unsub()
	ev := ports.LiveActivityEvent{Type: "task_dispatched", Ts: time.Now().UTC().Format(time.RFC3339Nano), TaskID: "t1"}
	if err := h.Publish(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	select {
	case b := <-ch:
		if !bytes.Contains(b, []byte(`task_dispatched`)) {
			t.Fatalf("payload %s", b)
		}
		var got ports.LiveActivityEvent
		if err := json.Unmarshal(b, &got); err != nil || got.TaskID != "t1" {
			t.Fatalf("decode %+v err %v", got, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}
