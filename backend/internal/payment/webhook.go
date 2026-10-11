package payment

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// TxRunner runs fn in a single DB transaction.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// WebhookProcessor verifies and applies a gateway webhook idempotently.
type WebhookProcessor struct {
	repo     Repository
	registry *Registry
	activate SubscriptionActivator
	tx       TxRunner
}

// NewWebhookProcessor builds the processor.
func NewWebhookProcessor(repo Repository, registry *Registry, activate SubscriptionActivator, tx TxRunner) *WebhookProcessor {
	return &WebhookProcessor{repo: repo, registry: registry, activate: activate, tx: tx}
}

// ProcessResult reports what the processor did (for logging/tests).
type ProcessResult struct {
	AlreadyProcessed bool
	Status           PaymentStatus
	PaymentID        string
}

// Process handles a Midtrans webhook:
//  1. Verify the signature (constant-time compare in the gateway).
//  2. Re-verify via the gateway Status API (anti-spoof).
//  3. Apply the status transition idempotently under a row lock.
//
// Replays (already-settled payments) return success without side effects.
func (p *WebhookProcessor) Process(ctx context.Context, gatewayName string, payload []byte, headers map[string]string) (*ProcessResult, error) {
	gw, ok := p.registry.Get(gatewayName)
	if !ok {
		return nil, ErrGatewayNotConfigured
	}

	// 1) Signature verification.
	res, err := gw.VerifyWebhook(ctx, payload, headers)
	if err != nil {
		return nil, err
	}

	// 2) Independent pull verification (authoritative status).
	if status, err := gw.CheckStatus(ctx, res.GatewayRef); err == nil && status != nil {
		res.Status = status.Status
		if status.AmountIDR > 0 {
			res.AmountIDR = status.AmountIDR
		}
	} else if err != nil {
		logger.Warn("webhook pull verification failed; using signed payload",
			zap.String("ref", res.GatewayRef), zap.Error(err))
	}

	// 3) Idempotent application under a row lock.
	var result ProcessResult
	err = p.tx.RunInTx(ctx, func(txCtx context.Context) error {
		pay, err := p.repo.GetPaymentByGatewayRefForUpdate(txCtx, res.GatewayRef)
		if err != nil {
			return err
		}
		if pay == nil {
			return ErrPaymentNotFound
		}

		// Already final -> idempotent no-op.
		if pay.Status.IsFinal() {
			result = ProcessResult{AlreadyProcessed: true, Status: pay.Status, PaymentID: pay.ID.String()}
			return nil
		}

		// Apply transition.
		method := res.PaymentMethod
		snapshot := res.Raw
		if err := p.repo.UpdatePaymentStatus(txCtx, pay.ID, res.Status, &method, nil, snapshot); err != nil {
			return err
		}

		if res.Status == PaymentSettled {
			if err := p.repo.MarkInvoicePaid(txCtx, pay.InvoiceID); err != nil {
				return err
			}
			if p.activate != nil {
				inv, ierr := p.repo.GetInvoiceByID(txCtx, pay.InvoiceID)
				if ierr == nil && inv != nil {
					// The plan code is stored in invoice notes/metadata by the
					// checkout flow; here we activate for the user.
					if aerr := p.activate.ActivateForUser(txCtx, pay.UserID, planCodeFromInvoice(inv)); aerr != nil {
						return fmt.Errorf("activate subscription: %w", aerr)
					}
				}
			}
		}

		result = ProcessResult{Status: res.Status, PaymentID: pay.ID.String()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// planCodeFromInvoice extracts the plan code recorded on the invoice notes.
func planCodeFromInvoice(inv *Invoice) string {
	if inv == nil || inv.Notes == nil {
		return ""
	}
	return *inv.Notes
}

var _ = errors.New
