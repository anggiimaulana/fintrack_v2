package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestCacheEngine_MemorySetGet verifies basic set/get with TTL on the in-memory
// fallback (no Redis client).
func TestCacheEngine_MemorySetGet(t *testing.T) {
	e := NewCacheEngine(nil)
	ctx := context.Background()
	if e.Backend() != "memory" {
		t.Fatalf("expected memory backend, got %s", e.Backend())
	}

	e.Set(ctx, "k1", []byte("v1"), time.Minute)
	got, ok := e.Get(ctx, "k1")
	if !ok || string(got) != "v1" {
		t.Fatalf("expected v1, got %q ok=%v", got, ok)
	}

	// Expired entry must miss.
	e.Set(ctx, "k2", []byte("v2"), time.Nanosecond)
	time.Sleep(2 * time.Millisecond)
	if _, ok := e.Get(ctx, "k2"); ok {
		t.Fatal("expected expired entry to miss")
	}
}

// TestCacheEngine_UserVersionBump verifies that bumping a user's version
// changes the cache key (invalidating prior entries) without affecting others.
func TestCacheEngine_UserVersionBump(t *testing.T) {
	e := NewCacheEngine(nil)
	ctx := context.Background()
	user := uuid.New()
	other := uuid.New()

	v0 := e.UserVersion(ctx, user)
	if v0 != 0 {
		t.Fatalf("expected initial version 0, got %d", v0)
	}
	v1 := e.BumpUserVersion(ctx, user)
	if v1 != 1 {
		t.Fatalf("expected version 1 after bump, got %d", v1)
	}
	if e.UserVersion(ctx, other) != 0 {
		t.Fatal("bumping one user must not affect another")
	}

	// Keys built before/after the bump must differ.
	k0 := CacheKey(user, 0, 0, "hash")
	k1 := CacheKey(user, v1, 0, "hash")
	if k0 == k1 {
		t.Fatal("cache key must change after version bump")
	}
}

// TestCacheEngine_GlobalVersionBump verifies global invalidation semantics.
func TestCacheEngine_GlobalVersionBump(t *testing.T) {
	e := NewCacheEngine(nil)
	ctx := context.Background()
	if e.GlobalVersion(ctx) != 0 {
		t.Fatal("expected initial global version 0")
	}
	if e.BumpGlobalVersion(ctx) != 1 {
		t.Fatal("expected global version 1 after bump")
	}
}

// TestCacheEngine_DeletePattern verifies glob-based deletion on memory backend.
func TestCacheEngine_DeletePattern(t *testing.T) {
	e := NewCacheEngine(nil)
	ctx := context.Background()
	e.Set(ctx, "cache:u:a:1", []byte("x"), time.Minute)
	e.Set(ctx, "cache:u:a:2", []byte("y"), time.Minute)
	e.Set(ctx, "cache:u:b:1", []byte("z"), time.Minute)

	e.DeletePattern(ctx, "cache:u:a:*")

	if _, ok := e.Get(ctx, "cache:u:a:1"); ok {
		t.Fatal("expected cache:u:a:1 deleted")
	}
	if _, ok := e.Get(ctx, "cache:u:a:2"); ok {
		t.Fatal("expected cache:u:a:2 deleted")
	}
	if _, ok := e.Get(ctx, "cache:u:b:1"); !ok {
		t.Fatal("expected cache:u:b:1 retained")
	}
}

// TestCacheEngine_RedisBackend is env-gated: runs only when REDIS_URL is set.
func TestCacheEngine_RedisBackend(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set — skipping live Redis cache test")
	}
	c, err := New(context.Background(), Config{URL: url, DialTimeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	e := NewCacheEngine(c)
	if e.Backend() != "redis" {
		t.Fatalf("expected redis backend, got %s", e.Backend())
	}
	ctx := context.Background()
	key := "cache:test:" + uuid.NewString()

	e.Set(ctx, key, []byte("hello"), time.Minute)
	got, ok := e.Get(ctx, key)
	if !ok || string(got) != "hello" {
		t.Fatalf("expected hello, got %q ok=%v", got, ok)
	}

	// Version bump must be shared (Redis-backed).
	user := uuid.New()
	if v := e.BumpUserVersion(ctx, user); v != 1 {
		t.Fatalf("expected redis version 1, got %d", v)
	}
	if v := e.UserVersion(ctx, user); v != 1 {
		t.Fatalf("expected redis version 1 read back, got %d", v)
	}

	e.DeletePattern(ctx, key)
	if _, ok := e.Get(ctx, key); ok {
		t.Fatal("expected key deleted")
	}
}
