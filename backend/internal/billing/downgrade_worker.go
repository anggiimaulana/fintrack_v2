package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// ScheduledDowngradeLister lists users whose end-of-cycle downgrade is due.
// Implemented by the billing repository; kept narrow so the worker is
// unit-testable without a database.
type ScheduledDowngradeLister interface {
	ListDueScheduledDowngrades(ctx context.Context, now time.Time) ([]uuid.UUID, error)
}

// DowngradeWorker applies scheduled downgrades once their cycle has ended.
//
// A downgrade is scheduled by SubscriptionManager.ScheduleDowngrade (records
// `metadata.scheduled_plan_code` + `cancel_at_period_end`). Without a worker the
// flag was written but never applied, so users kept the higher plan forever.
type DowngradeWorker struct {
	lister   ScheduledDowngradeLister
	apply    func(ctx context.Context, userID uuid.UUID) error
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// DowngradeConfig configures the worker.
type DowngradeConfig struct {
	// Interval between sweeps (default 15m).
	Interval time.Duration
}

// NewDowngradeWorker builds the worker with documented defaults.
func NewDowngradeWorker(lister ScheduledDowngradeLister, apply func(ctx context.Context, userID uuid.UUID) error, cfg DowngradeConfig) *DowngradeWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = 15 * time.Minute
	}
	return &DowngradeWorker{
		lister:   lister,
		apply:    apply,
		interval: cfg.Interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start launches the sweep loop.
func (w *DowngradeWorker) Start(ctx context.Context) {
	go func() {
		defer close(w.done)
		w.RunOnce(ctx)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stop:
				return
			case <-ticker.C:
				w.RunOnce(ctx)
			}
		}
	}()
}

// Stop signals the loop to exit and waits for it.
func (w *DowngradeWorker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.done
}

// RunOnce applies every due scheduled downgrade. Exposed for tests.
//
// Each user is applied independently: one failure does not abort the sweep, and
// ApplyScheduledDowngrade is idempotent (no-op once the schedule is cleared).
func (w *DowngradeWorker) RunOnce(ctx context.Context) {
	if w.lister == nil || w.apply == nil {
		return
	}
	due, err := w.lister.ListDueScheduledDowngrades(ctx, time.Now().UTC())
	if err != nil {
		logger.Warn("downgrade worker: list failed", zap.Error(err))
		return
	}
	for _, userID := range due {
		if err := w.apply(ctx, userID); err != nil {
			logger.Warn("downgrade worker: apply failed",
				zap.String("user", userID.String()), zap.Error(err))
			continue
		}
		logger.Info("downgrade worker: applied scheduled downgrade", zap.String("user", userID.String()))
	}
}
