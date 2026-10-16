package quota

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CycleType constants.
const (
	CycleMonthlyOnly   = "monthly_only"
	CycleWeeklyMonthly = "weekly_monthly"

	CycleTypeWeekly  = "weekly"
	CycleTypeMonthly = "monthly"
)

// DegradedMode constants.
const (
	DegradedHardBlock      = "hard_block"
	DegradedThrottle       = "throttle"
	DegradedModelDowngrade = "model_downgrade"
)

// Status constants.
const (
	StatusNormal            = "normal"
	StatusWarning           = "warning"
	StatusDegradedThrottle  = "degraded_throttle"
	StatusDegradedDowngrade = "degraded_downgrade"
	StatusBlocked           = "blocked"
)

// BlockReason constants.
const (
	BlockReasonWeeklyExhausted  = "weekly_exhausted"
	BlockReasonMonthlyExhausted = "monthly_exhausted"
	BlockReasonDegradedCap      = "degraded_cap"
)

// PlanLimitCycle defines cycle settings per plan and limit_key.
type PlanLimitCycle struct {
	ID                     uuid.UUID `json:"id"`
	PlanID                 uuid.UUID `json:"plan_id"`
	LimitKey               string    `json:"limit_key"`
	CycleType              string    `json:"cycle_type"` // "monthly_only" | "weekly_monthly"
	WeeklyLimit            *int64    `json:"weekly_limit,omitempty"`
	DegradedMode           *string   `json:"degraded_mode,omitempty"`
	ThrottleRateAmount     *int      `json:"throttle_rate_amount,omitempty"`
	ThrottleRateWindowSecs *int      `json:"throttle_rate_window_secs,omitempty"`
	FallbackModel          *string   `json:"fallback_model,omitempty"`
	MaxDegradedUsage       *int64    `json:"max_degraded_usage,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// UsageCounter records usage per window.
type UsageCounter struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	LimitKey     string    `json:"limit_key"`
	CycleType    string    `json:"cycle_type"` // "weekly" | "monthly"
	CycleStart   time.Time `json:"cycle_start"`
	CycleEnd     time.Time `json:"cycle_end"`
	Used         int64     `json:"used"`
	DegradedUsed int64     `json:"degraded_used"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// QuotaNotificationConfig holds warning threshold per plan.
type QuotaNotificationConfig struct {
	ID               uuid.UUID  `json:"id"`
	PlanID           *uuid.UUID `json:"plan_id,omitempty"`
	WarningThreshold int        `json:"warning_threshold"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// EvalResult is returned by DualCycleEnforcer.Evaluate.
type EvalResult struct {
	Allowed          bool      `json:"allowed"`
	Degraded         bool      `json:"degraded"`
	DegradedMode     string    `json:"degraded_mode,omitempty"`
	FallbackModel    string    `json:"fallback_model,omitempty"`
	WeeklyUsed       int64     `json:"weekly_used"`
	WeeklyLimit      int64     `json:"weekly_limit"`
	WeeklyRemaining  int64     `json:"weekly_remaining"`
	MonthlyUsed      int64     `json:"monthly_used"`
	MonthlyLimit     int64     `json:"monthly_limit"`
	MonthlyRemaining int64     `json:"monthly_remaining"`
	Effective        int64     `json:"effective"`
	ResetAt          time.Time `json:"reset_at"`
	BlockReason      string    `json:"block_reason,omitempty"`
	Status           string    `json:"status"`
}

// CycleStore resolves cycle configuration and notification thresholds.
type CycleStore interface {
	GetPlanCycle(ctx context.Context, planID uuid.UUID, limitKey string) (*PlanLimitCycle, error)
	GetWarningThreshold(ctx context.Context, planID uuid.UUID) (int, error)
}

// UsageCounterStore manages window counters.
type UsageCounterStore interface {
	GetOrCreateCounter(ctx context.Context, userID uuid.UUID, limitKey, cycleType string, start, end time.Time) (*UsageCounter, error)
	IncrementUsage(ctx context.Context, counterID uuid.UUID, delta int64, hardLimit int64) error
	IncrementDegradedUsage(ctx context.Context, counterID uuid.UUID, delta int64, maxDegraded int64) error
	GetCurrentCounters(ctx context.Context, userID uuid.UUID, now time.Time) ([]UsageCounter, error)
}
