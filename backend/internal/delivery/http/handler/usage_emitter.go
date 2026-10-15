package handler

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/contracts"
)

// UsageEmitter publishes a per-user `usage_changed` event so the client can
// refresh its quota bars live. Declared as a narrow interface (rather than
// depending on the concrete realtime publisher) so tests can capture emissions
// and the handler stays free of transport dependencies.
//
// Satisfied by *realtime.Publisher.
type UsageEmitter interface {
	PublishUser(ctx context.Context, userID uuid.UUID, ev contracts.Event)
}

// Limit keys reported by the emit sites. These are the wire values the
// entitlements payload uses as `usage` map keys, so they must stay in sync with
// the counters wired in cmd/server (UsageProvidersFn.Counters).
const (
	UsageKeyMaxTransactions = "max_transactions"
	UsageKeyAITokens        = "ai_tokens_per_cycle"
	UsageKeyAIRequests      = "ai_requests_per_cycle"
	UsageKeyOCRScans        = "ocr_scans_per_cycle"
	UsageKeyVoicePerDay     = "voice_requests_per_day"
)

// emitUsageChanged publishes a usage_changed event naming the limit keys whose
// consumption moved. Best-effort by design: a nil emitter is a no-op, and the
// event is only a hint — GET /me/entitlements remains the authoritative source,
// so a dropped event costs freshness, never correctness.
func emitUsageChanged(ctx context.Context, e UsageEmitter, userID uuid.UUID, reason string, keys ...string) {
	if e == nil || len(keys) == 0 {
		return
	}
	e.PublishUser(ctx, userID, contracts.NewEvent(contracts.EventUsageChanged, time.Now().Unix(),
		contracts.UsageChangedPayload{LimitKeys: keys, Reason: reason}))
}
