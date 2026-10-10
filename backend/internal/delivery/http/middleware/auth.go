package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/infrastructure/auth"
	"github.com/fintrack/backend/pkg/response"
)

// Cookie names for the access and refresh tokens.
const (
	AccessCookieName  = "ft_at"
	RefreshCookieName = "ft_rt"
)

// AuthMiddleware authenticates requests using the httpOnly `ft_at` cookie,
// falling back to an `Authorization: Bearer <token>` header (for curl/tests).
// On success it sets `user_id` and `role` in the gin context.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	secret := []byte(jwtSecret)

	return func(c *gin.Context) {
		tokenString := ""

		if cookie, err := c.Cookie(AccessCookieName); err == nil && cookie != "" {
			tokenString = cookie
		} else if header := c.GetHeader("Authorization"); header != "" {
			parts := strings.SplitN(header, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			response.Unauthorized(c, "Missing authentication token")
			c.Abort()
			return
		}

		claims, err := auth.ParseAccessToken(tokenString, secret)
		if err != nil {
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.Sub)
		if err != nil {
			response.Unauthorized(c, "Invalid user ID in token")
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("role", claims.Role)
		c.Set("email", claims.Email)

		c.Next()
	}
}
