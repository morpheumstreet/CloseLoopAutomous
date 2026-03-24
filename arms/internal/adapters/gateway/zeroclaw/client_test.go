package zeroclaw

import (
	"testing"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
)

func TestNewNonNil(t *testing.T) {
	c := New(Options{URL: "ws://127.0.0.1:42617/ws", Timeout: 5})
	if c == nil || c.oc == nil {
		t.Fatal("expected non-nil client")
	}
	_ = c.Close()
}

func TestNewFromOpenClawOptionsMatchesOpenClawNew(t *testing.T) {
	o := openclaw.Options{URL: "ws://127.0.0.1:42617/ws", Timeout: 5}
	a := NewFromOpenClawOptions(o)
	b := openclaw.New(o)
	if a == nil || a.oc == nil || b == nil {
		t.Fatalf("non-nil: a=%v b=%v", a, b)
	}
	_ = a.Close()
	_ = b.Close()
}

func TestClientCloseNilSafe(t *testing.T) {
	var c *Client
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
