package payment

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// GraceWorker enforces the grace period: invoices past due within the grace
// window keep access (with a warning); beyond it the subscription goes past_due
// (soft-limit — no data deletion).
type GraceWorker struct {
	repo     Repository
	settings SettingsRepository
	// setSubscriptionStatus transitions a user's subscription status.
	setSubscriptionStatus func(ctx context.Context, userID string, status string) error
	interval              time.Duration
	stop                  chan struct{}
	done                  chan struct{}
}

// NewGraceWorker builds the worker (30-minute scans by default).
func NewGraceWorker(repo Repository, settings SettingsRepository, setStatus func(context.Context, string, string) error, interval time.Duration) *GraceWorker {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &GraceWorker{repo: repo, settings: settings, setSubscriptionStatus: setStatus, interval: interval, stop: make(chan struct{}), done: make(chan struct{})}
}

// Start launches the loop.
func (w *GraceWorker) Start(ctx context.Context) {
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
				w.RunOnce(ctx)
			}
		}
	}()
}

// Stop halts the worker.
func (w *GraceWorker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.done
}

// GraceHours returns the configured grace window (default 72h).
func (w *GraceWorker) GraceHours(ctx context.Context) int64 {
	if w.settings != nil {
		if v, err := w.settings.GetInt(ctx, "grace_period_hours"); err == nil && v > 0 {
			return v
		}
	}
	return 72
}

// RunOnce transitions overdue invoices: within grace -> grace; beyond -> past_due.
func (w *GraceWorker) RunOnce(ctx context.Context) {
	now := time.Now().UTC()
	graceHours := w.GraceHours(ctx)
	overdue, err := w.repo.ListInvoicesDueBefore(ctx, now.Unix())
	if err != nil {
		logger.Warn("grace: list failed", zap.Error(err))
		return
	}
	for _, inv := range overdue {
		if inv.DueAt == nil {
			continue
		}
		hoursPast := now.Sub(*inv.DueAt).Hours()
		if hoursPast <= float64(graceHours) {
			// Within grace: keep access, set grace status.
			if w.setSubscriptionStatus != nil {
				_ = w.setSubscriptionStatus(ctx, inv.UserID.String(), "grace")
			}
		} else {
			// Beyond grace: past_due (soft-limit). Invoice marked past_due too.
			if err := w.repo.UpdateInvoiceStatus(ctx, inv.ID, InvoicePastDue, nil, nil); err != nil {
				logger.Warn("grace: update invoice failed", zap.Error(err))
			}
			if w.setSubscriptionStatus != nil {
				_ = w.setSubscriptionStatus(ctx, inv.UserID.String(), "past_due")
			}
		}
	}
}
