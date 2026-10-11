package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
)

// cacheEngine is the process-wide cache used by CacheMiddleware and the
// invalidation helpers. It defaults to an in-memory fallback and is upgraded to
// Redis via ConfigureCache (called during server bootstrap).
var (
	engineMu    sync.RWMutex
	cacheEngine = infraRedis.NewCacheEngine(nil)
	pubsub      *infraRedis.PubSub
)

// ConfigureCache installs a cache engine backed by the given Redis client.
// Passing nil keeps the in-memory fallback.
func ConfigureCache(client *infraRedis.Client) {
	engineMu.Lock()
	defer engineMu.Unlock()
	cacheEngine = infraRedis.NewCacheEngine(client)
	pubsub = infraRedis.NewPubSub(client)
}

// CacheBackend reports the active cache backend ("redis" or "memory").
func CacheBackend() string {
	engineMu.RLock()
	defer engineMu.RUnlock()
	return cacheEngine.Backend()
}

func currentEngine() *infraRedis.CacheEngine {
	engineMu.RLock()
	defer engineMu.RUnlock()
	return cacheEngine
}

// cacheKey builds the versioned, user-scoped cache key for a request. The key
// embeds both the user version and the global version so a single INCR
// invalidates all entries for that scope without scanning.
func cacheKey(ctx context.Context, c *gin.Context) string {
	userID := uuid.Nil
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uuid.UUID); ok {
			userID = id
		}
	}
	eng := currentEngine()
	userVer := eng.UserVersion(ctx, userID)
	globalVer := eng.GlobalVersion(ctx)
	h := sha256.Sum256([]byte(c.Request.Method + ":" + c.FullPath() + ":" + c.Request.URL.RawQuery))
	return infraRedis.CacheKey(userID, userVer, globalVer, fmt.Sprintf("%x", h))
}

// CacheMiddleware caches successful (200) GET responses for ttl. Responses are
// scoped per user (never leaked across users) and version-tagged so mutations
// invalidate them instantly.
func CacheMiddleware(ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		eng := currentEngine()
		ctx := c.Request.Context()
		key := cacheKey(ctx, c)

		if data, found := eng.Get(ctx, key); found {
			c.Header("X-Cache", "HIT")
			c.Data(200, "application/json; charset=utf-8", data)
			c.Abort()
			return
		}

		blw := &bodyLogWriter{body: make([]byte, 0), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		if c.Writer.Status() == 200 {
			var responseData interface{}
			if err := json.Unmarshal(blw.body, &responseData); err == nil {
				eng.Set(ctx, key, blw.body, ttl)
			}
		}
		c.Header("X-Cache", "MISS")
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

// InvalidateCache deletes every cached entry whose key contains the pattern.
// Prefer version bumps (InvalidateUserCache/InvalidateAllCache) which are O(1).
func InvalidateCache(ctx context.Context, pattern string) {
	currentEngine().DeletePattern(ctx, "*"+pattern+"*")
}

// InvalidateUserCache invalidates every cached entry for a single user via a
// version bump (O(1), propagates across instances through Redis) and broadcasts
// an event so peer instances can drop any local (L1) lookups.
func InvalidateUserCache(ctx context.Context, userID uuid.UUID) {
	currentEngine().BumpUserVersion(ctx, userID)
	if ps := currentPubSub(); ps != nil && ps.Enabled() {
		_ = ps.Publish(ctx, infraRedis.ChannelCacheInvalidate, "u:"+userID.String())
	}
}

// InvalidateAllCache invalidates the entire cache via a global version bump and
// broadcasts the event to peer instances.
func InvalidateAllCache(ctx context.Context) {
	currentEngine().BumpGlobalVersion(ctx)
	if ps := currentPubSub(); ps != nil && ps.Enabled() {
		_ = ps.Publish(ctx, infraRedis.ChannelCacheInvalidate, "all")
	}
}

func currentPubSub() *infraRedis.PubSub {
	engineMu.RLock()
	defer engineMu.RUnlock()
	return pubsub
}

// StartCacheInvalidationSubscriber listens for cache-invalidation events from
// peer instances and drops any process-local entries. It returns a stop
// function. When pub/sub is disabled it is a no-op (single-instance).
func StartCacheInvalidationSubscriber(ctx context.Context) (stop func()) {
	ps := currentPubSub()
	if ps == nil || !ps.Enabled() {
		return func() {}
	}
	return ps.Subscribe(ctx, infraRedis.ChannelCacheInvalidate, func(_ string) {
		// Drop local (memory) entries; Redis-backed entries are invalidated by
		// the shared version counters bumped by the originating instance.
		currentEngine().CleanupExpired()
	})
}

// CacheCleanup periodically prunes expired entries from the in-memory fallback.
// Redis entries expire via TTL and need no explicit cleanup.
func CacheCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			currentEngine().CleanupExpired()
		}
	}()
}
