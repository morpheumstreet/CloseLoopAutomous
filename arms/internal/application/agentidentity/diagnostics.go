package agentidentity

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/closeloopautomous/arms/internal/domain"
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
func RunConnectionTests(ctx context.Context, ep *domain.GatewayEndpoint) []ConnectionTestStep {
	if ep == nil {
		return []ConnectionTestStep{{ID: "config", Title: "Gateway configuration", Status: "fail", Detail: "nil endpoint"}}
	}
	var steps []ConnectionTestStep
	t0 := time.Now()

	// 1) Driver / MC registry shape
	s1 := ConnectionTestStep{ID: "mc_config", Title: "Mission Control profile (driver & URL rules)"}
	drv := domain.NormalizeGatewayDriver(ep.Driver)
	if drv == "" {
		s1.Status = "fail"
		s1.Detail = "Unknown or empty driver"
		s1.ElapsedMs = time.Since(t0).Milliseconds()
		return append(steps, s1)
	}
	ep.Driver = drv
	if drv != domain.GatewayDriverStub && drv != domain.GatewayDriverNanobotCLI && drv != domain.GatewayDriverInkOSCLI && strings.TrimSpace(ep.GatewayURL) == "" {
		s1.Status = "fail"
		s1.Detail = fmt.Sprintf("gateway_url is required for driver %s", drv)
		s1.ElapsedMs = time.Since(t0).Milliseconds()
		return append(steps, s1)
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
		return steps
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
		return steps
	}
	s2.Status = "pass"
	s2.Detail = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	s2.ElapsedMs = time.Since(t1).Milliseconds()
	steps = append(steps, s2)

	lowSc := strings.ToLower(u.Scheme)

	// 3–4) Transport + auth
	if strings.HasPrefix(lowSc, "ws") {
		steps = append(steps, wsHandshakeStep(ctx, raw, ep)...)
	} else if lowSc == "http" || lowSc == "https" {
		steps = append(steps, httpTransportSteps(ctx, raw, ep)...)
	} else {
		steps = append(steps, ConnectionTestStep{
			ID: "transport", Title: "Network transport", Status: "skip",
			Detail: "Scheme not probed here (use http(s) or ws(s) for automated checks)",
		})
	}

	steps = append(steps, ConnectionTestStep{
		ID:     "mc_dispatch",
		Title:  "Dispatch path (Mission Control)",
		Status: "pass",
		Detail: "Tasks use this gateway via execution agents + gateway_endpoint_id; run a test dispatch from the board when ready.",
	})
	return steps
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
