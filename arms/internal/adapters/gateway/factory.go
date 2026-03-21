package gateway

import (
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/gateway/openclaw"
	"github.com/closeloopautomous/arms/internal/ports"
)

// NewAgentGateway returns a stub when url is empty; otherwise a native OpenClaw WebSocket client
// (Mission Control–compatible handshake + chat.send).
// cleanup should be invoked on process shutdown (e.g. App.Close).
func NewAgentGateway(url, token, deviceID, sessionKey string, dispatchTimeout time.Duration) (ports.AgentGateway, func()) {
	url = strings.TrimSpace(url)
	if url == "" {
		s := &Stub{}
		return s, func() {}
	}
	c := openclaw.New(openclaw.Options{
		URL:        url,
		Token:      token,
		DeviceID:   deviceID,
		SessionKey: sessionKey,
		Timeout:    dispatchTimeout,
	})
	return c, func() { _ = c.Close() }
}
