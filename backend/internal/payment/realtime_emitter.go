package payment

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/realtime"
	"github.com/fintrack/backend/pkg/contracts"
)

// RealtimeEmitter publishes domain events to the realtime transport. Declared as
// an interface so payment code does not depend on the concrete publisher (and so
// tests can capture emissions without Redis).
type RealtimeEmitter interface {
	PublishUser(ctx context.Context, userID uuid.UUID, ev contracts.Event)
	PublishAdmins(ctx context.Context, ev contracts.Event)
}

// noopEmitter is used when realtime is not wired (tests, REALTIME_ENABLED=off).
type noopEmitter struct{}

func (noopEmitter) PublishUser(context.Context, uuid.UUID, contracts.Event) {}
func (noopEmitter) PublishAdmins(context.Context, contracts.Event)          {}

// SetEmitter installs the realtime emitter. Passing nil restores the no-op
// emitter, so emitting is always safe to call.
func (s *ManualService) SetEmitter(e RealtimeEmitter) {
	if e == nil {
		s.emitter = noopEmitter{}
		return
	}
	s.emitter = e
}

// SetEmitter installs the realtime emitter for the webhook processor.
func (p *WebhookProcessor) SetEmitter(e RealtimeEmitter) {
	if e == nil {
		p.emitter = noopEmitter{}
		return
	}
	p.emitter = e
}

// emitPaymentReceived notifies the paying user that their payment settled.
func emitPaymentReceived(ctx context.Context, e RealtimeEmitter, pay *Payment, invoiceNumber string) {
	if e == nil {
		return
	}
	e.PublishUser(ctx, pay.UserID, contracts.NewEvent(contracts.EventPaymentReceived, time.Now().Unix(),
		contracts.PaymentReceivedPayload{
			PaymentID:     pay.ID.String(),
			InvoiceID:     pay.InvoiceID.String(),
			InvoiceNumber: invoiceNumber,
			AmountIDR:     pay.AmountIDR,
			Status:        string(pay.Status),
		}))
}

// emitManualApprovalNeeded notifies admins that a manual transfer awaits review.
func emitManualApprovalNeeded(ctx context.Context, e RealtimeEmitter, pay *Payment, email, invoiceNumber string) {
	if e == nil {
		return
	}
	e.PublishAdmins(ctx, contracts.NewEvent(contracts.EventManualApprovalNeeded, time.Now().Unix(),
		contracts.ManualApprovalNeededPayload{
			PaymentID:  pay.ID.String(),
			UserEmail:  email,
			AmountIDR:  pay.AmountIDR,
			InvoiceNum: invoiceNumber,
		}))
}

// compile-time guard: the concrete publisher satisfies RealtimeEmitter.
var _ RealtimeEmitter = (*realtime.Publisher)(nil)
