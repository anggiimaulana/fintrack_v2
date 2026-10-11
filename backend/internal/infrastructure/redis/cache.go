package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CacheEngine is a versioned cache with an in-memory fallback.
//
// When a Redis client is provided and enabled, all operations go through Redis
// (shared across instances). Otherwise it falls back to a process-local map so
// the application keeps working (single-instance / tests) without Redis.
//
// Keys are user-scoped and version-tagged:
//
//	cache:u:{user_id}:v{version}:{hash}
//
// Bumping a user's version invalidates every cached entry for that user in O(1)
// without scanning keys, and propagates across instances via Redis.
type CacheEngine struct {
	client *Client

	mu       sync.RWMutex
	mem      map[string]memItem
	versions map[string]int64
}

type memItem struct {
	data      []byte
	expiresAt time.Time
}

// NewCacheEngine builds an engine. Pass a disabled/nil client to use the
// in-memory fallback only.
func NewCacheEngine(client *Client) *CacheEngine {
	return &CacheEngine{
		client:   client,
		mem:      make(map[string]memItem),
		versions: make(map[string]int64),
	}
}

// Backend reports which backend is active: "redis" or "memory".
func (e *CacheEngine) Backend() string {
	if e.client != nil && e.client.Enabled() {
		return "redis"
	}
	return "memory"
}

// Get returns the cached value for key.
func (e *CacheEngine) Get(ctx context.Context, key string) ([]byte, bool) {
	if e.Backend() == "redis" {
		b, err := e.client.Raw().Get(ctx, key).Bytes()
		if err != nil {
			return nil, false
		}
		return b, true
	}
	e.mu.RLock()
	item, ok := e.mem[key]
	e.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		return nil, false
	}
	return item.data, true
}

// Set stores val under key with the given TTL.
func (e *CacheEngine) Set(ctx context.Context, key string, val []byte, ttl time.Duration) {
	if e.Backend() == "redis" {
		_ = e.client.Raw().Set(ctx, key, val, ttl).Err()
		return
	}
	e.mu.Lock()
	e.mem[key] = memItem{data: val, expiresAt: time.Now().Add(ttl)}
	e.mu.Unlock()
}

// DeletePattern removes all keys matching the glob pattern.
func (e *CacheEngine) DeletePattern(ctx context.Context, pattern string) {
	if e.Backend() == "redis" {
		iter := e.client.Raw().Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			_ = e.client.Raw().Del(ctx, iter.Val()).Err()
		}
		return
	}
	e.mu.Lock()
	for k := range e.mem {
		if matchGlob(pattern, k) {
			delete(e.mem, k)
		}
	}
	e.mu.Unlock()
}

// GetVersion returns the current version for a scope (0 when unset).
func (e *CacheEngine) GetVersion(ctx context.Context, scope string) int64 {
	if e.Backend() == "redis" {
		n, err := e.client.Raw().Get(ctx, versionKey(scope)).Int64()
		if err != nil {
			return 0
		}
		return n
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.versions[scope]
}

// BumpVersion increments and returns the version for a scope. All previously
// cached entries for that scope become stale immediately.
func (e *CacheEngine) BumpVersion(ctx context.Context, scope string) int64 {
	if e.Backend() == "redis" {
		n, err := e.client.Raw().Incr(ctx, versionKey(scope)).Result()
		if err != nil {
			return 0
		}
		return n
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.versions[scope]++
	return e.versions[scope]
}

// BumpUserVersion invalidates all cache entries for a single user (O(1)).
func (e *CacheEngine) BumpUserVersion(ctx context.Context, userID uuid.UUID) int64 {
	return e.BumpVersion(ctx, "u:"+userID.String())
}

// UserVersion returns the current version for a user.
func (e *CacheEngine) UserVersion(ctx context.Context, userID uuid.UUID) int64 {
	return e.GetVersion(ctx, "u:"+userID.String())
}

// GlobalVersion returns the current global version (bumped to invalidate all
// cached entries across all users).
func (e *CacheEngine) GlobalVersion(ctx context.Context) int64 {
	return e.GetVersion(ctx, "global")
}

// BumpGlobalVersion invalidates every cached entry across all users.
func (e *CacheEngine) BumpGlobalVersion(ctx context.Context) int64 {
	return e.BumpVersion(ctx, "global")
}

// CleanupExpired prunes expired entries from the in-memory fallback. Redis
// entries expire via TTL and need no explicit cleanup.
func (e *CacheEngine) CleanupExpired() {
	if e.Backend() == "redis" {
		return
	}
	now := time.Now()
	e.mu.Lock()
	for k, v := range e.mem {
		if now.After(v.expiresAt) {
			delete(e.mem, k)
		}
	}
	e.mu.Unlock()
}

// CacheKey builds the canonical versioned key for a request.
func CacheKey(userID uuid.UUID, userVersion, globalVersion int64, hash string) string {
	return fmt.Sprintf("cache:u:%s:v%d:g%d:%s", userID.String(), userVersion, globalVersion, hash)
}

// UserPrefix returns the key prefix used to scope entries to a user.
func UserPrefix(userID uuid.UUID) string {
	return "cache:u:" + userID.String() + ":"
}

func versionKey(scope string) string {
	return "cache:ver:" + scope
}

// matchGlob implements a minimal glob matcher supporting '*' (the only
// wildcard we use for cache patterns).
func matchGlob(pattern, s string) bool {
	if pattern == "" {
		return s == ""
	}
	// Split by '*' and ensure the parts appear in order.
	parts := splitStar(pattern)
	if len(parts) == 1 {
		return pattern == s
	}
	// Must start with first part and end with last part.
	if parts[0] != "" && !hasPrefix(s, parts[0]) {
		return false
	}
	if last := parts[len(parts)-1]; last != "" && !hasSuffix(s, last) {
		return false
	}
	idx := 0
	for _, p := range parts {
		if p == "" {
			continue
		}
		j := indexFrom(s, p, idx)
		if j < 0 {
			return false
		}
		idx = j + len(p)
	}
	return true
}

func splitStar(s string) []string {
	var parts []string
	cur := ""
	for _, r := range s {
		if r == '*' {
			parts = append(parts, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	parts = append(parts, cur)
	return parts
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func indexFrom(s, sub string, from int) int {
	if from > len(s) {
		return -1
	}
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
