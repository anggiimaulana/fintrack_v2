package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// RenewalCandidate is one subscription that has reached the end of its cycle and
// is eligible for an automatic renewal invoice.
type RenewalCandidate struct {
	UserID         uuid.UUID
	SubscriptionID uuid.UUID
	PlanCode       string
	PlanName       string
	PriceIDR       int64
	PeriodEnd      time.Time
}

// RenewalSource lists subscriptions due for a renewal invoice.
type RenewalSource interface {
	ListRenewalCandidates(ctx context.Context, now time.Time) ([]RenewalCandidate, error)
}

// RenewalNotifier announces a created renewal invoice or successful auto-charge. Nil is safe.
type RenewalNotifier interface {
	RenewalInvoiceIssued(ctx context.Context, userID uuid.UUID, invoice *Invoice, planName string) error
	RenewalAutoCharged(ctx context.Context, userID uuid.UUID, invoice *Invoice, maskedCard, planName string) error
}

// RenewalWorker issues the next cycle's invoice when a subscription's period
// ends, so a subscriber who wants to continue can pay without re-checking out.
//
// Design:
//
//   - Runs at H-0 (period_end reached). The invoice due date follows the
//     operator's invoice window (invoice_expiry_hours/minutes), so the grace
//     window the lifecycle worker enforces lines up with when the invoice lapses.
//   - Skips a subscription the user asked to cancel (cancel_at_period_end) and
//     one that already has an open invoice (the DB enforces at most one open
//     invoice per user, so issuing a second would fail anyway).
//   - Idempotent per cycle: the invoice carries a deterministic marker in its
//     notes (`renewal:<subscriptionID>:<periodEndUnix>`), and the worker skips a
//     cycle that already produced an invoice — so a restart or a repeated sweep
//     never double-bills.
//   - Never blocks the lifecycle worker: a subscription with no successful
//     invoice still expires after grace.
type RenewalWorker struct {
	source      RenewalSource
	invoices    *InvoiceService
	repo        Repository
	notifier    RenewalNotifier
	autoCharger *AutoCharger
	interval    time.Duration
	stop        chan struct{}
	done        chan struct{}
}

// RenewalConfig configures the worker.
type RenewalConfig struct {
	// Interval between sweeps (default 30m).
	Interval time.Duration
}

// NewRenewalWorker builds the worker with documented defaults.
func NewRenewalWorker(source RenewalSource, repo Repository, invoices *InvoiceService, notifier RenewalNotifier, cfg RenewalConfig) *RenewalWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Minute
	}
	return &RenewalWorker{
		source:   source,
		repo:     repo,
		invoices: invoices,
		notifier: notifier,
		interval: cfg.Interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// WithAutoCharger configures an auto-charger to attempt silent recurring billing
// when candidates reach renewal.
func (w *RenewalWorker) WithAutoCharger(ac *AutoCharger) *RenewalWorker {
	w.autoCharger = ac
	return w
}
// Start launches the sweep loop.
func (w *RenewalWorker) Start(ctx context.Context) {
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
func (w *RenewalWorker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.done
}

// RenewalMarker is the deterministic, per-cycle idempotency marker stored in the
// invoice notes. It makes "has this cycle been billed?" a cheap lookup.
//
// EXPORTED and used by the repository's candidate query too: the Go side builds
// it with Unix() (truncation) and the SQL side with floor(extract(epoch ...)), so
// both must agree. A mismatch (e.g. SQL rounding) means the exclusion never
// matches and the same cycle is re-invoiced on every sweep — a double-bill.
func RenewalMarker(subscriptionID uuid.UUID, periodEnd time.Time) string {
	return fmt.Sprintf("renewal:%s:%d", subscriptionID.String(), periodEnd.Unix())
}

// RunOnce issues a renewal invoice for every due, eligible subscription. Each
// subscription is handled independently: one failure does not abort the sweep.
func (w *RenewalWorker) RunOnce(ctx context.Context) {
	if w.source == nil || w.invoices == nil || w.repo == nil {
		return
	}
	now := time.Now().UTC()
	due, err := w.source.ListRenewalCandidates(ctx, now)
	if err != nil {
		logger.Warn("renewal worker: list failed", zap.Error(err))
		return
	}
	for _, c := range due {
		if c.PriceIDR <= 0 {
			continue // free plan: nothing to bill
		}
		marker := RenewalMarker(c.SubscriptionID, c.PeriodEnd)
		already, err := w.repo.InvoiceExistsByMarker(ctx, marker)
		if err != nil {
			logger.Warn("renewal worker: marker lookup failed",
				zap.String("user", c.UserID.String()), zap.Error(err))
			continue
		}
		if already {
			continue // this cycle was already billed
		}
		// At most one open invoice per user: skip when one is already waiting.
		if open, oerr := w.repo.GetOpenInvoiceForUser(ctx, c.UserID); oerr == nil && open != nil {
			continue
		}

		inv, cerr := w.invoices.Create(ctx, CreateInvoiceDTO{
			UserID:         c.UserID,
			SubscriptionID: &c.SubscriptionID,
			SubtotalIDR:    c.PriceIDR,
			// The plan lives in its own column; the marker goes in notes so the
			// per-cycle dedupe never collides with the plan code.
			PlanCode: c.PlanCode,
			Notes:    marker,
		})
		if cerr != nil {
			logger.Warn("renewal worker: create invoice failed",
				zap.String("user", c.UserID.String()), zap.Error(cerr))
			continue
		}
		logger.Info("renewal worker: invoice issued",
			zap.String("user", c.UserID.String()),
			zap.String("invoice", inv.InvoiceNumber),
			zap.String("plan", c.PlanCode))

		// If user has a saved card and auto-charge is available, attempt auto-charge.
		autoCharged := false
		if w.autoCharger != nil {
			card, err := w.repo.GetDefaultCardToken(ctx, c.UserID)
			if err == nil && card != nil {
				res, cerr := w.autoCharger.ChargeInvoice(ctx, inv, card)
				if cerr != nil {
					logger.Warn("renewal worker: auto-charge error",
						zap.String("user", c.UserID.String()),
						zap.String("invoice", inv.InvoiceNumber),
						zap.Error(cerr))
				} else if res != nil && res.Settled {
					autoCharged = true
					logger.Info("renewal worker: auto-charge settled successfully",
						zap.String("user", c.UserID.String()),
						zap.String("invoice", inv.InvoiceNumber),
						zap.String("masked_card", res.MaskedCard))
					if w.notifier != nil {
						_ = w.notifier.RenewalAutoCharged(ctx, c.UserID, inv, res.MaskedCard, c.PlanName)
					}
				}
			}
		}

		// If auto-charge did not settle (no saved card or payment pending/failed), notify of issued invoice.
		if !autoCharged && w.notifier != nil {
			if nerr := w.notifier.RenewalInvoiceIssued(ctx, c.UserID, inv, c.PlanName); nerr != nil {
				logger.Warn("renewal worker: notify failed",
					zap.String("user", c.UserID.String()),
					zap.Error(nerr))
			}
		}
	}
}
