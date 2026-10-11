// Package entitlement resolves a user's effective access rights from their
// subscription plan (features + limits). Results are cached with a version
// stamp so a plan change propagates instantly without re-login.
package entitlement

import (
	"context"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// Resolved is the effective entitlement for a user.
type Resolved struct {
	UserID   uuid.UUID        `json:"user_id"`
	PlanID   string           `json:"plan_id,omitempty"`
	PlanCode string           `json:"plan_code"`
	PlanName string           `json:"plan_name"`
	Status   string           `json:"status"`
	Features map[string]bool  `json:"features"`
	Limits   map[string]int64 `json:"limits"`
	// Version is the user's entitlement version at resolve time.
	Version int64 `json:"version"`
	// PlanVersion is the plan's version this payload was built against; a
	// mismatch on read means the plan was edited and the payload is stale.
	PlanVersion int64 `json:"plan_version,omitempty"`
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
