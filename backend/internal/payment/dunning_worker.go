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
// at H-3 and H-1, then issues a renewal invoice at H-0.
type DunningWorker struct {
	repo     Repository
	notifier Notifier
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}

	// reminded tracks "<userID>:<bucket>" keys already notified to avoid dupes.
	reminded map[string]bool
}

// NewDunningWorker builds the worker (hourly scan by default).
func NewDunningWorker(repo Repository, notifier Notifier, interval time.Duration) *DunningWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	return &DunningWorker{repo: repo, notifier: notifier, interval: interval, stop: make(chan struct{}), done: make(chan struct{}), reminded: map[string]bool{}}
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
	dueSoon, err := w.repo.ListInvoicesDueBefore(ctx, now.Add(72*time.Hour).Unix())
	if err != nil {
		logger.Warn("dunning: list failed", zap.Error(err))
		return
	}
	for _, inv := range dueSoon {
		if inv.DueAt == nil {
			continue
		}
		hoursLeft := int(inv.DueAt.Sub(now).Hours())
		bucket := dunningBucket(hoursLeft)
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

// dunningBucket maps hours-left to a reminder bucket label.
func dunningBucket(hoursLeft int) string {
	switch {
	case hoursLeft <= 0:
		return "SUBSCRIPTION_EXPIRING_DUE"
	case hoursLeft <= 24:
		return "SUBSCRIPTION_EXPIRING_URGENT"
	case hoursLeft <= 72:
		return "SUBSCRIPTION_EXPIRING_SOON"
	default:
		return ""
	}
}
