package payment

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// ExpiryWorker closes unpaid invoices whose due date has passed: it voids the
// invoice (status `expired`) and expires any payment still pending against it,
// so a user is never left with a pay-able-looking invoice for a lapsed window.
//
// This is the automatic counterpart to a user cancelling their own invoice; the
// same close path is shared so the resulting state is identical.
type ExpiryWorker struct {
	repo     Repository
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
	// onClosed, when set, is notified after an invoice is expired (e.g. to emit
	// a realtime event so the UI drops the invoice without a refresh).
	onClosed func(ctx context.Context, userID string, invoiceNumber string)
}

// NewExpiryWorker builds the worker (5-minute scans by default — frequent enough
// that an invoice never lingers long past its due time).
func NewExpiryWorker(repo Repository, interval time.Duration) *ExpiryWorker {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &ExpiryWorker{
		repo:     repo,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// WithOnClosed installs a callback fired after each invoice is expired.
func (w *ExpiryWorker) WithOnClosed(fn func(ctx context.Context, userID string, invoiceNumber string)) *ExpiryWorker {
	w.onClosed = fn
	return w
}

// Start launches the loop.
func (w *ExpiryWorker) Start(ctx context.Context) {
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
func (w *ExpiryWorker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.done
}

// RunOnce expires every open invoice whose due date is in the past. Exported so
// a test (or an admin "sweep now" action) can run one pass deterministically.
func (w *ExpiryWorker) RunOnce(ctx context.Context) {
	now := time.Now().UTC()
	due, err := w.repo.ListInvoicesDueBefore(ctx, now.Unix())
	if err != nil {
		logger.Warn("expiry: list failed", zap.Error(err))
		return
	}
	for _, inv := range due {
		// CloseInvoice only transitions a still-open invoice, so a concurrent
		// payment settling between the scan and this call wins safely.
		changed, err := w.repo.CloseInvoice(ctx, inv.ID, InvoiceExpired, "expired")
		if err != nil {
			logger.Warn("expiry: close invoice failed", zap.Error(err), zap.String("invoice", inv.InvoiceNumber))
			continue
		}
		if !changed {
			continue // already closed by someone else
		}
		if err := w.repo.ExpirePaymentsForInvoice(ctx, inv.ID); err != nil {
			logger.Warn("expiry: expire payments failed", zap.Error(err), zap.String("invoice", inv.InvoiceNumber))
		}
		logger.Info("expiry: invoice expired", zap.String("invoice", inv.InvoiceNumber), zap.String("user_id", inv.UserID.String()))
		if w.onClosed != nil {
			w.onClosed(ctx, inv.UserID.String(), inv.InvoiceNumber)
		}
	}
}
