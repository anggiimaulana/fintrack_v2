// Package entitlement resolves a user's effective access rights from their
// subscription plan (features + limits). Results are cached with a version
// stamp so a plan change propagates instantly without re-login.
package entitlement

import (
	"context"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// UsageEntry is the current consumption for one limit key.
type UsageEntry struct {
	Used      int64 `json:"used"`
	Limit     int64 `json:"limit"`
	Remaining int64 `json:"remaining"`
	// Unlimited is true when the limit is the unlimited sentinel (-1).
	Unlimited bool `json:"unlimited"`
	// Unit is the display unit for this limit (e.g. "kuota"), sourced from the
	// metric catalog so the UI never hardcodes a unit that could change.
	Unit string `json:"unit,omitempty"`
}

// Period describes the active quota cycle (nil when none is tracked).
type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Resolved is the effective entitlement for a user.
type Resolved struct {
	UserID   uuid.UUID        `json:"user_id"`
	PlanID   string           `json:"plan_id,omitempty"`
	PlanCode string           `json:"plan_code"`
	PlanName string           `json:"plan_name"`
	Status   string           `json:"status"`
	Features map[string]bool  `json:"features"`
	Limits   map[string]int64 `json:"limits"`
	// Usage is the real-time consumption per limit key (F6). Additive: older
	// clients ignore it.
	Usage map[string]UsageEntry `json:"usage,omitempty"`
	// Period is the active quota cycle, when one is tracked (F6).
	Period *Period `json:"period,omitempty"`
	// Version is the user's entitlement version at resolve time.
	Version int64 `json:"version"`
	// PlanVersion is the plan's version this payload was built against; a
	// mismatch on read means the plan was edited and the payload is stale.
	PlanVersion int64 `json:"plan_version,omitempty"`
	// QuotaRatio is the plan's token→unit ratio (>= 1). Snapshotted into the
	// token period when a new cycle starts, so a change here never affects a
	// cycle already in progress.
	QuotaRatio int64 `json:"quota_ratio,omitempty"`
}

// QuotaRatioValue returns the plan's token→unit ratio, defaulting to 1 when the
// payload predates the ratio feature or carries an invalid value.
func (r *Resolved) QuotaRatioValue() int64 {
	if r == nil || r.QuotaRatio < 1 {
		return 1
	}
	return r.QuotaRatio
}

// Feature reports whether a feature is enabled.
func (r *Resolved) Feature(key string) bool {
	if r == nil || r.Features == nil {
		return false
	}
	return r.Features[key]
}

// Limit returns the numeric limit for a key (-1 = unlimited; 0 = absent/deny).
func (r *Resolved) Limit(key string) int64 {
	if r == nil || r.Limits == nil {
		return 0
	}
	if v, ok := r.Limits[key]; ok {
		return v
	}
	return 0
}

// WithinLimit reports whether `current` usage is within the limit for key.
// Unlimited (-1) always passes.
func (r *Resolved) WithinLimit(key string, current int64) bool {
	v := r.Limit(key)
	if billing.IsUnlimited(v) {
		return true
	}
	return current < v
}

// Source loads the raw data needed to resolve entitlements.
type Source interface {
	// GetActivePlanForUser returns the user's plan + subscription status.
	GetActivePlanForUser(ctx context.Context, userID uuid.UUID) (plan *billing.Plan, status billing.SubscriptionStatus, err error)
	// GetEntitlements and GetLimits return the plan's switches/limits.
	GetEntitlements(ctx context.Context, planID uuid.UUID) (map[string]bool, error)
	GetLimits(ctx context.Context, planID uuid.UUID) (map[string]int64, error)
}

// UsageSource computes real-time consumption per limit key. It is deliberately
// separate from Source and from the cache: usage changes on every AI call, so it
// must never be cached alongside the (slow-changing) plan resolution.
type UsageSource interface {
	// Usage returns consumption for the given plan limits plus the active quota
	// period (nil when no period is tracked). Implementations must be
	// best-effort: a single limit that cannot be computed is omitted, not fatal.
	Usage(ctx context.Context, userID uuid.UUID, limits map[string]int64) (map[string]UsageEntry, *Period, error)
}
