package middleware

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"

	"github.com/fintrack/backend/pkg/response"
)

type RateLimiter struct {
	limiter *limiter.Limiter
}

func NewRateLimiter(rate limiter.Rate) *RateLimiter {
	store := memory.NewStore()
	l := limiter.New(store, rate)
	return &RateLimiter{limiter: l}
}

func NewDefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  100,
	})
}

func (rl *RateLimiter) LimitByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, err := rl.limiter.Get(c.Request.Context(), c.ClientIP())
		if err != nil {
			c.Next()
			return
		}
		setRateLimitHeaders(c.Writer, ctx)
		if ctx.Reached {
			response.TooManyRequests(c, "Too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (rl *RateLimiter) LimitByUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		key := c.ClientIP()
		if exists {
			key = userID.(uuid.UUID).String()
		}
		ctx, err := rl.limiter.Get(c.Request.Context(), key)
		if err != nil {
			c.Next()
			return
		}
		setRateLimitHeaders(c.Writer, ctx)
		if ctx.Reached {
			response.TooManyRequests(c, "Too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

func setRateLimitHeaders(w gin.ResponseWriter, ctx limiter.Context) {
	w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(ctx.Limit, 10))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(ctx.Remaining, 10))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(ctx.Reset, 10))
}

// defaultLimit reads RATE_LIMIT_DEFAULT (default 100 requests/minute).
func defaultLimit() int64 {
	if v, err := strconv.ParseInt(os.Getenv("RATE_LIMIT_DEFAULT"), 10, 64); err == nil && v > 0 {
		return v
	}
	return 100
}

func RateLimitMiddlewareByPath(rates map[string]limiter.Rate) gin.HandlerFunc {
	var mu sync.RWMutex
	stores := make(map[string]*limiter.Limiter)

	return func(c *gin.Context) {
		path := c.FullPath()
		rate, ok := rates[path]
		if !ok {
			for pattern, r := range rates {
				if strings.HasPrefix(path, strings.TrimSuffix(pattern, "*")) {
					rate = r
					ok = true
					break
				}
			}
		}
		if !ok {
			rate = limiter.Rate{Period: 1 * time.Minute, Limit: defaultLimit()}
		}

		mu.RLock()
		lim, exists := stores[path]
		mu.RUnlock()
		if !exists {
			mu.Lock()
			lim, exists = stores[path]
			if !exists {
				store := memory.NewStore()
				lim = limiter.New(store, rate)
				stores[path] = lim
			}
			mu.Unlock()
		}

		userID, exists := c.Get("user_id")
		key := c.ClientIP()
		if exists {
			key = userID.(uuid.UUID).String()
		}

		ctx, err := lim.Get(c.Request.Context(), key)
		if err != nil {
			c.Next()
			return
		}

		setRateLimitHeaders(c.Writer, ctx)
		if ctx.Reached {
			response.TooManyRequests(c, "Too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}
