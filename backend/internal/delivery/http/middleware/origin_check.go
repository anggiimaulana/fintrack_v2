package middleware

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/pkg/response"
)

// OriginCheck guards state-changing requests (POST/PUT/PATCH/DELETE) by
// verifying that the request Origin (or Referer) is in the allowed set.
// Same-origin browser traffic via the Next.js proxy sends the frontend origin;
// requests with no Origin/Referer at all (curl, server-to-server, tests) are
// allowed, matching typical same-site CSRF defences for cookie auth.
func OriginCheck(allowOrigins string) gin.HandlerFunc {
	allowed := make(map[string]bool)
	allowAll := false
	for _, o := range strings.Split(allowOrigins, ",") {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
		}
		allowed[o] = true
	}

	return func(c *gin.Context) {
		switch c.Request.Method {
		case "POST", "PUT", "PATCH", "DELETE":
		default:
			c.Next()
			return
		}

		if allowAll {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			if ref := c.GetHeader("Referer"); ref != "" {
				if u, err := url.Parse(ref); err == nil {
					origin = u.Scheme + "://" + u.Host
				}
			}
		}

		if origin == "" {
			if cookie, err := c.Cookie(AccessCookieName); err == nil && cookie != "" {
				if c.GetHeader("Authorization") == "" && c.GetHeader("X-Requested-With") == "" {
					response.Forbidden(c, "Origin or CSRF protection header required for cookie-authenticated mutations")
					c.Abort()
					return
				}
			}
			c.Next()
			return
		}

		if !allowed[origin] {
			response.Forbidden(c, "Origin not allowed")
			c.Abort()
			return
		}

		c.Next()
	}
}
