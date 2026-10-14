package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/pkg/response"
)

// EntitlementHandler exposes the caller's effective entitlements.
type EntitlementHandler struct {
	resolver *entitlement.Resolver
	usage    entitlement.UsageSource
}

// NewEntitlementHandler builds the handler.
func NewEntitlementHandler(resolver *entitlement.Resolver) *EntitlementHandler {
	return &EntitlementHandler{resolver: resolver}
}

// WithUsage attaches the real-time usage source (F6). Without it the payload
// still carries limits/features (older behaviour).
func (h *EntitlementHandler) WithUsage(u entitlement.UsageSource) *EntitlementHandler {
	h.usage = u
	return h
}

// Me returns the authenticated user's effective entitlements (features, limits,
// plan, status, version) plus real-time usage (F6). The frontend mirrors this
// for UX; the backend remains authoritative.
//
// Usage is merged onto a shallow copy AFTER resolution so it never enters the
// entitlement cache (which would freeze consumption at the resolve moment).
func (h *EntitlementHandler) Me(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	res, err := h.resolver.Resolve(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Failed to resolve entitlements")
		return
	}

	if h.usage != nil {
		usage, period, uerr := h.usage.Usage(c.Request.Context(), userID, res.Limits)
		if uerr == nil {
			// Copy so the cached Resolved is never mutated.
			merged := *res
			merged.Usage = usage
			merged.Period = period
			response.OK(c, "Entitlements retrieved successfully", &merged)
			return
		}
	}

	response.OK(c, "Entitlements retrieved successfully", res)
}
