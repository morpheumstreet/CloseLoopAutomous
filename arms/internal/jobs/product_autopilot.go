package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hibiken/asynq"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/autopilot"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

const (
	// uniqueTTL must cover the longest expected gap between chained product ticks so duplicates are suppressed while one is pending.
	productAutopilotUniqueTTL = 48 * time.Hour
)

type productAutopilotPayload struct {
	ProductID string `json:"product_id"`
}

// ProductAutopilotTaskID is the stable Asynq task id per product (one pending chain per product).
func ProductAutopilotTaskID(productID domain.ProductID) string {
	return "arms:product_autopilot:" + string(productID)
}

// MarshalProductAutopilotPayload returns JSON for arms:product_autopilot_tick.
func MarshalProductAutopilotPayload(productID domain.ProductID) ([]byte, error) {
	return json.Marshal(productAutopilotPayload{ProductID: string(productID)})
}

// ParseProductAutopilotPayload extracts product id from task bytes.
func ParseProductAutopilotPayload(b []byte) (domain.ProductID, error) {
	var p productAutopilotPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return "", err
	}
	if p.ProductID == "" {
		return "", errors.New("missing product_id")
	}
	return domain.ProductID(p.ProductID), nil
}

// EnqueueProductAutopilotTick schedules a cadence check for one product. processIn==0 means run as soon as a worker picks it up.
func EnqueueProductAutopilotTick(client *asynq.Client, productID domain.ProductID, processIn time.Duration) error {
	payload, err := MarshalProductAutopilotPayload(productID)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TaskAutopilotProductTick, payload)
	opts := []asynq.Option{
		asynq.Queue(QueueName),
		asynq.TaskID(ProductAutopilotTaskID(productID)),
		asynq.Unique(productAutopilotUniqueTTL),
	}
	if processIn > 0 {
		opts = append(opts, asynq.ProcessIn(processIn))
	}
	_, err = client.Enqueue(task, opts...)
	return err
}

// ReconcileProductAutopilotTasks ensures every eligible product has a pending or scheduled product-autopilot task.
func ReconcileProductAutopilotTasks(ctx context.Context, client *asynq.Client, auto *autopilot.Service, now time.Time) error {
	if client == nil || auto == nil {
		return nil
	}
	list, err := auto.Products.ListAll(ctx)
	if err != nil {
		return err
	}
	for i := range list {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		pid := list[i].ID
		delay, keep, err := auto.NextAutopilotEnqueueDelay(ctx, pid, now)
		if err != nil || !keep {
			continue
		}
		err = EnqueueProductAutopilotTick(client, pid, delay)
		if err != nil && !errors.Is(err, asynq.ErrDuplicateTask) {
			return err
		}
	}
	return nil
}

// RunProductAutopilotTask executes one product tick and enqueues the follow-up delay (Asynq worker entrypoint helper).
func RunProductAutopilotTask(ctx context.Context, client *asynq.Client, auto *autopilot.Service, productID domain.ProductID, now time.Time) error {
	if err := auto.TickProduct(ctx, productID, now); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	if client == nil {
		return nil
	}
	delay, keep, err := auto.NextAutopilotEnqueueDelay(ctx, productID, now)
	if err != nil || !keep {
		return nil
	}
	err = EnqueueProductAutopilotTick(client, productID, delay)
	if err != nil && !errors.Is(err, asynq.ErrDuplicateTask) {
		return err
	}
	return nil
}
