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
}

// NewEntitlementHandler builds the handler.
func NewEntitlementHandler(resolver *entitlement.Resolver) *EntitlementHandler {
	return &EntitlementHandler{resolver: resolver}
}

// Me returns the authenticated user's effective entitlements (features, limits,
// plan, status, version). The frontend mirrors this for UX; the backend remains
// authoritative.
func (h *EntitlementHandler) Me(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	res, err := h.resolver.Resolve(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Failed to resolve entitlements")
		return
	}
	response.OK(c, "Entitlements retrieved successfully", res)
}
