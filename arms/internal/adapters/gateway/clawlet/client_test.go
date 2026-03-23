package clawlet

import "testing"

func TestNewNonNil(t *testing.T) {
	c := New(Options{URL: "ws://127.0.0.1:18790/ws", Timeout: 5})
	if c == nil || c.oc == nil {
		t.Fatal("expected non-nil client")
	}
	_ = c.Close()
}

func TestClientCloseNilSafe(t *testing.T) {
	var c *Client
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
