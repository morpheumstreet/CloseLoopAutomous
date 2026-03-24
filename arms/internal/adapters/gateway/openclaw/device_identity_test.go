package openclaw

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func TestBuildDeviceAuthPayloadV1(t *testing.T) {
	got := buildDeviceAuthPayload("did", "cli", "ui", "operator", []string{"operator.admin"}, 1700000000000, "tok", "")
	want := "v1|did|cli|ui|operator|operator.admin|1700000000000|tok"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildDeviceAuthPayloadV2Nonce(t *testing.T) {
	got := buildDeviceAuthPayload("did", "cli", "ui", "operator", []string{"operator.admin"}, 1700000000000, "", "n1")
	want := "v2|did|cli|ui|operator|operator.admin|1700000000000||n1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSignDevicePayloadRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := buildDeviceAuthPayload("did", "cli", "ui", "operator", []string{"operator.admin"}, 42, "tok", "nonce")
	sigB64 := signDevicePayloadBase64URL(priv, payload)
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(pub, []byte(payload), sig) {
		t.Fatal("signature verify failed")
	}
}
