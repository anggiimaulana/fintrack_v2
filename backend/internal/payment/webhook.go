package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	// emitter publishes realtime domain events (never nil; no-op by default).
	emitter RealtimeEmitter
}

// NewWebhookProcessor builds the processor.
func NewWebhookProcessor(repo Repository, registry *Registry, activate SubscriptionActivator, tx TxRunner) *WebhookProcessor {
	return &WebhookProcessor{repo: repo, registry: registry, activate: activate, tx: tx, emitter: noopEmitter{}}
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
	var settled *Payment
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
					// The plan lives in the invoice's dedicated `plan_code`
					// column (never `notes`, which carries metadata such as the
					// renewal marker). A blank code means the invoice is not
					// plan-bound (e.g. a legacy row): record the payment as
					// settled but skip activation, rather than failing the
					// transaction and losing a payment the user already made.
					if code := planCodeFromInvoice(inv); code == "" {
						logger.Warn("paid invoice has no plan_code; skipping activation",
							zap.String("invoice", inv.InvoiceNumber),
							zap.String("user", pay.UserID.String()))
					} else if aerr := p.activate.ActivateForUser(txCtx, pay.UserID, code); aerr != nil {
						return fmt.Errorf("activate subscription: %w", aerr)
					}
				}
			}
			// Tokenized card: persist saved card token for future auto-charges.
			if res.SavedTokenID != "" && pay.UserID != uuid.Nil {
				card := &CardToken{
					UserID:     pay.UserID,
					Gateway:    gatewayName,
					TokenID:    res.SavedTokenID,
					MaskedCard: res.MaskedCard,
					CardType:   res.CardType,
					Bank:       res.Bank,
					IsDefault:  true,
				}
				if res.SavedTokenIDExpiredAt != "" {
					if exp, perr := time.Parse("2006-01-02 15:04:05", res.SavedTokenIDExpiredAt); perr == nil {
						card.ExpiresAt = &exp
					}
				}
				if serr := p.repo.SaveCardToken(txCtx, card); serr != nil {
					logger.Warn("webhook: failed to persist saved card token",
						zap.String("user", pay.UserID.String()),
						zap.Error(serr))
				} else {
					logger.Info("webhook: saved card token recorded",
						zap.String("user", pay.UserID.String()),
						zap.String("masked_card", res.MaskedCard))
				}
			}

			pay.Status = PaymentSettled
			settled = pay
		}

		result = ProcessResult{Status: res.Status, PaymentID: pay.ID.String()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Emitted after the transaction commits, so the client never receives an
	// event for a payment that was rolled back.
	if settled != nil && !result.AlreadyProcessed {
		emitPaymentReceived(ctx, p.emitter, settled, "")
	}
	return &result, nil
}

// planCodeFromInvoice returns the plan an invoice buys. It reads the dedicated
// `plan_code` column: `notes` is free-form metadata (e.g. the renewal marker) and
// must never be interpreted as a plan name.
func planCodeFromInvoice(inv *Invoice) string {
	if inv == nil || inv.PlanCode == nil {
		return ""
	}
	return *inv.PlanCode
}

var _ = errors.New
