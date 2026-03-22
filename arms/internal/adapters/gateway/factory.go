package gateway

import (
	"context"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/gateway/openclaw"
	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// NewAgentGateway returns a stub when url is empty; otherwise a native OpenClaw WebSocket client
// (Mission Control–compatible handshake + chat.send).
// cleanup should be invoked on process shutdown (e.g. App.Close).
// knowledgeForDispatch is optional (#90); nil skips retrieval/injection.
func NewAgentGateway(url, token, deviceID, sessionKey string, dispatchTimeout time.Duration,
	knowledgeForDispatch func(ctx context.Context, productID domain.ProductID, query string) (string, error),
) (ports.AgentGateway, func()) {
	url = strings.TrimSpace(url)
	if url == "" {
		s := &Stub{}
		return s, func() {}
	}
	c := openclaw.New(openclaw.Options{
		URL:                  url,
		Token:                token,
		DeviceID:             deviceID,
		SessionKey:           sessionKey,
		Timeout:              dispatchTimeout,
		KnowledgeForDispatch: knowledgeForDispatch,
	})
	return c, func() { _ = c.Close() }
}
