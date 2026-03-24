// Package inkos dispatches tasks by running [Narcooo/inkos] `inkos write next … --json` in the configured project workspace.
//
// [Narcooo/inkos]: https://github.com/Narcooo/inkos
package inkos

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Client runs InkOS for each dispatch.
type Client struct {
	opts Options
}

// New constructs a Client. Zero Timeout defaults to 30s before dispatch clamps with context timeout.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &Client{opts: opts}
}

// Close is a no-op; satisfies pooling alongside WebSocket clients.
func (*Client) Close() error { return nil }

func (c *Client) inkosExecutable() string {
	b := strings.TrimSpace(c.opts.InkOSBin)
	if b == "" {
		return "inkos"
	}
	return b
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

func (c *Client) callContext(parent context.Context) (context.Context, context.CancelFunc) {
	if c.opts.Timeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, c.opts.Timeout)
}

func (c *Client) runWriteNext(ctx context.Context, contextText string) ([]byte, error) {
	bookID := strings.TrimSpace(c.opts.BookID)
	if bookID == "" {
		return nil, errors.New("inkos_cli: device_id (InkOS book id) is required on the gateway endpoint")
	}
	bin := c.inkosExecutable()
	ws := ExpandPath(c.opts.Workspace)
	args := BuildWriteNextArgs(bookID, contextText)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(),
		"NO_COLOR=1",
		"TERM=dumb",
		"CI=1",
	)
	if ws != "" {
		cmd.Dir = ws
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		preview := strings.TrimSpace(string(out))
		if len(preview) > 500 {
			preview = preview[:500] + "…"
		}
		if preview != "" {
			return out, fmt.Errorf("inkos_cli: %w: %s", err, preview)
		}
		return out, fmt.Errorf("inkos_cli: %w", err)
	}
	return out, nil
}

func (c *Client) externalRef(bookID string, stdout []byte) string {
	if seg := ExternalRefFromInkOSStdout(stdout); seg != "" {
		return fmt.Sprintf("inkos-cli:%s:%s", bookID, seg)
	}
	return fmt.Sprintf("inkos-cli:%s:%d", bookID, time.Now().UnixNano())
}

// DispatchTaskWithSession runs `inkos write next` for the task. sessionKey is required by arms' resolver but is not passed to InkOS.
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, _ string) (string, error) {
	bookID := strings.TrimSpace(c.opts.BookID)
	if bookID == "" {
		return "", errors.New("inkos_cli: device_id (InkOS book id) is required on the gateway endpoint")
	}
	kb := c.knowledgeMarkdown(ctx, task.ProductID, openclaw.KnowledgeQueryFromTask(task))
	msg := openclaw.TaskDispatchMarkdown(task, kb)
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	out, err := c.runWriteNext(callCtx, msg)
	if err != nil {
		return "", err
	}
	return c.externalRef(bookID, out), nil
}

// DispatchSubtaskWithSession runs `inkos write next` with convoy subtask guidance as --context.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, _ string) (string, error) {
	bookID := strings.TrimSpace(c.opts.BookID)
	if bookID == "" {
		return "", errors.New("inkos_cli: device_id (InkOS book id) is required on the gateway endpoint")
	}
	kb := c.knowledgeMarkdown(ctx, parent.ProductID, openclaw.KnowledgeQueryFromSubtask(parent, sub))
	msg := openclaw.SubtaskDispatchMarkdown(parent.ID, sub, kb)
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	out, err := c.runWriteNext(callCtx, msg)
	if err != nil {
		return "", err
	}
	return c.externalRef(bookID, out), nil
}
