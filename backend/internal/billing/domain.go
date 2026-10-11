// Package billing contains the subscription & token core domain: plans,
// subscriptions, token periods/ledger, and dynamic pricing. All monetary values
// are integer Rupiah (int64); token counts are int64. No floating point is used
// for money.
package billing

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ─── Enums ────────────────────────────────────────────────────

// SubscriptionStatus is the lifecycle state of a subscription.
type SubscriptionStatus string

const (
	StatusTrialing SubscriptionStatus = "trialing"
	StatusActive   SubscriptionStatus = "active"
	StatusPastDue  SubscriptionStatus = "past_due"
	StatusGrace    SubscriptionStatus = "grace"
	StatusCanceled SubscriptionStatus = "canceled"
	StatusExpired  SubscriptionStatus = "expired"

	// StatusNone is the sentinel returned when a user has no subscription row
	// at all. It is deliberately distinct from every lifecycle state so callers
	// can tell "never subscribed (on the default plan)" apart from "a paid plan
	// that lapsed". It is not part of the lifecycle state machine.
	StatusNone SubscriptionStatus = "none"
)

// GrantsPaidAccess reports whether a subscription in this state still entitles
// the user to its plan's features.
//
// active/trialing are obviously entitled. past_due and grace are deliberate
// tolerance states — dunning reminds the user and the subscription is only
// suspended once grace lapses — so access is retained while the user is being
// asked to pay. canceled and expired confer nothing; the user falls back to the
// default plan. Unknown values fail closed.
func (s SubscriptionStatus) GrantsPaidAccess() bool {
	switch s {
	case StatusActive, StatusTrialing, StatusPastDue, StatusGrace:
		return true
	default:
		return false
	}
}

// LedgerEntryType is the kind of token ledger mutation.
type LedgerEntryType string

const (
	EntryReserve LedgerEntryType = "reserve"
	EntryCommit  LedgerEntryType = "commit"
	EntryRelease LedgerEntryType = "release"
	EntryGrant   LedgerEntryType = "grant"
	EntryAdjust  LedgerEntryType = "adjust"
)

// Modality is the AI use case that produced token consumption.
type Modality string

const (
	ModalityChat      Modality = "chat"
	ModalityVision    Modality = "vision"
	ModalityVoice     Modality = "voice"
	ModalityEmbedding Modality = "embedding"
	ModalityReasoning Modality = "reasoning"
	ModalitySystem    Modality = "system"
)

// ─── Entities ─────────────────────────────────────────────────

// Plan is a subscription tier.
type Plan struct {
	ID                uuid.UUID `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	Description       string    `json:"description,omitempty"`
	PriceMonthlyIDR   int64     `json:"price_monthly_idr"`
	PriceYearlyIDR    int64     `json:"price_yearly_idr"`
	BillingPeriodDays int       `json:"billing_period_days"`
	IsActive          bool      `json:"is_active"`
	IsPublic          bool      `json:"is_public"`
	FallbackGateway   string    `json:"fallback_gateway"`
	SortOrder         int       `json:"sort_order"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// PlanEntitlement is a boolean feature switch for a plan.
type PlanEntitlement struct {
	PlanID     uuid.UUID `json:"plan_id"`
	FeatureKey string    `json:"feature_key"`
	IsEnabled  bool      `json:"is_enabled"`
}

// PlanLimit is a numeric limit for a plan. -1 means unlimited.
type PlanLimit struct {
	PlanID     uuid.UUID `json:"plan_id"`
	LimitKey   string    `json:"limit_key"`
	LimitValue int64     `json:"limit_value"`
}

// Unlimited is the sentinel value for unlimited numeric limits.
const Unlimited int64 = -1

// IsUnlimited reports whether a limit value means unlimited.
func IsUnlimited(v int64) bool { return v < 0 }

// Subscription is a user's active subscription.
type Subscription struct {
	ID                    uuid.UUID          `json:"id"`
	UserID                uuid.UUID          `json:"user_id"`
	PlanID                uuid.UUID          `json:"plan_id"`
	Status                SubscriptionStatus `json:"status"`
	CurrentPeriodStart    time.Time          `json:"current_period_start"`
	CurrentPeriodEnd      *time.Time         `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd     bool               `json:"cancel_at_period_end"`
	CanceledAt            *time.Time         `json:"canceled_at,omitempty"`
	Gateway               string             `json:"gateway"`
	GatewaySubscriptionID *string            `json:"gateway_subscription_id,omitempty"`
	Metadata              map[string]any     `json:"metadata,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`

	// Plan is joined for convenience (not stored on the row).
	Plan *Plan `json:"plan,omitempty"`
}

// TokenPeriod tracks quota and usage for one subscription cycle.
type TokenPeriod struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	QuotaTokens    int64     `json:"quota_tokens"`
	UsedTokens     int64     `json:"used_tokens"`
	ReservedTokens int64     `json:"reserved_tokens"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Available returns the tokens still available (quota - used - reserved).
func (p *TokenPeriod) Available() int64 {
	return p.QuotaTokens - p.UsedTokens - p.ReservedTokens
}

// TokenLedgerEntry is an append-only token mutation.
type TokenLedgerEntry struct {
	ID             uuid.UUID       `json:"id"`
	PeriodID       uuid.UUID       `json:"period_id"`
	UserID         uuid.UUID       `json:"user_id"`
	ReservationID  *uuid.UUID      `json:"reservation_id,omitempty"`
	EntryType      LedgerEntryType `json:"entry_type"`
	Amount         int64           `json:"amount"`
	Modality       string          `json:"modality"`
	CostIDR        int64           `json:"cost_idr"`
	IdempotencyKey string          `json:"idempotency_key"`
	ReferenceID    *string         `json:"reference_id,omitempty"`
	Notes          *string         `json:"notes,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// PricingRule converts tokens to integer Rupiah.
type PricingRule struct {
	ID                  uuid.UUID          `json:"id"`
	RuleName            string             `json:"rule_name"`
	BasePer1kIDR        int64              `json:"base_per_1k_idr"`
	MarkupMultiplier    float64            `json:"markup_multiplier"`
	MarkupPercent       int                `json:"markup_percent"`
	ModalityMultipliers map[string]float64 `json:"modality_multipliers"`
	IsActive            bool               `json:"is_active"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

// ─── Errors ───────────────────────────────────────────────────

var (
	ErrInsufficientQuota          = errors.New("insufficient token quota")
	ErrSubscriptionNotFound       = errors.New("subscription not found")
	ErrSubscriptionExpired        = errors.New("subscription expired")
	ErrDuplicateIdempotencyKey    = errors.New("duplicate idempotency key")
	ErrInvalidStateTransition     = errors.New("invalid subscription state transition")
	ErrReservationNotFound        = errors.New("reservation not found")
	ErrReservationAlreadyResolved = errors.New("reservation already resolved")
	ErrPlanNotFound               = errors.New("plan not found")
	ErrNoPricingRule              = errors.New("no active pricing rule")
)
