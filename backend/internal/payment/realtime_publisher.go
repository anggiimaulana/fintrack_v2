package payment

import (
	"context"
	"encoding/json"

	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
)

// UserEventsChannel returns the Redis pub/sub channel for a user's realtime
// events (consumed by the SSE hub).
func UserEventsChannel(userID string) string { return "user_events:" + userID }

// PaymentEvent is the payload pushed to the user when a payment settles.
type PaymentEvent struct {
	Type               string `json:"type"`
	PaymentID          string `json:"payment_id"`
	InvoiceNumber      string `json:"invoice_number"`
	AmountIDR          int64  `json:"amount_idr"`
	SubscriptionStatus string `json:"subscription_status"`
	UpdatedAt          string `json:"updated_at"`
}

// RealtimePublisher publishes payment events to the user's channel so the
// browser can update without a manual refresh.
type RealtimePublisher struct {
	ps *infraRedis.PubSub
}

// NewRealtimePublisher builds the publisher.
func NewRealtimePublisher(ps *infraRedis.PubSub) *RealtimePublisher {
	return &RealtimePublisher{ps: ps}
}

// PublishPaymentSettled notifies the user that a payment settled.
func (p *RealtimePublisher) PublishPaymentSettled(ctx context.Context, userID string, ev PaymentEvent) error {
	if p.ps == nil || !p.ps.Enabled() {
		return infraRedis.ErrDisabled
	}
	ev.Type = "payment.settled"
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return p.ps.Publish(ctx, UserEventsChannel(userID), string(payload))
}
