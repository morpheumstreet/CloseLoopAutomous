package convoy

import (
	"strings"
	"testing"
	"time"
)

func TestMergeMCCompatIntoMetadata_RoundTrip(t *testing.T) {
	at := time.Date(2020, 1, 2, 15, 4, 5, 0, time.UTC)
	meta, err := MergeMCCompatIntoMetadata(`{"foo":1}`, MCCompatFields{
		Name: "n", Strategy: "manual", Status: "active", DecompositionSpec: "x", UpdatedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(meta, `"foo":1`) || !strings.Contains(meta, mcCompatKey) {
		t.Fatalf("lost keys: %s", meta)
	}
	n, strat, st, ds, _ := MCCompatFromMetadata(meta)
	if n != "n" || strat != "manual" || st != "active" || ds != "x" {
		t.Fatalf("parse back: %q %q %q %q", n, strat, st, ds)
	}
	if !MCConvoyDispatchAllowed(meta) {
		t.Fatal("active should allow dispatch")
	}
	meta2, err := MergeMCCompatIntoMetadata(meta, MCCompatFields{Status: "paused", UpdatedAt: at})
	if err != nil {
		t.Fatal(err)
	}
	if MCConvoyDispatchAllowed(meta2) {
		t.Fatal("paused should block dispatch")
	}
}
