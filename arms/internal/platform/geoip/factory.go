package geoip

import (
	"log/slog"
	"strings"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// NewResolver returns a GeoIP resolver from optional MMDB path (ARMS_GEOIP2_CITY). On open failure, logs and returns NoopResolver.
func NewResolver(mmdbPath string) (ports.GeoIPResolver, func()) {
	p := strings.TrimSpace(mmdbPath)
	if p == "" {
		return NoopResolver{}, func() {}
	}
	r, err := OpenMaxMind(p)
	if err != nil {
		slog.Default().Warn("arms geoip: MaxMind DB not loaded", "path", p, "err", err)
		return NoopResolver{}, func() {}
	}
	return r, func() { _ = r.Close() }
}
