package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/contracts"
)

// captureEmitter records every event published through the UsageEmitter
// interface, so tests can assert on the emission without a realtime transport.
type captureEmitter struct {
	events []struct {
		userID uuid.UUID
		ev     contracts.Event
	}
}

func (c *captureEmitter) PublishUser(_ context.Context, userID uuid.UUID, ev contracts.Event) {
	c.events = append(c.events, struct {
		userID uuid.UUID
		ev     contracts.Event
	}{userID, ev})
}

func TestEmitUsageChanged(t *testing.T) {
	userID := uuid.New()

	t.Run("publishes usage_changed with the named limit keys", func(t *testing.T) {
		em := &captureEmitter{}
		emitUsageChanged(context.Background(), em, userID, "transaction_created", UsageKeyMaxTransactions)

		if len(em.events) != 1 {
			t.Fatalf("got %d events, want 1", len(em.events))
		}
		got := em.events[0]
		if got.userID != userID {
			t.Errorf("event routed to %s, want %s", got.userID, userID)
		}
		if got.ev.Event != contracts.EventUsageChanged {
			t.Errorf("event name = %q, want %q", got.ev.Event, contracts.EventUsageChanged)
		}

		var payload contracts.UsageChangedPayload
		if err := json.Unmarshal(got.ev.Payload, &payload); err != nil {
			t.Fatalf("payload unmarshal: %v", err)
		}
		if payload.Reason != "transaction_created" {
			t.Errorf("reason = %q, want transaction_created", payload.Reason)
		}
		if len(payload.LimitKeys) != 1 || payload.LimitKeys[0] != UsageKeyMaxTransactions {
			t.Errorf("limit keys = %v, want [%s]", payload.LimitKeys, UsageKeyMaxTransactions)
		}
	})

	t.Run("is a no-op with a nil emitter", func(t *testing.T) {
		// Must not panic — a deployment without realtime passes nil.
		emitUsageChanged(context.Background(), nil, userID, "reason", UsageKeyAITokens)
	})

	t.Run("is a no-op without any limit key", func(t *testing.T) {
		em := &captureEmitter{}
		emitUsageChanged(context.Background(), em, userID, "reason")
		if len(em.events) != 0 {
			t.Fatalf("got %d events, want 0 (an empty key list is not worth an event)", len(em.events))
		}
	})
}
