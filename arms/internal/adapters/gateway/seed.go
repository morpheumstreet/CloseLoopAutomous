package gateway

import (
	"context"
	"errors"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// EnsureDefaultStubEndpoint inserts the built-in stub profile (id gw-stub) when missing.
func EnsureDefaultStubEndpoint(ctx context.Context, reg ports.GatewayEndpointRegistry, clock ports.Clock) error {
	if reg == nil {
		return nil
	}
	_, err := reg.ByID(ctx, "gw-stub")
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	at := clock.Now()
	return reg.Save(ctx, &domain.GatewayEndpoint{
		ID:          "gw-stub",
		DisplayName: "Default stub",
		Driver:      domain.GatewayDriverStub,
		CreatedAt:   at.UTC(),
	})
}
