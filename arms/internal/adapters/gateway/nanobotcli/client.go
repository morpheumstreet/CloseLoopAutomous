// Package nanobotcli dispatches tasks by running [HKUDS/nanobot] `nanobot agent -m …` as a subprocess.
// Nanobot's `gateway` command does not expose an OpenClaw-class WebSocket control plane; CLI one-shot mode is the stable integration surface.
//
// [HKUDS/nanobot]: https://github.com/HKUDS/nanobot
package nanobotcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/gateway/openclaw"
	"github.com/closeloopautomous/arms/internal/domain"
)

// Options configure the nanobot CLI subprocess client.
type Options struct {
	// NanobotBin is the executable name or path (from gateway_token when non-empty; otherwise "nanobot" on PATH).
	NanobotBin string
	// ConfigPath is passed as `nanobot agent -c` (from gateway_url when non-empty).
	ConfigPath string
	// Workspace is passed as `nanobot agent -w` (from device_id when non-empty).
	Workspace string
	Timeout time.Duration
	// KnowledgeForDispatch appends ranked snippets to dispatch bodies when non-nil (same hook as OpenClaw).
	KnowledgeForDispatch func(context.Context, domain.ProductID, string) (string, error)
}

// Client runs nanobot in one-shot CLI mode for each dispatch.
type Client struct {
	opts Options
}

// New constructs a Client. Zero Timeout defaults to 30s before dispatch uses its own clamp.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &Client{opts: opts}
}

// Close is a no-op; satisfies pooling alongside WebSocket clients.
func (*Client) Close() error { return nil }

func expandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = os.ExpandEnv(p)
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			rest := strings.TrimPrefix(p[2:], "/")
			p = filepath.Join(h, rest)
		}
	}
	return p
}

func buildAgentArgs(sessionKey, message, cfg, ws string) []string {
	args := []string{"agent", "-m", message, "--no-markdown", "-s", sessionKey}
	if cfg != "" {
		args = append(args, "-c", cfg)
	}
	if ws != "" {
		args = append(args, "-w", ws)
	}
	return args
}

func (c *Client) knowledgeMarkdown(ctx context.Context, pid domain.ProductID, q string) string {
	if c.opts.KnowledgeForDispatch == nil {
		return ""
	}
	s, err := c.opts.KnowledgeForDispatch(ctx, pid, q)
	if err != nil || strings.TrimSpace(s) == "" {
		return ""
	}
	return s
}

func (c *Client) nanobotExecutable() string {
	b := strings.TrimSpace(c.opts.NanobotBin)
	if b == "" {
		return "nanobot"
	}
	return b
}

func (c *Client) runAgent(ctx context.Context, sessionKey, message string) error {
	bin := c.nanobotExecutable()
	cfg := expandPath(c.opts.ConfigPath)
	ws := expandPath(c.opts.Workspace)
	args := buildAgentArgs(sessionKey, message, cfg, ws)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(),
		"NO_COLOR=1",
		"TERM=dumb",
		"PYTHONUNBUFFERED=1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		preview := strings.TrimSpace(string(out))
		if len(preview) > 500 {
			preview = preview[:500] + "…"
		}
		if preview != "" {
			return fmt.Errorf("nanobot_cli: %w: %s", err, preview)
		}
		return fmt.Errorf("nanobot_cli: %w", err)
	}
	return nil
}

func (c *Client) callContext(parent context.Context) (context.Context, context.CancelFunc) {
	if c.opts.Timeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, c.opts.Timeout)
}

// DispatchTaskWithSession runs `nanobot agent` with the given nanobot session id (-s).
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	sk := strings.TrimSpace(sessionKey)
	if sk == "" {
		return "", errors.New("nanobot_cli: session_key required (nanobot --session / -s)")
	}
	kb := c.knowledgeMarkdown(ctx, task.ProductID, openclaw.KnowledgeQueryFromTask(task))
	msg := openclaw.TaskDispatchMarkdown(task, kb)
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	if err := c.runAgent(callCtx, sk, msg); err != nil {
		return "", err
	}
	return fmt.Sprintf("nanobot-cli:%s:%d", task.ID, time.Now().UnixNano()), nil
}

// DispatchSubtaskWithSession runs `nanobot agent` for a convoy subtask body.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	sk := strings.TrimSpace(sessionKey)
	if sk == "" {
		return "", errors.New("nanobot_cli: session_key required (nanobot --session / -s)")
	}
	kb := c.knowledgeMarkdown(ctx, parent.ProductID, openclaw.KnowledgeQueryFromSubtask(parent, sub))
	msg := openclaw.SubtaskDispatchMarkdown(parent.ID, sub, kb)
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	if err := c.runAgent(callCtx, sk, msg); err != nil {
		return "", err
	}
	return fmt.Sprintf("nanobot-cli:%s:%s:%d", parent.ID, sub.ID, time.Now().UnixNano()), nil
}
