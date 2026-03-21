// Package jobs holds shared Asynq task type names and queue names for cmd/arms and cmd/arms-worker.
package jobs

const (
	QueueDefault               = "arms"
	TypeAutopilotTick          = "arms:autopilot_tick"
	TypeProductAutopilotTick   = "arms:product_autopilot_tick"
)
