package domain

import "time"

// ProductSchedule gates whether autopilot cadence ticks apply to a product (Mission Control–style product_schedules row).
// No row in the store means “enabled” for backward compatibility.
// When Redis + Asynq are configured, optional cron_expr / delay_seconds enqueue per-product ticks (see internal/jobs).
type ProductSchedule struct {
	ProductID       ProductID
	Enabled         bool
	SpecJSON        string
	CronExpr        string
	DelaySeconds    int
	AsynqTaskID     string
	LastEnqueuedAt  *time.Time
	NextScheduledAt *time.Time
	UpdatedAt       time.Time
}
