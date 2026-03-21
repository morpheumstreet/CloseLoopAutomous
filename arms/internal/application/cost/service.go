package cost

import (
	"context"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// Service records spend events for observability and budget inputs.
type Service struct {
	Costs ports.CostRepository
	Clock ports.Clock
	IDs   ports.IdentityGenerator
}

func (s *Service) Record(ctx context.Context, productID domain.ProductID, taskID domain.TaskID, amount float64, note string) error {
	e := domain.CostEvent{
		ID:        s.IDs.NewCostEventID(),
		ProductID: productID,
		TaskID:    taskID,
		Amount:    amount,
		Note:      note,
		At:        s.Clock.Now(),
	}
	return s.Costs.Append(ctx, e)
}
