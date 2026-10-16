package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// AutoCharger executes auto-charges against unpaid invoices using tokenized cards.
// Card numbers never touch this application (PCI-DSS SAQ-A compliant).
type AutoCharger struct {
	repo     Repository
	charger  CardCharger
	activate SubscriptionActivator
	tx       TxRunner
	emitter  RealtimeEmitter
	now      func() time.Time
}

// NewAutoCharger builds an auto-charge coordinator.
func NewAutoCharger(repo Repository, charger CardCharger, activate SubscriptionActivator, tx TxRunner) *AutoCharger {
	return &AutoCharger{
		repo:     repo,
		charger:  charger,
		activate: activate,
		tx:       tx,
		emitter:  noopEmitter{},
		now:      time.Now,
	}
}

// WithEmitter sets the realtime emitter for publishing settlement events.
func (a *AutoCharger) WithEmitter(e RealtimeEmitter) *AutoCharger {
	if e != nil {
		a.emitter = e
	}
	return a
}

// AutoChargeResult reports the outcome of an auto-charge attempt.
type AutoChargeResult struct {
	PaymentID  uuid.UUID     `json:"payment_id"`
	Settled    bool          `json:"settled"`
	MaskedCard string        `json:"masked_card"`
	Bank       string        `json:"bank,omitempty"`
	Status     PaymentStatus `json:"status"`
	Error      string        `json:"error,omitempty"`
}

// ChargeInvoice attempts to settle an invoice using a user's saved card.
// If cardToken is nil, it automatically fetches the user's default saved card.
// Returns (result, nil) if an attempt was made, or (nil, ErrCardTokenNotFound) if no card is available.
func (a *AutoCharger) ChargeInvoice(ctx context.Context, inv *Invoice, cardToken *CardToken) (*AutoChargeResult, error) {
	if inv == nil {
		return nil, ErrInvoiceNotFound
	}
	if inv.Status != InvoiceIssued && inv.Status != InvoicePastDue {
		return nil, fmt.Errorf("%w: invoice is %s", ErrInvalidState, inv.Status)
	}

	if cardToken == nil {
		card, err := a.repo.GetDefaultCardToken(ctx, inv.UserID)
		if err != nil {
			return nil, err
		}
		if card == nil {
			return nil, ErrCardTokenNotFound
		}
		cardToken = card
	}

	if a.charger == nil {
		return nil, ErrGatewayNotConfigured
	}

	orderID := fmt.Sprintf("%s-AC-%d", inv.InvoiceNumber, a.now().Unix())
	idemKey := "autocharge:" + orderID

	pay := &Payment{
		ID:             uuid.New(),
		InvoiceID:      inv.ID,
		UserID:         inv.UserID,
		Gateway:        cardToken.Gateway,
		GatewayRef:     &orderID,
		AmountIDR:      inv.TotalIDR,
		Status:         PaymentPending,
		IdempotencyKey: idemKey,
	}
	if err := a.repo.CreatePayment(ctx, pay); err != nil {
		return nil, fmt.Errorf("create payment attempt: %w", err)
	}

	res, err := a.charger.ChargeSavedCard(ctx, ChargeSavedCardRequest{
		UserID:    inv.UserID,
		OrderID:   orderID,
		AmountIDR: inv.TotalIDR,
		TokenID:   cardToken.TokenID,
	})
	if err != nil {
		logger.Warn("autocharge: card charge call failed",
			zap.String("invoice", inv.InvoiceNumber),
			zap.String("user", inv.UserID.String()),
			zap.Error(err))
		_ = a.repo.UpdatePaymentStatus(ctx, pay.ID, PaymentFailed, nil, nil, map[string]any{"error": err.Error()})
		return &AutoChargeResult{
			PaymentID:  pay.ID,
			Settled:    false,
			MaskedCard: cardToken.MaskedCard,
			Bank:       cardToken.Bank,
			Status:     PaymentFailed,
			Error:      err.Error(),
		}, nil
	}

	method := res.PaymentMethod
	snapshot := res.Raw

	if res.Status == PaymentSettled {
		err := a.tx.RunInTx(ctx, func(txCtx context.Context) error {
			if err := a.repo.UpdatePaymentStatus(txCtx, pay.ID, PaymentSettled, &method, nil, snapshot); err != nil {
				return err
			}
			if err := a.repo.MarkInvoicePaid(txCtx, inv.ID); err != nil {
				return err
			}
			if a.activate != nil && inv.PlanCode != nil && *inv.PlanCode != "" {
				if err := a.activate.ActivateForUser(txCtx, inv.UserID, *inv.PlanCode); err != nil {
					return fmt.Errorf("activate subscription: %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("commit autocharge settlement: %w", err)
		}

		pay.Status = PaymentSettled
		pay.PaymentMethod = &method
		emitPaymentReceived(ctx, a.emitter, pay, "")

		return &AutoChargeResult{
			PaymentID:  pay.ID,
			Settled:    true,
			MaskedCard: res.MaskedCard,
			Bank:       res.Bank,
			Status:     PaymentSettled,
		}, nil
	}

	// Non-settled status (e.g. pending 3DS challenge, or denied)
	_ = a.repo.UpdatePaymentStatus(ctx, pay.ID, res.Status, &method, nil, snapshot)
	return &AutoChargeResult{
		PaymentID:  pay.ID,
		Settled:    false,
		MaskedCard: res.MaskedCard,
		Bank:       res.Bank,
		Status:     res.Status,
	}, nil
}
