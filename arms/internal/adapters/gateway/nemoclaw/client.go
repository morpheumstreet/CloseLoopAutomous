package nemoclaw

import (
	"context"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Options configure a NemoClaw gateway client: OpenClaw-class WebSocket to the OpenShell-proxied
// gateway plus optional local `nemoclaw <sandbox> start` before each dispatch.
type Options struct {
	URL                  string
	Token                string
	DeviceID             string
	Timeout              time.Duration
	SandboxName          string // NemoClaw sandbox; typically gateway endpoint device_id
	NemoClawBin          string
	AutoStart            bool
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
	DeviceSigning        bool
	DeviceIdentityFile   string
}

// Client dispatches via OpenClaw WebSocket JSON-RPC after optional NemoClaw sandbox lifecycle.
type Client struct {
	oc          *openclaw.Client
	bin         string
	autoStart   bool
	sandboxName string
}

// New builds a client. SessionKey is supplied per Dispatch*WithSession call.
func New(opts Options) *Client {
	to := opts.Timeout
	if to <= 0 {
		to = 30 * time.Second
	}
	oc := openclaw.New(openclaw.Options{
		URL:                  strings.TrimSpace(opts.URL),
		Token:                strings.TrimSpace(opts.Token),
		DeviceID:             strings.TrimSpace(opts.DeviceID),
		SessionKey:           "",
		Timeout:              to,
		KnowledgeForDispatch: opts.KnowledgeForDispatch,
		DeviceSigning:        opts.DeviceSigning,
		DeviceIdentityFile:   opts.DeviceIdentityFile,
	})
	return &Client{
		oc:          oc,
		bin:         strings.TrimSpace(opts.NemoClawBin),
		autoStart:   opts.AutoStart,
		sandboxName: strings.TrimSpace(opts.SandboxName),
	}
}

// Close releases the underlying WebSocket client.
func (c *Client) Close() error {
	if c == nil || c.oc == nil {
		return nil
	}
	return c.oc.Close()
}

func (c *Client) maybeEnsureSandbox(ctx context.Context) error {
	if c == nil || !c.autoStart {
		return nil
	}
	return EnsureSandboxRunning(ctx, c.bin, c.sandboxName)
}

// DispatchTaskWithSession implements the pool contract (OpenClaw-class session key).
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	if c == nil || c.oc == nil {
		return "", domain.ErrNoDispatchTarget
	}
	if err := c.maybeEnsureSandbox(ctx); err != nil {
		return "", err
	}
	return c.oc.DispatchTaskWithSession(ctx, task, sessionKey)
}

// DispatchSubtaskWithSession implements the pool contract for convoy subtasks.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	if c == nil || c.oc == nil {
		return "", domain.ErrNoDispatchTarget
	}
	if err := c.maybeEnsureSandbox(ctx); err != nil {
		return "", err
	}
	return c.oc.DispatchSubtaskWithSession(ctx, parent, sub, sessionKey)
}
