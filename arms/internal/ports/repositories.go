package ports

import (
	"context"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

type ProductRepository interface {
	Save(ctx context.Context, p *domain.Product) error
	ByID(ctx context.Context, id domain.ProductID) (*domain.Product, error)
	ListAll(ctx context.Context) ([]domain.Product, error)
}

// MaybePoolRepository tracks ideas swiped "maybe" for later promotion (Mission Control–style).
type MaybePoolRepository interface {
	Add(ctx context.Context, ideaID domain.IdeaID, productID domain.ProductID, at time.Time) error
	Remove(ctx context.Context, ideaID domain.IdeaID) error
	ListIdeaIDsByProduct(ctx context.Context, productID domain.ProductID) ([]domain.IdeaID, error)
}

type IdeaRepository interface {
	Save(ctx context.Context, i *domain.Idea) error
	ByID(ctx context.Context, id domain.IdeaID) (*domain.Idea, error)
	ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Idea, error)
}

type TaskRepository interface {
	Save(ctx context.Context, t *domain.Task) error
	ByID(ctx context.Context, id domain.TaskID) (*domain.Task, error)
	ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Task, error)
}

type ConvoyRepository interface {
	Save(ctx context.Context, c *domain.Convoy) error
	ByID(ctx context.Context, id domain.ConvoyID) (*domain.Convoy, error)
	ListByProduct(ctx context.Context, productID domain.ProductID) ([]domain.Convoy, error)
}

type CostRepository interface {
	Append(ctx context.Context, e domain.CostEvent) error
	SumByProduct(ctx context.Context, productID domain.ProductID) (float64, error)
}

type CheckpointRepository interface {
	Save(ctx context.Context, taskID domain.TaskID, payload string) error
	Load(ctx context.Context, taskID domain.TaskID) (string, error)
}
