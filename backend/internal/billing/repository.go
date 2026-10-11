package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PlanRepository reads plans, entitlements, and limits.
type PlanRepository interface {
	GetPlanByCode(ctx context.Context, code string) (*Plan, error)
	GetPlanByID(ctx context.Context, id uuid.UUID) (*Plan, error)
	ListActivePlans(ctx context.Context) ([]*Plan, error)
	GetEntitlements(ctx context.Context, planID uuid.UUID) (map[string]bool, error)
	GetLimits(ctx context.Context, planID uuid.UUID) (map[string]int64, error)
}

// SubscriptionRepository persists subscriptions and their event log.
type SubscriptionRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	Create(ctx context.Context, sub *Subscription) error
	Update(ctx context.Context, sub *Subscription) error
	RecordEvent(ctx context.Context, ev *SubscriptionEvent) error
	ListEvents(ctx context.Context, userID uuid.UUID, limit int) ([]*SubscriptionEvent, error)
}

// SubscriptionEvent is an append-only lifecycle audit row.
type SubscriptionEvent struct {
	ID             uuid.UUID      `json:"id"`
	SubscriptionID uuid.UUID      `json:"subscription_id"`
	UserID         uuid.UUID      `json:"user_id"`
	EventType      string         `json:"event_type"`
	PreviousState  map[string]any `json:"previous_state"`
	NewState       map[string]any `json:"new_state"`
	Reason         *string        `json:"reason,omitempty"`
	ActorID        *uuid.UUID     `json:"actor_id,omitempty"`
	CreatedAt      string         `json:"created_at,omitempty"`
}

// TokenRepository performs token-period and ledger mutations. The write methods
// accept a transaction-aware context (see postgres.TxManager) so callers can
// hold a short row lock around reserve/commit/release.
type TokenRepository interface {
	// GetOrCreatePeriod returns the token period for the user's current cycle,
	// creating it with the given quota when absent.
	GetOrCreatePeriod(ctx context.Context, userID uuid.UUID, periodStart, periodEnd int64, quota int64) (*TokenPeriod, error)
	// GetPeriodForUpdate locks the current period row (SELECT ... FOR UPDATE).
	GetPeriodForUpdate(ctx context.Context, userID uuid.UUID, nowUnix int64) (*TokenPeriod, error)
	// GetCurrentPeriod reads the current period WITHOUT taking a row lock, for
	// pre-flight checks (e.g. entitlement usage). Returns (nil, nil) when the
	// user has no period yet — callers treat that as zero usage.
	GetCurrentPeriod(ctx context.Context, userID uuid.UUID, nowUnix int64) (*TokenPeriod, error)
	// AdjustPeriod applies deltas to used/reserved tokens atomically.
	AdjustPeriod(ctx context.Context, periodID uuid.UUID, usedDelta, reservedDelta int64) error
	// InsertLedger appends a ledger entry; duplicate idempotency keys return
	// ErrDuplicateIdempotencyKey.
	InsertLedger(ctx context.Context, e *TokenLedgerEntry) error
	// GetLedgerByIdempotencyKey returns an existing entry (for idempotent replay).
	GetLedgerByIdempotencyKey(ctx context.Context, key string) (*TokenLedgerEntry, error)
	// GetLedgerByID returns a ledger entry by primary key (reservation lookup).
	GetLedgerByID(ctx context.Context, id uuid.UUID) (*TokenLedgerEntry, error)
	// IsReservationResolved reports whether a commit/release exists for the
	// reservation (prevents double resolution).
	IsReservationResolved(ctx context.Context, reservationID uuid.UUID) (bool, error)
	// ListExpiredReservations returns reserve entries older than the cutoff that
	// have no matching commit/release (for the auto-release worker).
	ListExpiredReservations(ctx context.Context, cutoff time.Time) ([]*TokenLedgerEntry, error)
}

// PricingRepository reads pricing rules.
type PricingRepository interface {
	GetActiveRule(ctx context.Context) (*PricingRule, error)
}
