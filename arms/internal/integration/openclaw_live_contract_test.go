//go:build integration

package integration_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Live OpenClaw contract tests (#105): hit a real gateway WebSocket endpoint.
//
// Default: skipped (no secrets in normal CI). Enable with:
//
//	ARMS_OPENCLAW_LIVE_CONTRACT=1 \
//	OPENCLAW_GATEWAY_URL=wss://... \
//	OPENCLAW_GATEWAY_TOKEN=... \
//	ARMS_OPENCLAW_SESSION_KEY=agent:main:... \
//	go test -tags=integration ./internal/integration/... -run LiveOpenClaw -count=1 -timeout 120s
//
// Optional: ARMS_DEVICE_ID, OPENCLAW_DISPATCH_TIMEOUT_SEC

func liveOpenClawEnabled(t *testing.T) (url, token, sessionKey string) {
	t.Helper()
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ARMS_OPENCLAW_LIVE_CONTRACT")))
	if v != "1" && v != "true" && v != "yes" {
		t.Skip("set ARMS_OPENCLAW_LIVE_CONTRACT=1 and gateway env to run live OpenClaw contracts (#105)")
	}
	url = strings.TrimSpace(os.Getenv("OPENCLAW_GATEWAY_URL"))
	token = strings.TrimSpace(os.Getenv("OPENCLAW_GATEWAY_TOKEN"))
	sessionKey = strings.TrimSpace(os.Getenv("ARMS_OPENCLAW_SESSION_KEY"))
	if url == "" || sessionKey == "" {
		t.Skip("OPENCLAW_GATEWAY_URL and ARMS_OPENCLAW_SESSION_KEY must be set for live contract tests (e.g. CI secrets or local env)")
	}
	return url, token, sessionKey
}

func openClawTestTimeout() time.Duration {
	if s := strings.TrimSpace(os.Getenv("OPENCLAW_DISPATCH_TIMEOUT_SEC")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 60 * time.Second
}

func TestLiveOpenClaw_ConnectAndDispatchTask(t *testing.T) {
	gwURL, tok, sk := liveOpenClawEnabled(t)
	to := openClawTestTimeout()

	cl := openclaw.New(openclaw.Options{
		URL:        gwURL,
		Token:      tok,
		DeviceID:   strings.TrimSpace(os.Getenv("ARMS_DEVICE_ID")),
		SessionKey: sk,
		Timeout:    to,
	})
	t.Cleanup(func() { _ = cl.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), to+5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	task := domain.Task{
		ID:           domain.TaskID("contract-task-" + now.Format("150405")),
		ProductID:    "contract-product",
		IdeaID:       "contract-idea",
		Spec:         "OpenClaw live contract: acknowledge and exit (automated test).",
		Status:       domain.StatusInProgress,
		PlanApproved: true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	ref, err := cl.DispatchTask(ctx, task)
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}
	if strings.TrimSpace(ref) == "" {
		t.Fatal("DispatchTask: empty external ref")
	}
	t.Logf("DispatchTask ref=%q", ref)
}

func TestLiveOpenClaw_ConnectAndDispatchSubtask(t *testing.T) {
	gwURL, tok, sk := liveOpenClawEnabled(t)
	to := openClawTestTimeout()

	cl := openclaw.New(openclaw.Options{
		URL:        gwURL,
		Token:      tok,
		DeviceID:   strings.TrimSpace(os.Getenv("ARMS_DEVICE_ID")),
		SessionKey: sk,
		Timeout:    to,
	})
	t.Cleanup(func() { _ = cl.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), to+5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	parent := domain.Task{
		ID:           domain.TaskID("contract-parent-" + now.Format("150405")),
		ProductID:    "contract-product",
		IdeaID:       "contract-idea",
		Spec:         "Parent spec for subtask contract test.",
		Status:       domain.StatusConvoyActive,
		PlanApproved: true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	sub := domain.Subtask{
		ID:        domain.SubtaskID("contract-sub-" + now.Format("150405")),
		AgentRole: "worker",
	}

	ref, err := cl.DispatchSubtask(ctx, parent, sub)
	if err != nil {
		t.Fatalf("DispatchSubtask: %v", err)
	}
	if strings.TrimSpace(ref) == "" {
		t.Fatal("DispatchSubtask: empty external ref")
	}
	t.Logf("DispatchSubtask ref=%q", ref)
}

func TestLiveOpenClaw_BadTokenFails(t *testing.T) {
	gwURL, _, sk := liveOpenClawEnabled(t)
	to := openClawTestTimeout()

	cl := openclaw.New(openclaw.Options{
		URL:        gwURL,
		Token:      "definitely-not-a-valid-token-for-contract-test",
		SessionKey: sk,
		Timeout:    to,
	})
	t.Cleanup(func() { _ = cl.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), to+5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	task := domain.Task{
		ID:           domain.TaskID("contract-badauth-" + now.Format("150405")),
		ProductID:    "contract-product",
		IdeaID:       "contract-idea",
		Spec:         "should not run",
		Status:       domain.StatusInProgress,
		PlanApproved: true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := cl.DispatchTask(ctx, task)
	if err == nil {
		t.Fatal("expected error with invalid token")
	}
	t.Logf("expected failure: %v", err)
}
