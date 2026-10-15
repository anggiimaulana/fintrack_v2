package notification

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/pkg/contracts"
)

// Emitter wraps a realtime emitter so every user-facing domain event is ALSO
// written to the durable in-app inbox (and emailed when permitted). It satisfies
// payment.RealtimeEmitter, so it can be installed in place of the raw publisher
// without touching the payment/billing call sites.
//
// The realtime publish happens first and unconditionally: the live bell update
// must never be delayed or dropped because persistence had a hiccup.
type Emitter struct {
	inner   payment.RealtimeEmitter
	service *Service
}

// NewEmitter wraps inner. A nil inner or service makes every method a no-op.
func NewEmitter(inner payment.RealtimeEmitter, service *Service) *Emitter {
	return &Emitter{inner: inner, service: service}
}

func (e *Emitter) PublishUser(ctx context.Context, userID uuid.UUID, ev contracts.Event) {
	if e.inner != nil {
		e.inner.PublishUser(ctx, userID, ev)
	}
	if e.service == nil {
		return
	}
	if msg, ok := messageForUser(userID, ev); ok {
		// Best-effort: a persistence/email failure must not break the business
		// operation that produced the event.
		_ = e.service.Notify(ctx, msg)
	}
}

// PublishAdmins forwards admin-facing events unchanged. Admin notifications are
// operational (the super-admin dashboard already surfaces them) and are not part
// of a user's inbox.
func (e *Emitter) PublishAdmins(ctx context.Context, ev contracts.Event) {
	if e.inner != nil {
		e.inner.PublishAdmins(ctx, ev)
	}
}

// messageForUser maps a user-facing event to a notification, or reports false
// when the event is not user-notifiable.
func messageForUser(userID uuid.UUID, ev contracts.Event) (Message, bool) {
	switch ev.Event {
	case contracts.EventPaymentReceived:
		var p contracts.PaymentReceivedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		return Message{
			UserID:  userID,
			Event:   ev.Event,
			Title:   "Pembayaran diterima",
			Body:    "Pembayaran Anda sebesar " + formatIDR(p.AmountIDR) + " telah kami terima.",
			Href:    "/user/subscription?tab=payment",
			Channel: ChannelTransaction,
		}, true

	case contracts.EventPlanUpdated:
		var p contracts.PlanUpdatedPayload
		_ = json.Unmarshal(ev.Payload, &p)
		body := "Paket langganan Anda telah diperbarui."
		if p.CurrentPlan != "" {
			body = "Paket Anda kini " + p.CurrentPlan + "."
		}
		return Message{
			UserID:  userID,
			Event:   ev.Event,
			Title:   "Paket diperbarui",
			Body:    body,
			Href:    "/user/subscription",
			Channel: ChannelTransaction,
		}, true

	case contracts.EventQuotaReset:
		var p contracts.QuotaResetPayload
		_ = json.Unmarshal(ev.Payload, &p)
		return Message{
			UserID:  userID,
			Event:   ev.Event,
			Title:   "Kuota direset",
			Body:    "Siklus baru dimulai. Kuota Anda telah diperbarui.",
			Href:    "/user/usage",
			Channel: ChannelTransaction,
		}, true

	default:
		// ai_endpoint_down / manual_approval_needed are admin-facing.
		return Message{}, false
	}
}

// formatIDR renders an integer Rupiah amount with dot thousands separators
// (kept local so the notification package has no formatting dependency).
func formatIDR(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	digits := strconv.FormatInt(v, 10)
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(d)
	}
	if neg {
		return "-Rp " + b.String()
	}
	return "Rp " + b.String()
}
