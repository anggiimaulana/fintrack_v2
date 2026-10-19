package payment

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/logger"
)

// Notification is an in-app/system notification emitted by dunning.
type Notification struct {
	UserID  string `json:"user_id"`
	Type    string `json:"type"`
	Message string `json:"message"`
	// DedupeKey makes delivery idempotent across restarts and replicas.
	DedupeKey string `json:"dedupe_key"`
}

// Notifier receives notifications (persisted + pushed via SSE/pub-sub).
type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

// SubscriptionExpiry is the subset of a subscription the dunning scan needs.
type SubscriptionExpiry struct {
	UserID    uuid.UUID
	PeriodEnd time.Time
}

// DunningWorker reminds users before their paid period ends.
//
// It scans SUBSCRIPTIONS, not invoices: an invoice's due date is the payment
// window (often 72h), so keying on it warned users "subscription ending" weeks
// before it actually ended (and after an upgrade invoice, when nothing was
// ending at all).
//
// Offsets come from `settings` (dunning_hours_soon/urgent), default H-7 / H-1.
// Dedupe is persisted (notifications.dedupe_key = user + period end + bucket),
// so a process restart or a second replica cannot re-send the same reminder.
type DunningWorker struct {
	repo     Repository
	settings SettingsRepository
	notifier Notifier
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// Dunning defaults (hours before period end) when the setting is absent.
const (
	defaultDunningUrgentHours = 24
	defaultDunningSoonHours   = 168 // H-7
)

// NewDunningWorker builds the worker (hourly scan by default). settings may be
// nil, in which case the built-in default offsets are used.
func NewDunningWorker(repo Repository, settings SettingsRepository, notifier Notifier, interval time.Duration) *DunningWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	return &DunningWorker{repo: repo, settings: settings, notifier: notifier, interval: interval, stop: make(chan struct{}), done: make(chan struct{})}
}

// dunningThresholds returns the configured (urgent, soon) hour offsets. Each
// falls back to its default when the setting is absent, zero, or out of order,
// and the pair is kept monotonic (urgent <= soon) so buckets never overlap.
func (w *DunningWorker) dunningThresholds(ctx context.Context) (urgent, soon int) {
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
	return urgent, soon
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

// RunOnce reminds subscriptions whose period ends inside the configured
// window. Already-ended subscriptions (expired/period_end past) are not
// returned by the repository, so no reminder can arrive after expiry.
func (w *DunningWorker) RunOnce(ctx context.Context) {
	now := time.Now().UTC()
	urgent, soon := w.dunningThresholds(ctx)
	ending, err := w.repo.ListSubscriptionsEndingBefore(ctx, now.Add(time.Duration(soon)*time.Hour).Unix())
	if err != nil {
		logger.Warn("dunning: list failed", zap.Error(err))
		return
	}
	for _, sub := range ending {
		hoursLeft := int(sub.PeriodEnd.Sub(now).Hours())
		bucket := dunningBucket(hoursLeft, urgent, soon)
		if bucket == "" {
			continue
		}
		if w.notifier == nil {
			continue
		}
		_ = w.notifier.Notify(ctx, Notification{
			UserID:    sub.UserID.String(),
			Type:      bucket,
			Message:   "Langganan Anda akan berakhir. Segera lakukan pembayaran.",
			DedupeKey: fmt.Sprintf("subscription:%s:%d:%s", sub.UserID, sub.PeriodEnd.Unix(), bucket),
		})
	}
}

// dunningBucket maps hours-left to a reminder bucket label using the configured
// thresholds: urgent (<=urgent), soon (<=soon), else none. A non-positive value
// means the period already ended, which is never reminded.
func dunningBucket(hoursLeft, urgent, soon int) string {
	switch {
	case hoursLeft < 0:
		return ""
	case hoursLeft <= urgent:
		return "SUBSCRIPTION_EXPIRING_URGENT"
	case hoursLeft <= soon:
		return "SUBSCRIPTION_EXPIRING_SOON"
	default:
		return ""
	}
}
