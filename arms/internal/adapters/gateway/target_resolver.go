package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// TargetResolver maps a task's bound execution agent to a concrete [domain.DispatchTarget].
type TargetResolver struct {
	Agents         ports.ExecutionAgentRegistry
	Endpoints      ports.GatewayEndpointRegistry
	DefaultTimeout time.Duration
}

// Resolve returns the dispatch target for the task's CurrentExecutionAgentID.
func (r *TargetResolver) Resolve(ctx context.Context, task *domain.Task) (domain.DispatchTarget, error) {
	if r == nil || r.Agents == nil || r.Endpoints == nil {
		return domain.DispatchTarget{}, domain.ErrNoDispatchTarget
	}
	aid := strings.TrimSpace(task.CurrentExecutionAgentID)
	if aid == "" {
		return domain.DispatchTarget{}, domain.ErrNoDispatchTarget
	}
	ag, err := r.Agents.ByID(ctx, aid)
	if err != nil {
		return domain.DispatchTarget{}, err
	}
	eid := strings.TrimSpace(ag.EndpointID)
	if eid == "" {
		return domain.DispatchTarget{}, fmt.Errorf("%w: execution agent %q has no gateway_endpoint_id", domain.ErrInvalidInput, aid)
	}
	ep, err := r.Endpoints.ByID(ctx, eid)
	if err != nil {
		return domain.DispatchTarget{}, err
	}
	drv := domain.NormalizeGatewayDriver(ep.Driver)
	if drv == "" {
		return domain.DispatchTarget{}, fmt.Errorf("%w: unknown gateway driver %q on endpoint %q", domain.ErrInvalidInput, ep.Driver, ep.ID)
	}
	sk := strings.TrimSpace(ag.SessionKey)
	if drv != domain.GatewayDriverStub && sk == "" {
		return domain.DispatchTarget{}, fmt.Errorf("%w: execution agent %q needs session_key for driver %s", domain.ErrInvalidInput, aid, drv)
	}
	to := r.DefaultTimeout
	if ep.TimeoutSec > 0 {
		to = time.Duration(ep.TimeoutSec) * time.Second
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	return domain.DispatchTarget{
		Driver:       drv,
		GatewayURL:   ep.GatewayURL,
		GatewayToken: ep.GatewayToken,
		DeviceID:     ep.DeviceID,
		SessionKey:   sk,
		Timeout:      to,
	}, nil
}
