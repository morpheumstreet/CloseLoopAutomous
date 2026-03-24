// Package ironclaw integrates arms with IronClaw (OpenClaw-class, Rust-native agent gateway).
//
// IronClaw exposes the same WebSocket JSON-RPC flow as stock OpenClaw (connect.challenge → connect,
// chat.send). This client wraps the shared [openclaw.Client]. Use driver ironclaw_ws on a gateway
// endpoint whose gateway_url points at the IronClaw web gateway WebSocket; optional NEAR or other
// bearer auth is supplied via gateway_token (Authorization header + URL token query, same as OpenClaw).
package ironclaw

import (
	"context"
	"errors"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Options configure the IronClaw gateway WebSocket client.
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

// Client dispatches tasks to IronClaw via the OpenClaw-compatible WebSocket protocol.
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
		return "", errors.New("ironclaw: client is nil")
	}
	return c.oc.DispatchTaskWithSession(ctx, task, sessionKey)
}

// DispatchSubtaskWithSession sends chat.send for a convoy subtask with the given sessionKey.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	if c == nil || c.oc == nil {
		return "", errors.New("ironclaw: client is nil")
	}
	return c.oc.DispatchSubtaskWithSession(ctx, parent, sub, sessionKey)
}
