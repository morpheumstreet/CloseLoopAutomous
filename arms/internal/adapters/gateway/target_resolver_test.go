package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/memory"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestTargetResolver_ResolveStub(t *testing.T) {
	ctx := context.Background()
	eps := memory.NewGatewayEndpointStore()
	agents := memory.NewExecutionAgentStore()
	at := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	_ = eps.Save(ctx, &domain.GatewayEndpoint{ID: "gw-stub", Driver: domain.GatewayDriverStub, CreatedAt: at})
	_ = agents.Save(ctx, &domain.ExecutionAgent{
		ID: "ag1", DisplayName: "a", EndpointID: "gw-stub", SessionKey: "", CreatedAt: at,
	})
	r := &TargetResolver{Agents: agents, Endpoints: eps, DefaultTimeout: 5 * time.Second}
	task := &domain.Task{CurrentExecutionAgentID: "ag1"}
	got, err := r.Resolve(ctx, task)
	if err != nil {
		t.Fatal(err)
	}
	if got.Driver != domain.GatewayDriverStub || got.SessionKey != "" {
		t.Fatalf("stub target %+v", got)
	}
}

func TestTargetResolver_ResolveRequiresSessionForOpenClaw(t *testing.T) {
	ctx := context.Background()
	eps := memory.NewGatewayEndpointStore()
	agents := memory.NewExecutionAgentStore()
	at := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	_ = eps.Save(ctx, &domain.GatewayEndpoint{
		ID: "gw-oc", Driver: domain.GatewayDriverOpenClawWS, GatewayURL: "wss://x/ws", CreatedAt: at,
	})
	_ = agents.Save(ctx, &domain.ExecutionAgent{
		ID: "ag1", DisplayName: "a", EndpointID: "gw-oc", SessionKey: "", CreatedAt: at,
	})
	r := &TargetResolver{Agents: agents, Endpoints: eps, DefaultTimeout: 5 * time.Second}
	_, err := r.Resolve(ctx, &domain.Task{CurrentExecutionAgentID: "ag1"})
	if err == nil {
		t.Fatal("expected error for empty session_key")
	}
}
