package middleware

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/pkg/logger"
	"github.com/fintrack/backend/pkg/response"
)

// UsageResolver returns the current numeric usage for a limit key (e.g. count
// of transactions). Used by WithinLimit middleware.
type UsageResolver func(ctx context.Context, userID uuid.UUID, limitKey string) (int64, error)

// UpgradeURL is where the frontend should send users who hit a plan gate.
const UpgradeURL = "/user/settings/subscription"

// EntitlementMiddleware enforces feature flags and numeric limits at the
// backend (the authoritative layer). super_admin bypasses all checks.
type EntitlementMiddleware struct {
	resolver *entitlement.Resolver
	// enabled is false when ENTITLEMENT_ENFORCEMENT=off: the middleware becomes a
	// no-op (incident rollback valve, see the P07 entitlement spec §3.5).
	enabled bool
	// warned tracks whether the disabled-mode warning was already logged, so we
	// log once rather than on every request.
	warned atomic.Bool
}

// NewEntitlementMiddleware builds the middleware. Enforcement is on unless
// ENTITLEMENT_ENFORCEMENT is explicitly "off"/"false"/"0".
func NewEntitlementMiddleware(resolver *entitlement.Resolver) *EntitlementMiddleware {
	return &EntitlementMiddleware{resolver: resolver, enabled: EnforcementEnabled()}
}

// EnforcementEnabled reads ENTITLEMENT_ENFORCEMENT (default on).
func EnforcementEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ENTITLEMENT_ENFORCEMENT"))) {
	case "off", "false", "0", "no":
		return false
	default:
		return true
	}
}

// skip reports whether this request should bypass enforcement.
func (m *EntitlementMiddleware) skip(c *gin.Context) bool {
	if m.enabled {
		return false
	}
	if m.warned.CompareAndSwap(false, true) {
		logger.Warn("ENTITLEMENT_ENFORCEMENT is off: plan checks are disabled")
	}
	return true
}

// isSuperAdmin reports whether the request actor bypasses entitlement checks.
func isSuperAdmin(c *gin.Context) bool {
	if v, ok := c.Get("role"); ok {
		if r, ok := v.(string); ok && r == "super_admin" {
			return true
		}
	}
	return false
}

// RequireFeature blocks the request (403) unless the user's plan enables the
// feature. The response includes the feature key and an upsell hint.
func (m *EntitlementMiddleware) RequireFeature(featureKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.skip(c) || isSuperAdmin(c) {
			c.Next()
			return
		}
		userID, ok := currentUserID(c)
		if !ok {
			response.Unauthorized(c, "Missing authentication")
			c.Abort()
			return
		}

		res, err := m.resolver.Resolve(c.Request.Context(), userID)
		if err != nil {
			response.InternalError(c, "Failed to resolve entitlements")
			c.Abort()
			return
		}
		if !res.Feature(featureKey) {
			c.JSON(403, gin.H{
				"success":      false,
				"error":        "feature_not_in_plan",
				"message":      "Fitur ini tidak tersedia pada paket Anda. Silakan tingkatkan paket.",
				"feature_key":  featureKey,
				"upgrade_hint": true,
				"upgrade_url":  UpgradeURL,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// WithinLimit blocks the request (402) when the user's usage reaches the plan
// limit for limitKey. Unlimited (-1) always passes.
func (m *EntitlementMiddleware) WithinLimit(limitKey string, usage UsageResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.skip(c) || isSuperAdmin(c) {
			c.Next()
			return
		}
		userID, ok := currentUserID(c)
		if !ok {
			response.Unauthorized(c, "Missing authentication")
			c.Abort()
			return
		}

		res, err := m.resolver.Resolve(c.Request.Context(), userID)
		if err != nil {
			response.InternalError(c, "Failed to resolve entitlements")
			c.Abort()
			return
		}

		limit := res.Limit(limitKey)
		if limit == 0 {
			// Limit key absent -> deny (fail-closed).
			c.JSON(402, gin.H{
				"success": false, "error": "limit_not_in_plan",
				"message": "Batas ini tidak tersedia pada paket Anda.", "limit_key": limitKey,
				"upgrade_hint": true, "upgrade_url": UpgradeURL,
			})
			c.Abort()
			return
		}
		if limit < 0 {
			c.Next() // unlimited
			return
		}

		var current int64
		if usage != nil {
			cur, uerr := usage(c.Request.Context(), userID, limitKey)
			if uerr != nil {
				response.InternalError(c, "Failed to evaluate usage")
				c.Abort()
				return
			}
			current = cur
		}

		if current >= limit {
			c.JSON(402, gin.H{
				"success":       false,
				"error":         "limit_exceeded",
				"message":       "Anda telah mencapai batas paket Anda. Silakan tingkatkan paket.",
				"limit_key":     limitKey,
				"limit_value":   limit,
				"current_usage": current,
				"upgrade_hint":  true,
				"upgrade_url":   UpgradeURL,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func currentUserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

var _ = strconv.Itoa
