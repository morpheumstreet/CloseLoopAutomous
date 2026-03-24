// Package clawlet integrates arms with Clawlet (https://github.com/mosaxiv/clawlet).
//
// Clawlet is an OpenClaw-inspired, local-first Go gateway. This client uses the same
// WebSocket JSON-RPC sequence as other OpenClaw-class runtimes (connect.challenge → connect,
// chat.send) via the shared [openclaw.Client] implementation. Use driver clawlet_ws on a
// gateway endpoint whose gateway_url points at a Clawlet (or compatible) control listener
// once that surface is available; until then the same wire format applies to any endpoint
// that implements it.
package clawlet

import (
	"context"
	"errors"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Options configure the Clawlet gateway WebSocket client.
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

// Client dispatches tasks to a Clawlet-class gateway via the OpenClaw-compatible WebSocket protocol.
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
		return "", errors.New("clawlet: client is nil")
	}
	return c.oc.DispatchTaskWithSession(ctx, task, sessionKey)
}

// DispatchSubtaskWithSession sends chat.send for a convoy subtask with the given sessionKey.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	if c == nil || c.oc == nil {
		return "", errors.New("clawlet: client is nil")
	}
	return c.oc.DispatchSubtaskWithSession(ctx, parent, sub, sessionKey)
}
