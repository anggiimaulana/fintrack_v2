package payment

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// Notification is an in-app/system notification emitted by dunning/grace.
type Notification struct {
	UserID  string `json:"user_id"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Notifier receives notifications (persisted + pushed via SSE/pub-sub).
type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

// DunningWorker scans for subscriptions approaching expiry and emits reminders
// at configurable H-N offsets, then a renewal reminder at H-0.
//
// The reminder offsets are read from `settings` (dunning_hours_due/urgent/soon)
// so an operator can retune the cadence from the Super Admin portal without a
// redeploy. When a setting is absent the built-in defaults (0/24/72h) apply.
type DunningWorker struct {
	repo     Repository
	settings SettingsRepository
	notifier Notifier
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}

	// reminded tracks "<userID>:<bucket>" keys already notified to avoid dupes.
	reminded map[string]bool
}

// Dunning defaults (hours before due) when the corresponding setting is absent.
const (
	defaultDunningUrgentHours = 24
	defaultDunningSoonHours   = 72
)

// NewDunningWorker builds the worker (hourly scan by default). settings may be
// nil, in which case the built-in default offsets are used.
func NewDunningWorker(repo Repository, settings SettingsRepository, notifier Notifier, interval time.Duration) *DunningWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	return &DunningWorker{repo: repo, settings: settings, notifier: notifier, interval: interval, stop: make(chan struct{}), done: make(chan struct{}), reminded: map[string]bool{}}
}

// dunningWindow returns the look-ahead horizon (the largest configured offset)
// so the scan only loads invoices that could match any bucket.
func (w *DunningWorker) dunningWindow(ctx context.Context) time.Duration {
	_, _, soon := w.dunningThresholds(ctx)
	return time.Duration(soon) * time.Hour
}

// dunningThresholds returns the configured (urgent, soon) hour offsets. Each
// falls back to its default when the setting is absent, zero, or out of order,
// and the pair is kept monotonic (urgent <= soon) so buckets never overlap.
func (w *DunningWorker) dunningThresholds(ctx context.Context) (due, urgent, soon int) {
	urgent = defaultDunningUrgentHours
	soon = defaultDunningSoonHours
	if w.settings != nil {
		if v, err := w.settings.GetInt(ctx, "dunning_hours_urgent"); err == nil && v > 0 {
			urgent = int(v)
		}
		if v, err := w.settings.GetInt(ctx, "dunning_hours_soon"); err == nil && v > 0 {
			soon = int(v)
		}
	}
	if urgent > soon {
		urgent = soon
	}
	return 0, urgent, soon
}

// Start launches the scan loop.
func (w *DunningWorker) Start(ctx context.Context) {
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
func (w *DunningWorker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.done
}

// RunOnce scans due-soon subscriptions and emits reminders (H-3/H-1) or a
// renewal invoice (H-0). Deduplicates per user+bucket.
func (w *DunningWorker) RunOnce(ctx context.Context) {
	now := time.Now().UTC()
	due, urgent, soon := w.dunningThresholds(ctx)
	dueSoon, err := w.repo.ListInvoicesDueBefore(ctx, now.Add(w.dunningWindow(ctx)).Unix())
	if err != nil {
		logger.Warn("dunning: list failed", zap.Error(err))
		return
	}
	for _, inv := range dueSoon {
		if inv.DueAt == nil {
			continue
		}
		hoursLeft := int(inv.DueAt.Sub(now).Hours())
		bucket := dunningBucket(hoursLeft, due, urgent, soon)
		if bucket == "" {
			continue
		}
		key := inv.UserID.String() + ":" + bucket
		if w.reminded[key] {
			continue
		}
		w.reminded[key] = true
		if w.notifier != nil {
			_ = w.notifier.Notify(ctx, Notification{
				UserID:  inv.UserID.String(),
				Type:    bucket,
				Message: "Langganan Anda akan berakhir. Segera lakukan pembayaran.",
			})
		}
	}
}

// dunningBucket maps hours-left to a reminder bucket label using the configured
// thresholds: due (<=0), urgent (<=urgent), soon (<=soon), else none.
func dunningBucket(hoursLeft, due, urgent, soon int) string {
	switch {
	case hoursLeft <= due:
		return "SUBSCRIPTION_EXPIRING_DUE"
	case hoursLeft <= urgent:
		return "SUBSCRIPTION_EXPIRING_URGENT"
	case hoursLeft <= soon:
		return "SUBSCRIPTION_EXPIRING_SOON"
	default:
		return ""
	}
}
