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

func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || (role.(string) != "user" && role.(string) != "admin") {
			response.Forbidden(c, "User access required")
			c.Abort()
			return
		}
		c.Next()
	}
}
