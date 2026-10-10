package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.Int("size", c.Writer.Size()),
		}

		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Stringer("user_id", userID.(uuid.UUID)))
		}

		switch {
		case status >= 500:
			logger.Error("HTTP request error", fields...)
		case status >= 400:
			logger.Warn("HTTP request warning", fields...)
		default:
			logger.Info("HTTP request", fields...)
		}
	}
}
