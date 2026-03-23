package ports

import (
	"context"

	"github.com/closeloopautomous/arms/internal/domain"
)

// GatewayEndpointRegistry persists OpenClaw-class gateway connection profiles (multi-endpoint dispatch).
type GatewayEndpointRegistry interface {
	Save(ctx context.Context, e *domain.GatewayEndpoint) error
	ByID(ctx context.Context, id string) (*domain.GatewayEndpoint, error)
	List(ctx context.Context, limit int) ([]domain.GatewayEndpoint, error)
}
