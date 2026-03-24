// Package zeroclaw integrates arms with ZeroClaw (https://github.com/zeroclaw-labs/zeroclaw).
//
// The gateway control plane uses the same WebSocket JSON-RPC sequence as OpenClaw-class
// runtimes (connect.challenge → connect, chat.send). [Client] wraps [openclaw.Client] so
// ZeroClaw-specific behavior can be added without forking the wire implementation.
package zeroclaw

import (
	"context"
	"errors"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Options configure the ZeroClaw gateway WebSocket client.
type Options struct {
	URL      string
	Token    string
	DeviceID string
	Timeout  time.Duration
	MinProto int
	MaxProto int
	// KnowledgeForDispatch appends ranked snippets to dispatch bodies when non-nil (same hook as OpenClaw).
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
}

func (o Options) toOpenClaw() openclaw.Options {
	return openclaw.Options{
		URL:                  o.URL,
		Token:                o.Token,
		DeviceID:             o.DeviceID,
		SessionKey:           "",
		Timeout:              o.Timeout,
		MinProto:             o.MinProto,
		MaxProto:             o.MaxProto,
		KnowledgeForDispatch: o.KnowledgeForDispatch,
	}
}

// Client dispatches tasks to a ZeroClaw gateway via the OpenClaw-compatible WebSocket protocol.
type Client struct {
	oc *openclaw.Client
}

// New constructs a Client. Connection is established on first dispatch.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &Client{oc: openclaw.New(opts.toOpenClaw())}
}

// NewFromOpenClawOptions builds a Client from [openclaw.Options] (interop / tests).
func NewFromOpenClawOptions(o openclaw.Options) *Client {
	return &Client{oc: openclaw.New(o)}
}

// Close drops the cached connection.
func (c *Client) Close() error {
	if c == nil || c.oc == nil {
		return nil
	}
	return c.oc.Close()
}

// DispatchTaskWithSession sends chat.send with the given sessionKey.
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	if c == nil || c.oc == nil {
		return "", errors.New("zeroclaw: client is nil")
	}
	return c.oc.DispatchTaskWithSession(ctx, task, sessionKey)
}

// DispatchSubtaskWithSession sends chat.send for a convoy subtask with the given sessionKey.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	if c == nil || c.oc == nil {
		return "", errors.New("zeroclaw: client is nil")
	}
	return c.oc.DispatchSubtaskWithSession(ctx, parent, sub, sessionKey)
}
