package openclaw

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ed25519SPKIPrefix is the fixed SPKI wrapper for Ed25519 public keys (same as Mission Control device-identity.ts).
var ed25519SPKIPrefix = []byte{0x30, 0x2a, 0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x70, 0x03, 0x21, 0x00}

type mcDeviceFile struct {
	Version       int    `json:"version"`
	DeviceID      string `json:"deviceId"`
	PublicKeyPem  string `json:"publicKeyPem"`
	PrivateKeyPem string `json:"privateKeyPem"`
}

type deviceIdentity struct {
	DeviceID     string
	privateKey   ed25519.PrivateKey
	publicKeyB64 string // raw 32-byte pubkey, base64url (wire field "publicKey")
}

func defaultMissionControlDevicePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mission-control", "identity", "device.json"), nil
}

func loadMissionControlDeviceJSON(path string) (*deviceIdentity, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read identity file: %w", err)
	}
	var f mcDeviceFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse identity json: %w", err)
	}
	if f.Version != 1 {
		return nil, fmt.Errorf("identity version %d not supported (want 1)", f.Version)
	}
	derived, err := fingerprintPublicKeyPEM(f.PublicKeyPem)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(f.DeviceID) != "" && !strings.EqualFold(strings.TrimSpace(f.DeviceID), derived) {
		return nil, fmt.Errorf("deviceId does not match public key fingerprint")
	}
	rawPub, err := rawPublicKeyFromSPKIPEM(f.PublicKeyPem)
	if err != nil {
		return nil, err
	}
	priv, err := parseEd25519PrivateKeyPEM(f.PrivateKeyPem)
	if err != nil {
		return nil, err
	}
	return &deviceIdentity{
		DeviceID:     derived,
		privateKey:   priv,
		publicKeyB64: base64URLEncode(rawPub),
	}, nil
}

func fingerprintPublicKeyPEM(pemStr string) (string, error) {
	raw, err := rawPublicKeyFromSPKIPEM(pemStr)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func rawPublicKeyFromSPKIPEM(pemStr string) ([]byte, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("public key PEM decode failed")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	edPub, ok := pub.(ed25519.PublicKey)
	if !ok || len(edPub) != ed25519.PublicKeySize {
		return nil, errors.New("public key is not Ed25519")
	}
	// Prefer SPKI suffix check to match MC / browser device id derivation.
	if len(block.Bytes) == len(ed25519SPKIPrefix)+ed25519.PublicKeySize &&
		string(block.Bytes[:len(ed25519SPKIPrefix)]) == string(ed25519SPKIPrefix) {
		return block.Bytes[len(ed25519SPKIPrefix):], nil
	}
	return edPub, nil
}

func parseEd25519PrivateKeyPEM(pemStr string) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("private key PEM decode failed")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse pkcs8 private key: %w", err)
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok || len(priv) != ed25519.PrivateKeySize {
		return nil, errors.New("private key is not Ed25519")
	}
	return priv, nil
}

// buildDeviceAuthPayload matches Mission Control [device-identity.ts] buildDeviceAuthPayload (v1 without nonce, v2 with).
func buildDeviceAuthPayload(deviceID, clientID, clientMode, role string, scopes []string, signedAtMs int64, token string, nonce string) string {
	version := "v1"
	if strings.TrimSpace(nonce) != "" {
		version = "v2"
	}
	scopeStr := strings.Join(scopes, ",")
	tok := token
	base := []string{version, deviceID, clientID, clientMode, role, scopeStr, fmt.Sprintf("%d", signedAtMs), tok}
	if version == "v2" {
		base = append(base, nonce)
	}
	return strings.Join(base, "|")
}

func signDevicePayloadBase64URL(priv ed25519.PrivateKey, payload string) string {
	sig := ed25519.Sign(priv, []byte(payload))
	return base64URLEncode(sig)
}

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
