package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/response"
)

// SuperAdminRole is the role name required by RequireSuperAdmin.
const SuperAdminRole = "super_admin"

// RoleResolver returns the *current* role of a user. Implementations back this
// with a short-TTL Redis cache so that raising a user's role version revokes
// privileged access instantly (K-14) without waiting for the JWT to expire.
type RoleResolver interface {
	CurrentRole(ctx context.Context, userID uuid.UUID) (string, error)
}

// RoleResolverFunc adapts a plain function to RoleResolver (used in tests).
type RoleResolverFunc func(ctx context.Context, userID uuid.UUID) (string, error)

// CurrentRole implements RoleResolver.
func (f RoleResolverFunc) CurrentRole(ctx context.Context, userID uuid.UUID) (string, error) {
	return f(ctx, userID)
}

// RequireSuperAdmin gates a route on the `super_admin` role.
//
// Defense in depth: the JWT role must already be `super_admin`, and the role
// is re-validated against the (versioned) role source so a just-demoted user
// is rejected even while their JWT is still cryptographically valid.
//
// When resolve is nil only the JWT claim is checked (useful for tests and for
// deployments without a shared cache).
func RequireSuperAdmin(resolve RoleResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Unauthorized(c, "Missing authentication context")
			c.Abort()
			return
		}
		if roleStr, _ := role.(string); roleStr != SuperAdminRole {
			response.Error(c, 403, "Super admin access required", "FORBIDDEN_SUPER_ADMIN_REQUIRED")
			c.Abort()
			return
		}

		if resolve != nil {
			userID, ok := c.MustGet("user_id").(uuid.UUID)
			if !ok {
				response.Unauthorized(c, "Invalid authentication context")
				c.Abort()
				return
			}
			current, err := resolve.CurrentRole(c.Request.Context(), userID)
			if err != nil || current != SuperAdminRole {
				response.Error(c, 403, "Super admin access required", "FORBIDDEN_SUPER_ADMIN_REQUIRED")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
