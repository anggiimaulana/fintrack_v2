package middleware

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type cacheItem struct {
	Data      interface{}
	ExpiresAt time.Time
}

type InMemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

var globalCache = &InMemoryCache{
	items: make(map[string]cacheItem),
}

func (c *InMemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, exists := c.items[key]
	if !exists || time.Now().After(item.ExpiresAt) {
		return nil, false
	}
	return item.Data, true
}

func (c *InMemoryCache) Set(key string, data interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheItem{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (c *InMemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// userPrefix returns the cache-key prefix that scopes entries to a user.
func userPrefix(userID string) string {
	return "u:" + userID + ":"
}

// cacheKey is scoped to the authenticated user so responses never leak across
// users. The key embeds the user prefix so InvalidateUserCache can match it.
func cacheKey(c *gin.Context) string {
	userID := ""
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uuid.UUID); ok {
			userID = id.String()
		}
	}
	h := sha256.Sum256([]byte(c.Request.Method + ":" + c.FullPath() + ":" + c.Request.URL.RawQuery))
	return fmt.Sprintf("%s%x", userPrefix(userID), h)
}

func CacheMiddleware(ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		key := cacheKey(c)
		if data, found := globalCache.Get(key); found {
			c.JSON(200, data)
			c.Abort()
			return
		}

		blw := &bodyLogWriter{body: make([]byte, 0), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		if c.Writer.Status() == 200 {
			var responseData interface{}
			if err := json.Unmarshal(blw.body, &responseData); err == nil {
				globalCache.Set(key, responseData, ttl)
			}
		}
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

// InvalidateCache deletes every entry whose key contains the given pattern.
func InvalidateCache(pattern string) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	for k := range globalCache.items {
		if strings.Contains(k, pattern) {
			delete(globalCache.items, k)
		}
	}
}

// InvalidateUserCache removes every cached entry for a single user.
func InvalidateUserCache(userID uuid.UUID) {
	InvalidateCache(userPrefix(userID.String()))
}

// InvalidateAllCache clears the entire cache (admin-wide mutations).
func InvalidateAllCache() {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	globalCache.items = make(map[string]cacheItem)
}

func CacheCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			globalCache.mu.Lock()
			now := time.Now()
			for k, v := range globalCache.items {
				if now.After(v.ExpiresAt) {
					delete(globalCache.items, k)
				}
			}
			globalCache.mu.Unlock()
		}
	}()
}
