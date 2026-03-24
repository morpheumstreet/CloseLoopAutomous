package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/nemoclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/gateway/openclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// RoutingGateway implements [ports.AgentGateway] by resolving execution agent → gateway endpoint → transport.
type RoutingGateway struct {
	stub     *SimulationMockClaw
	pool     *clientPool
	resolver *TargetResolver
}

var (
	_ ports.AgentGateway             = (*RoutingGateway)(nil)
	_ ports.RemoteAgentProfileSource = (*RoutingGateway)(nil)
)

func isPooledRemoteDriver(d string) bool {
	switch d {
	case domain.GatewayDriverOpenClawWS,
		domain.GatewayDriverNemoClawWS,
		domain.GatewayDriverNullClawWS,
		domain.GatewayDriverNullClawA2A,
		domain.GatewayDriverPicoClawWS,
		domain.GatewayDriverZeroClawWS,
		domain.GatewayDriverClawletWS,
		domain.GatewayDriverIronClawWS,
		domain.GatewayDriverMimiClawWS,
		domain.GatewayDriverNanobotCLI,
		domain.GatewayDriverInkOSCLI,
		domain.GatewayDriverZClawRelayHTTP,
		domain.GatewayDriverMisterMorphHTTP,
		domain.GatewayDriverCoPawHTTP,
		domain.GatewayDriverMetaClawHTTP:
		return true
	default:
		return false
	}
}

// NewRoutingGateway wires multi-endpoint dispatch. defaultTimeout applies when a gateway_endpoints row has timeout_sec = 0.
func NewRoutingGateway(
	endpoints ports.GatewayEndpointRegistry,
	agents ports.ExecutionAgentRegistry,
	knowledge func(context.Context, domain.ProductID, string) (string, error),
	defaultTimeout time.Duration,
	nemo nemoclaw.PoolSettings,
	openclawConnect openclaw.ConnectEnv,
) (*RoutingGateway, func()) {
	stub := &SimulationMockClaw{}
	pool := newClientPool(knowledge, defaultTimeout, nemo, openclawConnect)
	resolver := &TargetResolver{Agents: agents, Endpoints: endpoints, DefaultTimeout: defaultTimeout}
	rg := &RoutingGateway{stub: stub, pool: pool, resolver: resolver}
	return rg, func() { pool.close() }
}

// DispatchTask implements [ports.AgentGateway].
func (r *RoutingGateway) DispatchTask(ctx context.Context, task domain.Task) (string, error) {
	if r == nil || r.resolver == nil {
		return "", domain.ErrNoDispatchTarget
	}
	target, err := r.resolver.Resolve(ctx, &task)
	if err != nil {
		return "", err
	}
	switch target.Driver {
	case domain.GatewayDriverStub:
		return r.stub.DispatchTask(ctx, task)
	default:
		if isPooledRemoteDriver(target.Driver) {
			if strings.TrimSpace(target.GatewayURL) == "" && target.Driver != domain.GatewayDriverNanobotCLI && target.Driver != domain.GatewayDriverInkOSCLI {
				return "", fmt.Errorf("%w: gateway_url required for driver %s", domain.ErrInvalidInput, target.Driver)
			}
			return r.pool.dispatchTask(ctx, target, task)
		}
		return "", fmt.Errorf("%w: unsupported gateway driver %q", domain.ErrInvalidInput, target.Driver)
	}
}

// DispatchSubtask implements [ports.AgentGateway].
func (r *RoutingGateway) DispatchSubtask(ctx context.Context, parent domain.Task, sub domain.Subtask) (string, error) {
	if r == nil || r.resolver == nil {
		return "", domain.ErrNoDispatchTarget
	}
	target, err := r.resolver.Resolve(ctx, &parent)
	if err != nil {
		return "", err
	}
	switch target.Driver {
	case domain.GatewayDriverStub:
		return r.stub.DispatchSubtask(ctx, parent, sub)
	default:
		if isPooledRemoteDriver(target.Driver) {
			if strings.TrimSpace(target.GatewayURL) == "" && target.Driver != domain.GatewayDriverNanobotCLI && target.Driver != domain.GatewayDriverInkOSCLI {
				return "", fmt.Errorf("%w: gateway_url required for driver %s", domain.ErrInvalidInput, target.Driver)
			}
			return r.pool.dispatchSubtask(ctx, target, parent, sub)
		}
		return "", fmt.Errorf("%w: unsupported gateway driver %q", domain.ErrInvalidInput, target.Driver)
	}
}

// ListRemoteProfiles implements [ports.RemoteAgentProfileSource] (OpenClaw-class WebSocket: agents.list RPC).
func (r *RoutingGateway) ListRemoteProfiles(ctx context.Context, ep *domain.GatewayEndpoint) ([]domain.AgentIdentity, error) {
	if r == nil || ep == nil {
		return nil, nil
	}
	drv := domain.NormalizeGatewayDriver(ep.Driver)
	now := time.Now().UTC()
	switch drv {
	case domain.GatewayDriverStub:
		return stubRemoteFleet(ep, now), nil
	case domain.GatewayDriverOpenClawWS, domain.GatewayDriverNullClawWS,
		domain.GatewayDriverZeroClawWS, domain.GatewayDriverClawletWS, domain.GatewayDriverIronClawWS:
		list, err := r.pool.listOpenClawRemoteAgents(ctx, ep)
		if err != nil {
			return nil, err
		}
		for i := range list {
			list[i].Driver = drv
			if strings.TrimSpace(list[i].GatewayURL) == "" {
				list[i].GatewayURL = ep.GatewayURL
			}
		}
		return list, nil
	default:
		return nil, domain.ErrRemoteAgentListUnsupported
	}
}

func stubRemoteFleet(ep *domain.GatewayEndpoint, now time.Time) []domain.AgentIdentity {
	id := domain.FleetProfileID(ep.ID, "stub-local")
	return []domain.AgentIdentity{{
		ID:         id,
		GatewayURL: "",
		Name:       "Stub (local)",
		Driver:     domain.GatewayDriverStub,
		Version:    "1.0",
		Status:     domain.StatusOnline,
		LastSeen:   now,
		Custom: map[string]any{
			"gateway_endpoint_id":   ep.ID,
			"remote_agent_id":       "stub-local",
			"suggested_session_key": "",
			"discovery_kind":        "stub",
			"on_registry":           false,
		},
	}}
}
