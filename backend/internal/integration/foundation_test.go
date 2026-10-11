// Package integration holds end-to-end tests that exercise the platform
// foundation (P02) against live services. Each test is env-gated so the default
// `go test ./...` stays green without external dependencies:
//
//	REDIS_URL          for cache + pub/sub tests
//	TEST_DATABASE_URL  for audit + feature flag tests (falls back to DATABASE_URL)
package integration

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/audit"
	"github.com/fintrack/backend/internal/featureflag"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/pkg/pagination"
)

func redisURL() string { return os.Getenv("REDIS_URL") }
func dbURL() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return os.Getenv("DATABASE_URL")
}

// TestFoundation_CacheVersioning exercises the versioned cache end-to-end.
func TestFoundation_CacheVersioning(t *testing.T) {
	url := redisURL()
	if url == "" {
		t.Skip("REDIS_URL not set")
	}
	c, err := infraRedis.New(context.Background(), infraRedis.Config{URL: url, DialTimeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("redis connect: %v", err)
	}
	defer c.Close()

	eng := infraRedis.NewCacheEngine(c)
	ctx := context.Background()
	user := uuid.New()

	// Write under the current version.
	v0 := eng.UserVersion(ctx, user)
	key0 := infraRedis.CacheKey(user, v0, eng.GlobalVersion(ctx), "h")
	eng.Set(ctx, key0, []byte("v0"), time.Minute)
	if _, ok := eng.Get(ctx, key0); !ok {
		t.Fatal("expected cache hit before bump")
	}

	// Bump -> new key must differ and be empty.
	v1 := eng.BumpUserVersion(ctx, user)
	key1 := infraRedis.CacheKey(user, v1, eng.GlobalVersion(ctx), "h")
	if key0 == key1 {
		t.Fatal("cache key must change after version bump")
	}
	if _, ok := eng.Get(ctx, key1); ok {
		t.Fatal("expected cache miss under new version")
	}
}

// TestFoundation_AuditPersistAndMask exercises the asynchronous audit pipeline
// against a real database, asserting persistence and masking.
func TestFoundation_AuditPersistAndMask(t *testing.T) {
	dsn := dbURL()
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	svc := audit.NewService(audit.NewPostgresRepository(pool), audit.Config{
		Buffer: 16, BatchSize: 2, FlushInterval: 10 * time.Millisecond,
	})
	marker := "p02-e2e-" + uuid.NewString()
	svc.Log(ctx, audit.Event{
		Action:   "create",
		Entity:   "foundation_itest",
		EntityID: marker,
		After:    map[string]any{"alias": "x", "password": "should-be-masked"},
	})
	svc.Close()

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id = $1`, marker).Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 audit row, got %d", count)
	}

	var after map[string]any
	if err := pool.QueryRow(ctx, `SELECT after FROM audit_logs WHERE entity_id = $1`, marker).Scan(&after); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if after["password"] != "[REDACTED]" {
		t.Fatalf("expected masked password, got %v", after["password"])
	}
}

// TestFoundation_FeatureFlagEvaluation exercises the flag evaluator with the
// real repository and a Redis cache.
func TestFoundation_FeatureFlagEvaluation(t *testing.T) {
	dsn := dbURL()
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	key := "ff_e2e_" + uuid.NewString()
	if _, err := pool.Exec(ctx,
		`INSERT INTO feature_flags (key, name, is_enabled, rollout) VALUES ($1, $2, true, 100)`,
		key, "e2e"); err != nil {
		t.Fatalf("insert flag: %v", err)
	}

	repo := featureflag.NewPostgresRepository(pool)
	svc := featureflag.NewService(repo, nil) // no cache: direct DB read

	if !svc.IsEnabled(ctx, key, new(uuid.New())) {
		t.Fatal("enabled + 100% rollout flag must evaluate true")
	}

	// Disable -> false.
	if _, err := pool.Exec(ctx, `UPDATE feature_flags SET is_enabled=false WHERE key=$1`, key); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if svc.IsEnabled(ctx, key, new(uuid.New())) {
		t.Fatal("disabled flag must evaluate false")
	}
}

// TestFoundation_PaginationParsing exercises the shared table query parser.
func TestFoundation_PaginationParsing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/x?page=2&per_page=50&sort=amount&order=asc&q=kopi&filter_status=active", nil)

	q := pagination.Parse(c, pagination.Options{AllowedSort: []string{"amount", "created_at"}})
	if q.Page != 2 || q.PerPage != 50 || q.Sort != "amount" || q.Order != "asc" {
		t.Fatalf("unexpected parse: %+v", q)
	}
	if q.Search != "kopi" || q.Filters["status"] != "active" {
		t.Fatalf("unexpected search/filters: %q %v", q.Search, q.Filters)
	}
	if q.Offset() != 50 {
		t.Fatalf("expected offset 50, got %d", q.Offset())
	}
	if got := pagination.OrderByClause(q, []string{"amount"}); got != "amount ASC" {
		t.Fatalf("expected 'amount ASC', got %q", got)
	}
}
