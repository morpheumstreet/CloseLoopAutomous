package agentidentity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// ConnectionTestStep is one row in the Mission Control gateway connection checklist.
type ConnectionTestStep struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"` // pass, fail, skip, warn
	Detail    string `json:"detail,omitempty"`
	ElapsedMs int64  `json:"elapsed_ms,omitempty"`
}

// RunConnectionTests walks configuration → URL → transport (HTTP or WebSocket) → auth hints for Mission Control visibility.
// When the second return value is non-nil, POST /api/gateway-endpoints/{id}/test-connection persists it onto the gateway row (clears pairing fields on successful OpenClaw handshake).
func RunConnectionTests(ctx context.Context, ep *domain.GatewayEndpoint) ([]ConnectionTestStep, *domain.GatewayConnectivitySnapshot) {
	if ep == nil {
		return []ConnectionTestStep{{ID: "config", Title: "Gateway configuration", Status: "fail", Detail: "nil endpoint"}}, nil
	}
	var steps []ConnectionTestStep
	var connectivity *domain.GatewayConnectivitySnapshot
	t0 := time.Now()

	// 1) Driver / MC registry shape
	s1 := ConnectionTestStep{ID: "mc_config", Title: "Mission Control profile (driver & URL rules)"}
	drv := domain.NormalizeGatewayDriver(ep.Driver)
	if drv == "" {
		s1.Status = "fail"
		s1.Detail = "Unknown or empty driver"
		s1.ElapsedMs = time.Since(t0).Milliseconds()
		return append(steps, s1), nil
	}
	ep.Driver = drv
	if drv != domain.GatewayDriverStub && drv != domain.GatewayDriverNanobotCLI && drv != domain.GatewayDriverInkOSCLI && strings.TrimSpace(ep.GatewayURL) == "" {
		s1.Status = "fail"
		s1.Detail = fmt.Sprintf("gateway_url is required for driver %s", drv)
		s1.ElapsedMs = time.Since(t0).Milliseconds()
		return append(steps, s1), nil
	}
	s1.Status = "pass"
	s1.Detail = fmt.Sprintf("Driver %s", drv)
	s1.ElapsedMs = time.Since(t0).Milliseconds()
	steps = append(steps, s1)

	// 2) URL parse
	t1 := time.Now()
	s2 := ConnectionTestStep{ID: "url_parse", Title: "URL / endpoint address"}
	raw := strings.TrimSpace(ep.GatewayURL)
	if raw == "" && (drv == domain.GatewayDriverStub || drv == domain.GatewayDriverNanobotCLI || drv == domain.GatewayDriverInkOSCLI) {
		s2.Status = "skip"
		s2.Detail = "No remote URL for this driver"
		s2.ElapsedMs = time.Since(t1).Milliseconds()
		steps = append(steps, s2)
		steps = append(steps, cliOrStubTail(drv)...)
		return steps, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		s2.Status = "fail"
		if err != nil {
			s2.Detail = err.Error()
		} else {
			s2.Detail = "Missing scheme or host"
		}
		s2.ElapsedMs = time.Since(t1).Milliseconds()
		steps = append(steps, s2)
		return steps, nil
	}
	s2.Status = "pass"
	s2.Detail = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	s2.ElapsedMs = time.Since(t1).Milliseconds()
	steps = append(steps, s2)

	lowSc := strings.ToLower(u.Scheme)

	// 3–4) Transport + auth
	var runOpenClawAgentsList bool
	if strings.HasPrefix(lowSc, "ws") {
		if openClawHandshakeProbeDriver(drv) {
			step, snap := openClawHandshakeStep(ctx, ep)
			steps = append(steps, step)
			connectivity = snap
			runOpenClawAgentsList = step.Status == "pass"
		} else {
			steps = append(steps, wsHandshakeStep(ctx, raw, ep)...)
		}
	} else if lowSc == "http" || lowSc == "https" {
		steps = append(steps, httpTransportSteps(ctx, raw, ep)...)
	} else {
		steps = append(steps, ConnectionTestStep{
			ID: "transport", Title: "Network transport", Status: "skip",
			Detail: "Scheme not probed here (use http(s) or ws(s) for automated checks)",
		})
	}

	// agents.list verifies operator scopes for fleet discovery; keep it immediately before the MC dispatch checklist row.
	if runOpenClawAgentsList {
		steps = append(steps, openClawAgentsListStep(ctx, ep))
	}
	steps = append(steps, ConnectionTestStep{
		ID:     "mc_dispatch",
		Title:  "Dispatch path (Mission Control)",
		Status: "pass",
		Detail: "Tasks use this gateway via execution agents + gateway_endpoint_id; run a test dispatch from the board when ready.",
	})
	return steps, connectivity
}

func openClawHandshakeProbeDriver(drv string) bool {
	switch drv {
	case domain.GatewayDriverOpenClawWS, domain.GatewayDriverNemoClawWS, domain.GatewayDriverNullClawWS,
		domain.GatewayDriverZeroClawWS, domain.GatewayDriverClawletWS, domain.GatewayDriverIronClawWS:
		return true
	default:
		return false
	}
}

func openClawProbeTimeout(ep *domain.GatewayEndpoint) time.Duration {
	to := time.Duration(ep.TimeoutSec) * time.Second
	if to < 12*time.Second {
		to = 12 * time.Second
	}
	if to > 90*time.Second {
		to = 90 * time.Second
	}
	return to
}

func openClawHandshakeStep(ctx context.Context, ep *domain.GatewayEndpoint) (ConnectionTestStep, *domain.GatewayConnectivitySnapshot) {
	t0 := time.Now()
	s := ConnectionTestStep{
		ID: "ws_openclaw_handshake", Title: "OpenClaw WebSocket (handshake + pairing check)",
	}
	to := openClawProbeTimeout(ep)
	subCtx, cancel := context.WithTimeout(ctx, to+3*time.Second)
	defer cancel()

	ocOpts := openclaw.Options{
		URL:      ep.GatewayURL,
		Token:    ep.GatewayToken,
		DeviceID: ep.DeviceID,
		Timeout:  to,
	}
	openclaw.ApplyArmsDeviceEnv(&ocOpts)
	cl := openclaw.New(ocOpts)
	defer func() { _ = cl.Close() }()

	_, detail, err := cl.TestConnectionAndDetectPairing(subCtx)
	s.ElapsedMs = time.Since(t0).Milliseconds()
	if err == nil {
		s.Status = "pass"
		s.Detail = "connect.challenge → connect succeeded"
		return s, &domain.GatewayConnectivitySnapshot{}
	}
	if errors.Is(err, openclaw.ErrPairingRequired) {
		s.Status = "warn"
		s.Detail = detail
		snap := &domain.GatewayConnectivitySnapshot{
			ConnectionStatus: domain.GatewayConnectionStatusPairingRequired,
			LastCloseCode:    int(websocket.StatusPolicyViolation),
		}
		var pe *openclaw.PairingError
		if errors.As(err, &pe) && pe != nil {
			snap.PairingRequestID = pe.RequestID
			snap.PairingMessage = pe.Reason
			if pe.CloseCode != 0 {
				snap.LastCloseCode = pe.CloseCode
			}
		}
		return s, snap
	}
	s.Status = "fail"
	s.Detail = err.Error()
	return s, nil
}

// openClawAgentsListStep runs agents.list after a successful handshake (separate dial) to verify
// operator scopes (e.g. operator.read) and device approval for fleet discovery.
func openClawAgentsListStep(ctx context.Context, ep *domain.GatewayEndpoint) ConnectionTestStep {
	t0 := time.Now()
	s := ConnectionTestStep{
		ID:    "ws_openclaw_agents_list",
		Title: "OpenClaw agents.list (scopes / fleet discovery)",
	}
	to := openClawProbeTimeout(ep)
	subCtx, cancel := context.WithTimeout(ctx, to+3*time.Second)
	defer cancel()

	ocOpts := openclaw.Options{
		URL:      ep.GatewayURL,
		Token:    ep.GatewayToken,
		DeviceID: ep.DeviceID,
		Timeout:  to,
	}
	openclaw.ApplyArmsDeviceEnv(&ocOpts)
	cl := openclaw.New(ocOpts)
	defer func() { _ = cl.Close() }()

	idents, err := cl.ListAgentIdentities(subCtx)
	s.ElapsedMs = time.Since(t0).Milliseconds()
	if err != nil {
		s.Status = "fail"
		s.Detail = err.Error()
		return s
	}
	s.Status = "pass"
	s.Detail = fmt.Sprintf("agents.list OK — %d remote profile(s) (operator connect / pairing sufficient for discovery)", len(idents))
	return s
}

func cliOrStubTail(drv string) []ConnectionTestStep {
	if drv == domain.GatewayDriverStub {
		return []ConnectionTestStep{{
			ID: "transport", Title: "Remote transport", Status: "skip",
			Detail: "Stub driver — no outbound connection",
		}}
	}
	return []ConnectionTestStep{{
		ID: "transport", Title: "CLI subprocess transport",
		Status: "skip",
		Detail: "nanobot_cli / inkos_cli: connectivity is exercised when arms runs dispatch (binary on server host).",
	}}
}

func wsHandshakeStep(ctx context.Context, wsURL string, ep *domain.GatewayEndpoint) []ConnectionTestStep {
	t0 := time.Now()
	s := ConnectionTestStep{ID: "ws_handshake", Title: "WebSocket handshake (OpenClaw-class path)"}
	subCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	hdr := http.Header{}
	if tok := strings.TrimSpace(ep.GatewayToken); tok != "" {
		hdr.Set("Authorization", "Bearer "+tok)
	}
	conn, _, err := websocket.Dial(subCtx, wsURL, &websocket.DialOptions{HTTPHeader: hdr})
	if err != nil {
		s.Status = "fail"
		s.Detail = err.Error()
		s.ElapsedMs = time.Since(t0).Milliseconds()
		return []ConnectionTestStep{s}
	}
	_ = conn.Close(websocket.StatusNormalClosure, "arms gateway test")
	s.Status = "pass"
	s.Detail = "TCP/TLS + HTTP upgrade succeeded (credentials accepted at handshake layer)"
	s.ElapsedMs = time.Since(t0).Milliseconds()
	return []ConnectionTestStep{s}
}

func httpTransportSteps(ctx context.Context, raw string, ep *domain.GatewayEndpoint) []ConnectionTestStep {
	var out []ConnectionTestStep
	tTLS := time.Now()
	tlsStep := ConnectionTestStep{ID: "tls_http", Title: "HTTP(S) reachability"}
	if strings.HasPrefix(strings.ToLower(raw), "https://") {
		tlsStep.Status = "pass"
		tlsStep.Detail = "HTTPS — TLS used for transport"
	} else {
		tlsStep.Status = "warn"
		tlsStep.Detail = "Plain HTTP — no TLS to remote (acceptable on loopback only)"
	}
	tlsStep.ElapsedMs = time.Since(tTLS).Milliseconds()
	out = append(out, tlsStep)

	if _, ok := httpBearerProbeDrivers[ep.Driver]; !ok {
		out = append(out, ConnectionTestStep{
			ID: "http_auth", Title: "HTTP authorization probe", Status: "skip",
			Detail: "No automated Bearer probe for this driver; use dispatch or driver-specific tools.",
		})
		return out
	}
	if _, need := httpExpectsBearerToken[ep.Driver]; need && strings.TrimSpace(ep.GatewayToken) == "" {
		out = append(out, ConnectionTestStep{
			ID: "http_auth", Title: "HTTP authorization", Status: "fail",
			Detail: "gateway_token is empty — this driver expects a Bearer token for API access",
		})
		return out
	}

	t0 := time.Now()
	authStep := ConnectionTestStep{ID: "http_auth", Title: "HTTP GET + Bearer (API surface)"}
	subCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(subCtx, http.MethodGet, raw, nil)
	if err != nil {
		authStep.Status = "fail"
		authStep.Detail = err.Error()
		authStep.ElapsedMs = time.Since(t0).Milliseconds()
		out = append(out, authStep)
		return out
	}
	if tok := strings.TrimSpace(ep.GatewayToken); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	req.Header.Set("Accept", "application/json, */*;q=0.8")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		authStep.Status = "fail"
		authStep.Detail = err.Error()
		authStep.ElapsedMs = time.Since(t0).Milliseconds()
		out = append(out, authStep)
		return out
	}
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()
	authStep.ElapsedMs = time.Since(t0).Milliseconds()
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusProxyAuthRequired:
		authStep.Status = "fail"
		authStep.Detail = fmt.Sprintf("HTTP %d — credentials rejected", resp.StatusCode)
	case http.StatusOK, http.StatusNoContent, http.StatusCreated, http.StatusAccepted:
		authStep.Status = "pass"
		authStep.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
	default:
		if resp.StatusCode >= 500 {
			authStep.Status = "warn"
			authStep.Detail = fmt.Sprintf("HTTP %d — server error (auth may still be valid for other routes)", resp.StatusCode)
		} else {
			authStep.Status = "pass"
			authStep.Detail = fmt.Sprintf("HTTP %d — host responded (path may not be a health URL)", resp.StatusCode)
		}
	}
	out = append(out, authStep)
	return out
}
