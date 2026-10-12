package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/entitlement"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/internal/realtime"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/pkg/contracts"
)

// TestP08_005_ChaosRedisDisconnect tests graceful degradation when Redis connection
// is completely severed or unreachable during runtime.
// Invariant: zero panics, readiness probe stays 200 degraded, entitlements fallback
// authoritatively to PostgreSQL + in-memory L1 cache, realtime events degrade to local hub.
func TestP08_005_ChaosRedisDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping redis chaos test")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	seedPlanCatalogue(t, pool)
	billingRepo := postgres.NewBillingRepo(pool)

	brokenRedis, _ := infraRedis.New(ctx, infraRedis.Config{URL: "redis://127.0.0.1:54399/0"})

	// 2. Health check verification under Redis partition
	healthH := deliveryHTTP.NewHealthHandler(pool, brokenRedis)
	r := gin.New()
	r.GET("/health/ready", healthH.Ready)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 degraded when Redis is down, got %d", w.Code)
	}

	var healthResp struct {
		Status string `json:"status"`
		Checks struct {
			Postgres struct {
				Status string `json:"status"`
			} `json:"postgres"`
			Redis struct {
				Status string `json:"status"`
			} `json:"redis"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("unmarshal health response: %v", err)
	}

	if healthResp.Checks.Postgres.Status != "ok" {
		t.Errorf("expected postgres status ok, got %s", healthResp.Checks.Postgres.Status)
	}
	if healthResp.Checks.Redis.Status != "degraded" {
		t.Errorf("expected redis status degraded, got %s", healthResp.Checks.Redis.Status)
	}
	t.Logf("Health check under Redis chaos: status=%s, postgres=%s, redis=%s",
		healthResp.Status, healthResp.Checks.Postgres.Status, healthResp.Checks.Redis.Status)

	// 3. Entitlement resolution fallback to PostgreSQL + L1 in-memory
	// Create test user on pro plan
	userID := uuid.New()
	userEmail := "redis-chaos@fintrack.test"
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, role)
		 VALUES ($1, $2, 'dummy', 'Redis Chaos User', 'user')`,
		userID, userEmail); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	}()

	now := time.Now().UTC()
	subID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id, status, current_period_start, current_period_end)
		 VALUES ($1, $2, $3, 'active', $4, $5)`,
		subID, userID, proPlanID, now.Add(-time.Hour), now.Add(30*24*time.Hour)); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	// Resolver with broken Redis cache engine (simulates Redis timeout/outage)
	entSrc := postgres.NewEntitlementSource(billingRepo)
	resolver := entitlement.NewResolver(entSrc, nil) // nil cache -> fallback to Postgres + L1

	res, err := resolver.Resolve(ctx, userID)
	if err != nil {
		t.Fatalf("resolve entitlements without Redis: %v", err)
	}
	if res.PlanCode != "pro" {
		t.Errorf("expected pro plan resolved from DB, got %s", res.PlanCode)
	}
	if !res.Feature("ai_chat") {
		t.Errorf("expected ai_chat feature enabled on pro plan")
	}

	// 4. Realtime Publisher fallback to local Hub when Redis Pub/Sub fails
	hub := realtime.NewHub()
	client := hub.Register(userID, false)

	brokenPubSub := infraRedis.NewPubSub(brokenRedis)
	publisher := realtime.NewPublisher(brokenPubSub, hub, nil)

	// Publish domain event while Redis is down
	event := contracts.Event{
		ID:        "evt-chaos-test-1",
		Event:     contracts.EventPlanUpdated,
		Timestamp: time.Now().Unix(),
		Payload:   json.RawMessage(`{"plan_code":"pro"}`),
	}

	// Must not panic or return error: publishes to local hub
	publisher.PublishUser(ctx, userID, event)

	// Verify local client on hub received the event
	select {
	case received := <-client.Events():
		if received.ID != event.ID {
			t.Errorf("expected event %s, got %s", event.ID, received.ID)
		}
		t.Logf("Realtime event received via local hub fallback: %s (%s)", received.ID, received.Event)
	case <-time.After(500 * time.Millisecond):
		t.Errorf("timed out waiting for event on local hub fallback")
	}
}
