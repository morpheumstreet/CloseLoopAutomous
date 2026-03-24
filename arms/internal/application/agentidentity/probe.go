package agentidentity

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Drivers where we send Authorization: Bearer gateway_token on HTTP reachability probe.
var httpBearerProbeDrivers = map[string]struct{}{
	domain.GatewayDriverMetaClawHTTP:      {},
	domain.GatewayDriverMisterMorphHTTP:   {},
	domain.GatewayDriverCoPawHTTP:         {},
	domain.GatewayDriverZClawRelayHTTP:    {},
}

// Drivers that typically require a non-empty gateway_token for remote HTTP; missing token is an auth/config issue.
var httpExpectsBearerToken = map[string]struct{}{
	domain.GatewayDriverMetaClawHTTP:    {},
	domain.GatewayDriverMisterMorphHTTP: {},
	domain.GatewayDriverCoPawHTTP:       {},
	domain.GatewayDriverZClawRelayHTTP:  {},
}

// enrichReachability adjusts identity status after a lightweight HTTP probe (never WebSocket dial).
// Offline means "could not reach or probe not applicable"; unauthorized means "reachable but rejected credentials".
func enrichReachability(ctx context.Context, ep *domain.GatewayEndpoint, ident *domain.AgentIdentity) {
	if ident == nil || ep == nil {
		return
	}
	if ep.Driver == domain.GatewayDriverStub {
		return
	}
	raw := strings.TrimSpace(ep.GatewayURL)
	if raw == "" {
		ident.Custom["reachability"] = "no_gateway_url"
		return
	}
	low := strings.ToLower(raw)
	if strings.HasPrefix(low, "ws://") || strings.HasPrefix(low, "wss://") {
		ident.Custom["reachability"] = "websocket"
		ident.Custom["status_note"] = "Identity refresh does not open a WebSocket; auth is validated on dispatch. Offline here does not mean your token was rejected."
		return
	}
	if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
		ident.Custom["reachability"] = "non_http_url"
		return
	}
	if _, ok := httpBearerProbeDrivers[ep.Driver]; !ok {
		ident.Custom["reachability"] = "probe_skipped_driver"
		return
	}
	if _, needTok := httpExpectsBearerToken[ep.Driver]; needTok && strings.TrimSpace(ep.GatewayToken) == "" {
		ident.Status = domain.StatusUnauthorized
		ident.Custom["auth_error"] = "missing_gateway_token"
		ident.Custom["reachability"] = "not_probed"
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		ident.Custom["reachability"] = "bad_url"
		return
	}
	tok := strings.TrimSpace(ep.GatewayToken)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	req.Header.Set("Accept", "application/json, */*;q=0.8")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ident.Status = domain.StatusOffline
		ident.Custom["reachability"] = "unreachable"
		ident.Custom["reachability_error"] = err.Error()
		return
	}
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		ident.Status = domain.StatusUnauthorized
		ident.Custom["auth_error"] = fmt.Sprintf("http_%d", resp.StatusCode)
		ident.Custom["reachability"] = "rejected_credentials"
	case http.StatusProxyAuthRequired:
		ident.Status = domain.StatusUnauthorized
		ident.Custom["auth_error"] = "http_407"
		ident.Custom["reachability"] = "rejected_credentials"
	default:
		if resp.StatusCode >= 500 {
			ident.Status = domain.StatusError
			ident.Custom["reachability"] = "server_error"
			ident.Custom["http_status"] = resp.StatusCode
			return
		}
		ident.Status = domain.StatusOnline
		ident.Custom["reachability"] = "http_ok"
		ident.Custom["http_status"] = resp.StatusCode
	}
}
