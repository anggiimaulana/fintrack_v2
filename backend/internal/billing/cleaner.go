package billing

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// CleanerWorker auto-releases reservations that were never committed or
// released (e.g. the process crashed mid-request). It prevents quota from
// staying locked forever.
type CleanerWorker struct {
	tokens   TokenRepository
	ledger   *LedgerService
	interval time.Duration
	maxAge   time.Duration

	stop chan struct{}
	done chan struct{}
}

// CleanerConfig configures the worker.
type CleanerConfig struct {
	// Interval between sweeps (default 15m).
	Interval time.Duration
	// MaxAge is how old a reservation must be to be released (default 30m).
	MaxAge time.Duration
}

// NewCleanerWorker builds the worker with documented defaults.
func NewCleanerWorker(tokens TokenRepository, ledger *LedgerService, cfg CleanerConfig) *CleanerWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = 15 * time.Minute
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 30 * time.Minute
	}
	return &CleanerWorker{
		tokens:   tokens,
		ledger:   ledger,
		interval: cfg.Interval,
		maxAge:   cfg.MaxAge,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start launches the sweep loop.
func (w *CleanerWorker) Start(ctx context.Context) {
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stop:
				return
			case <-ticker.C:
				w.sweep(ctx)
			}
		}
	}()
}

// Stop signals the loop to exit and waits for it.
func (w *CleanerWorker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.done
}

// sweep releases all reservations older than maxAge.
func (w *CleanerWorker) sweep(ctx context.Context) {
	cutoff := time.Now().UTC().Add(-w.maxAge)
	expired, err := w.tokens.ListExpiredReservations(ctx, cutoff)
	if err != nil {
		logger.Warn("ledger cleaner: list failed", zap.Error(err))
		return
	}
	for _, r := range expired {
		key := "cleanup:" + r.ID.String()
		if err := w.ledger.Release(ctx, r.UserID, r.ID, key); err != nil {
			logger.Warn("ledger cleaner: release failed",
				zap.String("reservation", r.ID.String()), zap.Error(err))
			continue
		}
		logger.Info("ledger cleaner: released stale reservation",
			zap.String("reservation", r.ID.String()), zap.Int64("amount", r.Amount))
	}
}

// SweepOnce exposes a single sweep for tests.
func (w *CleanerWorker) SweepOnce(ctx context.Context) { w.sweep(ctx) }
