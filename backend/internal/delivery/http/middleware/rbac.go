package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/pkg/response"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Forbidden(c, "Admin access required")
			c.Abort()
			return
		}
		r, _ := role.(string)
		if r != "admin" && r != SuperAdminRole {
			response.Forbidden(c, "Admin access required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireUser guards end-user routes. It admits every authenticated non-guest
// role: plain users, admins, and super_admins. Excluding super_admin here (as
// this function previously did) locked platform owners out of the product
// surface entirely — transactions, dashboard, AI chat and reports all 403'd for
// them. Mirrors RequireAdmin, which likewise accepts both admin roles.
func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Forbidden(c, "User access required")
			c.Abort()
			return
		}
		switch role.(string) {
		case "user", "admin", SuperAdminRole:
			c.Next()
		default:
			response.Forbidden(c, "User access required")
			c.Abort()
		}
	}
}
