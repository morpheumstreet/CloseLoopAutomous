package agentidentity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

func TestEnrichReachability_HTTP401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	ep := &domain.GatewayEndpoint{
		ID:           "gw1",
		Driver:       domain.GatewayDriverMetaClawHTTP,
		GatewayURL:   srv.URL,
		GatewayToken: "bad",
	}
	ident := &domain.AgentIdentity{
		Status: domain.StatusOffline,
		Custom: map[string]any{"gateway_endpoint_id": "gw1"},
	}
	enrichReachability(context.Background(), ep, ident)
	if ident.Status != domain.StatusUnauthorized {
		t.Fatalf("status = %q want unauthorized", ident.Status)
	}
}

func TestEnrichReachability_MissingTokenMetaClaw(t *testing.T) {
	ep := &domain.GatewayEndpoint{
		ID:           "gw1",
		Driver:       domain.GatewayDriverMetaClawHTTP,
		GatewayURL:   "https://example.com/v1",
		GatewayToken: "  ",
	}
	ident := &domain.AgentIdentity{
		Status: domain.StatusOffline,
		Custom: map[string]any{},
	}
	enrichReachability(context.Background(), ep, ident)
	if ident.Status != domain.StatusUnauthorized {
		t.Fatalf("status = %q want unauthorized", ident.Status)
	}
	if ident.Custom["auth_error"] != "missing_gateway_token" {
		t.Fatalf("auth_error = %v", ident.Custom["auth_error"])
	}
}

func TestEnrichReachability_WebSocketSkipped(t *testing.T) {
	ep := &domain.GatewayEndpoint{
		ID:         "gw1",
		Driver:     domain.GatewayDriverOpenClawWS,
		GatewayURL: "wss://example.com/ws",
	}
	ident := &domain.AgentIdentity{Status: domain.StatusOffline, Custom: map[string]any{}}
	enrichReachability(context.Background(), ep, ident)
	if ident.Status != domain.StatusOffline {
		t.Fatalf("status = %q want offline (no HTTP probe)", ident.Status)
	}
	if ident.Custom["reachability"] != "websocket" {
		t.Fatalf("reachability = %v", ident.Custom["reachability"])
	}
}

func TestEnrichReachability_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ep := &domain.GatewayEndpoint{
		ID:           "gw1",
		Driver:       domain.GatewayDriverMetaClawHTTP,
		GatewayURL:   srv.URL,
		GatewayToken: "tok",
	}
	ident := &domain.AgentIdentity{Status: domain.StatusOffline, Custom: map[string]any{}}
	enrichReachability(context.Background(), ep, ident)
	if ident.Status != domain.StatusOnline {
		t.Fatalf("status = %q want online", ident.Status)
	}
}

func TestEnrichReachability_NetworkOffline(t *testing.T) {
	ep := &domain.GatewayEndpoint{
		ID:           "gw1",
		Driver:       domain.GatewayDriverMetaClawHTTP,
		GatewayURL:   "http://127.0.0.1:1",
		GatewayToken: "x",
	}
	ident := &domain.AgentIdentity{Status: domain.StatusOffline, Custom: map[string]any{}}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	enrichReachability(ctx, ep, ident)
	if ident.Status != domain.StatusOffline {
		t.Fatalf("status = %q want offline", ident.Status)
	}
	if ident.Custom["reachability"] != "unreachable" {
		t.Fatalf("reachability = %v", ident.Custom["reachability"])
	}
}
