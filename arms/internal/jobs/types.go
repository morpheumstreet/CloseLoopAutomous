// Package jobs holds shared Asynq task type names and queue name for cmd/arms and cmd/arms-worker.
package jobs

const (
	// QueueName is the default Asynq queue for all arms tasks.
	QueueName = "arms"

	TaskAutopilotGlobalTick  = "arms:autopilot_tick"
	TaskAutopilotProductTick = "arms:product_autopilot_tick"
	TaskProductScheduleTick  = "product:schedule:tick"
	TaskStallAutoNudgeTick   = "arms:stall_autonudge_tick"
)
