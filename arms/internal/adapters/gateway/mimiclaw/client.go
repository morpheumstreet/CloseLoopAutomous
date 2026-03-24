// Package mimiclaw implements dispatch to [MimiClaw]'s LAN WebSocket gateway (port 18789, URI /).
// Wire format per docs/ARCHITECTURE.md: client sends {"type":"message","content":"...","chat_id":"..."};
// the device uses chat_id as the session key (max 31 chars stored in firmware).
//
// [MimiClaw]: https://github.com/memovai/mimiclaw
package mimiclaw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

const (
	typeMessage = "message"
	// maxChatIDRunes matches MimiClaw ws_client_t.chat_id[32] (31 chars + NUL).
	maxChatIDRunes = 31
)

type wireOutbound struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	ChatID  string `json:"chat_id"`
}

// Options configure the MimiClaw WebSocket client.
type Options struct {
	URL    string
	Token  string // optional; sent as Authorization: Bearer (ignored by stock firmware if unset)
	Timeout time.Duration
	// KnowledgeForDispatch appends ranked snippets to dispatch bodies when non-nil (same hook as OpenClaw).
	KnowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error)
}

// Client maintains one WebSocket and a background reader (drains server frames).
type Client struct {
	opts       Options
	mu         sync.Mutex
	conn       *websocket.Conn
	readCancel context.CancelFunc
	readDone   sync.WaitGroup
}

// New constructs a Client. Empty URL yields an error on first dispatch.
func New(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &Client{opts: opts}
}

// Close shuts down the reader and connection.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dropConnLocked()
}

func (c *Client) dropConnLocked() error {
	if c.readCancel != nil {
		c.readCancel()
		c.readCancel = nil
	}
	c.readDone.Wait()
	var err error
	if c.conn != nil {
		err = c.conn.Close(websocket.StatusNormalClosure, "arms close")
		c.conn = nil
	}
	return err
}

func (c *Client) callContext(parent context.Context) (context.Context, context.CancelFunc) {
	if c.opts.Timeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, c.opts.Timeout)
}

func (c *Client) ensureConnLocked(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}
	u := strings.TrimSpace(c.opts.URL)
	if u == "" {
		return errors.New("mimiclaw: gateway URL is required")
	}
	hdr := http.Header{}
	if tok := strings.TrimSpace(c.opts.Token); tok != "" {
		hdr.Set("Authorization", "Bearer "+tok)
	}
	conn, _, err := websocket.Dial(ctx, u, &websocket.DialOptions{HTTPHeader: hdr})
	if err != nil {
		return fmt.Errorf("mimiclaw dial: %w", err)
	}
	c.conn = conn
	rctx, cancel := context.WithCancel(context.Background())
	c.readCancel = cancel
	c.readDone.Add(1)
	go func() {
		defer c.readDone.Done()
		c.pumpReads(rctx, conn)
	}()
	return nil
}

func (c *Client) pumpReads(ctx context.Context, conn *websocket.Conn) {
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			return
		}
	}
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

func truncateChatID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) > maxChatIDRunes {
		return string(r[:maxChatIDRunes])
	}
	return s
}

// DispatchTaskWithSession sends a MimiClaw message frame; sessionKey is mapped to chat_id.
func (c *Client) DispatchTaskWithSession(ctx context.Context, task domain.Task, sessionKey string) (string, error) {
	cid := truncateChatID(sessionKey)
	if cid == "" {
		return "", errors.New("mimiclaw: session_key (chat_id) is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	if err := c.ensureConnLocked(callCtx); err != nil {
		return "", err
	}
	kb := c.knowledgeMarkdown(callCtx, task.ProductID, knowledgeQueryFromTask(task))
	body := openclaw.TaskDispatchMarkdown(task, kb)
	msg := wireOutbound{Type: typeMessage, Content: body, ChatID: cid}
	b, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	if err := c.conn.Write(callCtx, websocket.MessageText, b); err != nil {
		_ = c.dropConnLocked()
		return "", fmt.Errorf("mimiclaw write: %w", err)
	}
	return uuid.NewString(), nil
}

// DispatchSubtaskWithSession sends a message frame for a convoy subtask.
func (c *Client) DispatchSubtaskWithSession(ctx context.Context, parent domain.Task, sub domain.Subtask, sessionKey string) (string, error) {
	cid := truncateChatID(sessionKey)
	if cid == "" {
		return "", errors.New("mimiclaw: session_key (chat_id) is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	if err := c.ensureConnLocked(callCtx); err != nil {
		return "", err
	}
	kb := c.knowledgeMarkdown(callCtx, parent.ProductID, knowledgeQueryFromSubtask(parent, sub))
	body := openclaw.SubtaskDispatchMarkdown(parent.ID, sub, kb)
	msg := wireOutbound{Type: typeMessage, Content: body, ChatID: cid}
	b, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	if err := c.conn.Write(callCtx, websocket.MessageText, b); err != nil {
		_ = c.dropConnLocked()
		return "", fmt.Errorf("mimiclaw write: %w", err)
	}
	return uuid.NewString(), nil
}
