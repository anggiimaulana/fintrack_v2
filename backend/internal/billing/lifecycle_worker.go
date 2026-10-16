package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// DueSubscription is the minimal snapshot the lifecycle worker needs to decide a
// subscription's end-of-cycle outcome. Keeping it a plain struct (not the full
// Subscription) lets the lister stay a narrow query and the worker be tested
// without a database.
type DueSubscription struct {
	UserID            uuid.UUID
	Status            SubscriptionStatus
	PeriodEnd         time.Time
	CancelAtPeriodEnd bool
	// HasScheduledPlan marks a subscription with an end-of-cycle plan change.
	// Those are owned by the downgrade worker, so this worker skips them.
	HasScheduledPlan bool
}

// DueSubscriptionLister lists subscriptions whose cycle has ended and that are
// NOT scheduled for a plan change (those belong to the downgrade worker).
type DueSubscriptionLister interface {
	ListSubscriptionsDueForLifecycle(ctx context.Context, now time.Time) ([]DueSubscription, error)
}

// LifecycleWorker enforces the end of a billing cycle for subscriptions that
// were neither renewed nor downgraded: a user who simply stops paying.
//
// Before this worker, nothing moved such a subscription out of `active` — it
// kept granting the paid plan forever, and its token period kept rolling forward
// (see LedgerService.ensurePeriodLocked). The worker applies the outcome:
//
//   - cancelled (cancel_at_period_end): access ends now -> `canceled`;
//   - otherwise: a grace window (grace_period_hours) keeps access while the user
//     is reminded to pay -> `past_due`, then `expired` once grace lapses.
//
// `expired`/`canceled` fall back to the default plan (see
// SubscriptionStatus.GrantsPaidAccess), so a lapsed subscriber is soft-limited,
// never locked out and never loses data.
type LifecycleWorker struct {
	lister     DueSubscriptionLister
	transition func(ctx context.Context, userID uuid.UUID, to SubscriptionStatus, eventType, reason string) error
	graceHours func(ctx context.Context) int64
	interval   time.Duration
	stop       chan struct{}
	done       chan struct{}
}

// LifecycleConfig configures the worker.
type LifecycleConfig struct {
	// Interval between sweeps (default 30m). The grace window is measured in
	// hours, so a 30-minute scan keeps the lag well inside tolerance.
	Interval time.Duration
}

// defaultGraceHours is used when the operator setting is absent.
const defaultGraceHours = 72

// NewLifecycleWorker builds the worker with documented defaults.
func NewLifecycleWorker(lister DueSubscriptionLister, transition func(ctx context.Context, userID uuid.UUID, to SubscriptionStatus, eventType, reason string) error, graceHours func(ctx context.Context) int64, cfg LifecycleConfig) *LifecycleWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Minute
	}
	if graceHours == nil {
		graceHours = func(context.Context) int64 { return defaultGraceHours }
	}
	return &LifecycleWorker{
		lister:     lister,
		transition: transition,
		graceHours: graceHours,
		interval:   cfg.Interval,
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// Start launches the sweep loop.
func (w *LifecycleWorker) Start(ctx context.Context) {
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
func (w *LifecycleWorker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.done
}

// RunOnce applies the end-of-cycle outcome to every due subscription. Each
// subscription is handled independently: one failure does not abort the sweep,
// and transitions are idempotent (a subscription already expired is skipped).
func (w *LifecycleWorker) RunOnce(ctx context.Context) {
	if w.lister == nil || w.transition == nil {
		return
	}
	now := time.Now().UTC()
	due, err := w.lister.ListSubscriptionsDueForLifecycle(ctx, now)
	if err != nil {
		logger.Warn("lifecycle worker: list failed", zap.Error(err))
		return
	}
	grace := time.Duration(w.graceHours(ctx)) * time.Hour
	for _, d := range due {
		to, event, reason := lifecycleOutcome(d, now, grace)
		if to == "" {
			continue
		}
		if err := w.transition(ctx, d.UserID, to, event, reason); err != nil {
			logger.Warn("lifecycle worker: transition failed",
				zap.String("user", d.UserID.String()),
				zap.String("to", string(to)), zap.Error(err))
			continue
		}
		logger.Info("lifecycle worker: subscription transitioned",
			zap.String("user", d.UserID.String()),
			zap.String("from", string(d.Status)),
			zap.String("to", string(to)),
			zap.String("reason", reason))
	}
}

// lifecycleOutcome decides the target status for a due subscription. It returns
// an empty status when no transition is warranted (nothing to do).
//
// Pure function so the policy is unit-testable without a worker or a clock.
func lifecycleOutcome(d DueSubscription, now time.Time, grace time.Duration) (SubscriptionStatus, string, string) {
	// A scheduled plan change is handled by the downgrade worker.
	if d.HasScheduledPlan {
		return "", "", ""
	}
	// Terminal states need no action.
	if d.Status == StatusExpired || d.Status == StatusCanceled {
		return "", "", ""
	}
	// The user asked to stop: access ends at the cycle boundary, no grace.
	if d.CancelAtPeriodEnd {
		return StatusCanceled, "canceled", "cancel_at_period_end reached"
	}
	// Unpaid: keep access during the grace window, then expire.
	if now.After(d.PeriodEnd.Add(grace)) {
		return StatusExpired, "expired", "billing cycle ended without renewal"
	}
	// Inside grace: flag the lapse so dunning and the UI can nudge the user.
	if d.Status != StatusPastDue {
		return StatusPastDue, "past_due", "billing cycle ended, within grace window"
	}
	return "", "", ""
}
