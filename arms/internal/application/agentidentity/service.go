package agentidentity

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service synthesizes and stores [domain.AgentIdentity] rows from gateway endpoints (MVP: registry + optional GeoIP).
type Service struct {
	Endpoints ports.GatewayEndpointRegistry
	Profiles  ports.AgentProfileRepository
	Geo       ports.GeoIPResolver
	Events    ports.LiveActivityPublisher
	Clock     func() time.Time
}

// RefreshAll rebuilds identities for every gateway endpoint and upserts profiles.
func (s *Service) RefreshAll(ctx context.Context) error {
	if s == nil || s.Endpoints == nil || s.Profiles == nil {
		return nil
	}
	list, err := s.Endpoints.List(ctx, 500)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if s.Clock != nil {
		now = s.Clock().UTC()
	}
	for i := range list {
		ep := &list[i]
		ident := synthesize(ep, s.Geo, now)
		pctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		enrichReachability(pctx, ep, ident)
		cancel()
		if err := s.Profiles.Upsert(ctx, ep.ID, ident); err != nil {
			return err
		}
		if s.Events != nil {
			_ = s.Events.Publish(ctx, ports.LiveActivityEvent{
				Type: "agent_identity_updated",
				Ts:   now.Format(time.RFC3339Nano),
				Data: map[string]any{
					"identity_id": ident.ID,
					"gateway_id":  ep.ID,
					"driver":      ep.Driver,
				},
			})
		}
	}
	return nil
}

func synthesize(ep *domain.GatewayEndpoint, geo ports.GeoIPResolver, now time.Time) *domain.AgentIdentity {
	id := domain.StableAgentProfileID(ep.ID, ep.DeviceID)
	name := strings.TrimSpace(ep.DisplayName)
	if name == "" {
		name = ep.ID
	}
	st := domain.StatusOffline
	if ep.Driver == domain.GatewayDriverStub {
		st = domain.StatusOnline
	}
	ident := &domain.AgentIdentity{
		ID:         id,
		GatewayURL: ep.GatewayURL,
		Name:       name,
		Driver:     ep.Driver,
		Status:     st,
		LastSeen:   now,
		Platform:   domain.PlatformInfo{},
		Metrics:    domain.Metrics{},
		Custom: map[string]any{
			"gateway_endpoint_id": ep.ID,
			"device_id":           strings.TrimSpace(ep.DeviceID),
		},
	}
	if geo != nil && strings.TrimSpace(ep.GatewayURL) != "" {
		host := hostFromGatewayURL(ep.GatewayURL)
		if host != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			g, err := geo.LookupHost(ctx, host)
			cancel()
			if err != nil {
				slog.Default().Debug("agentidentity geo lookup", "host", host, "err", err)
			}
			if g != nil && g.Source != "" && g.Source != "none" {
				ident.Geo = g
			}
		}
	}
	return ident
}

func hostFromGatewayURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	return host
}
