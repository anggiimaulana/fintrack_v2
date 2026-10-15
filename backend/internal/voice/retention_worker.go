package voice

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// HistoryPruner deletes voice-history rows older than a cutoff. Implemented by
// the postgres repository; kept narrow so the worker is unit-testable.
type HistoryPruner interface {
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
	// PruneDailyUsageBefore deletes daily-counter rows for days before a date,
	// keeping that table bounded (it is separate from the history rows).
	PruneDailyUsageBefore(ctx context.Context, beforeDay time.Time) (int64, error)
}

// RetentionWorker prunes voice transcriptions older than the configured
// retention window (N-01). The window is dynamic: it is re-read from the config
// on every sweep, so a Super Admin change applies without a restart.
type RetentionWorker struct {
	pruner HistoryPruner
	// retentionDays resolves the current retention window. Returning <= 0 skips
	// the sweep (a guard against a misconfigured zero deleting everything).
	retentionDays func(ctx context.Context) int
	interval      time.Duration
	now           func() time.Time
	stop          chan struct{}
	done          chan struct{}
}

// RetentionConfig configures the worker.
type RetentionConfig struct {
	// Interval between sweeps (default 6h; retention is a slow-moving concern).
	Interval time.Duration
}

// NewRetentionWorker builds the worker with documented defaults.
func NewRetentionWorker(pruner HistoryPruner, retentionDays func(ctx context.Context) int, cfg RetentionConfig) *RetentionWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = 6 * time.Hour
	}
	return &RetentionWorker{
		pruner:        pruner,
		retentionDays: retentionDays,
		interval:      cfg.Interval,
		now:           func() time.Time { return time.Now().UTC() },
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
}

// WithClock overrides the time source (tests).
func (w *RetentionWorker) WithClock(now func() time.Time) *RetentionWorker {
	w.now = now
	return w
}

// Start launches the sweep loop.
func (w *RetentionWorker) Start(ctx context.Context) {
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
func (w *RetentionWorker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.done
}

// RunOnce prunes rows older than the retention window. Exposed for tests.
func (w *RetentionWorker) RunOnce(ctx context.Context) {
	if w.pruner == nil || w.retentionDays == nil {
		return
	}
	days := w.retentionDays(ctx)
	if days <= 0 {
		// Never prune on a non-positive window: that would wipe the table.
		logger.Warn("voice retention worker: non-positive window, skipping sweep",
			zap.Int("days", days))
		return
	}
	cutoff := w.now().AddDate(0, 0, -days)
	removed, err := w.pruner.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		logger.Warn("voice retention sweep failed", zap.Error(err))
		return
	}
	if removed > 0 {
		logger.Info("voice retention sweep pruned history",
			zap.Int64("rows", removed), zap.Int("retention_days", days))
	}

	// The daily counter table is separate and also unbounded; prune rows for days
	// older than the same window. Only past days are removed, so the current
	// day's counter (which the gate reads) is never affected.
	if pruned, perr := w.pruner.PruneDailyUsageBefore(ctx, cutoff); perr != nil {
		logger.Warn("voice daily-usage prune failed", zap.Error(perr))
	} else if pruned > 0 {
		logger.Info("voice retention sweep pruned daily counters", zap.Int64("rows", pruned))
	}
}
