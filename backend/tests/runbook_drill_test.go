package tests

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/internal/infrastructure/crypto"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/internal/repository/postgres"
)

// TestP08_012_RunbookOperasionalDrill tests RUNBOOK-03 (Zero-downtime master key rotation)
// and RUNBOOK-04 (Cross-pod surgical cache invalidation via Redis Pub/Sub without restart).
func TestP08_012_RunbookOperasionalDrill(t *testing.T) {
	// ─── 1. RUNBOOK-03: Rotasi Master Key (Re-encryption Drill) ────────
	oldKey := []byte("01234567890123456789012345678901") // 32 bytes
	newKey := []byte("newkeynewkeynewkeynewkeynewkey12") // 32 bytes

	originalSecret := "sk-live-provider-key-secret-999"

	// Step a: Encrypt with old master key
	oldCiphertext, err := crypto.Encrypt([]byte(originalSecret), oldKey)
	if err != nil {
		t.Fatalf("encrypt with oldKey failed: %v", err)
	}

	// Step b: Re-encryption procedure (read with oldKey -> re-encrypt with newKey)
	decryptedSecret, err := crypto.Decrypt(oldCiphertext, oldKey)
	if err != nil {
		t.Fatalf("decrypt with oldKey failed: %v", err)
	}
	if string(decryptedSecret) != originalSecret {
		t.Fatalf("decrypted secret mismatch: got %s, want %s", string(decryptedSecret), originalSecret)
	}

	newCiphertext, err := crypto.Encrypt(decryptedSecret, newKey)
	if err != nil {
		t.Fatalf("re-encrypt with newKey failed: %v", err)
	}

	// Step c: Verify new ciphertext only decrypts with newKey
	recoveredWithNewKey, err := crypto.Decrypt(newCiphertext, newKey)
	if err != nil {
		t.Fatalf("decrypt with newKey failed: %v", err)
	}
	if string(recoveredWithNewKey) != originalSecret {
		t.Fatalf("recovered secret with newKey mismatch: got %s, want %s", string(recoveredWithNewKey), originalSecret)
	}

	// Must fail with oldKey
	if _, err := crypto.Decrypt(newCiphertext, oldKey); err == nil {
		t.Fatal("expected oldKey to fail on new ciphertext, but decrypt succeeded")
	}

	t.Log("RUNBOOK-03 Drill PASSED: Master key rotation & re-encryption verified without data loss.")

	// ─── 2. RUNBOOK-04: Cross-Pod Cache Invalidation via Redis Pub/Sub ─
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6380/0"
	}
	redisClient, err := infraRedis.New(context.Background(), infraRedis.Config{URL: redisURL})
	if err != nil || !redisClient.Enabled() {
		t.Skip("Redis not available; skipping Redis pubsub runbook drill")
	}

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live DB part of runbook drill")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer pool.Close()

	seedPlanCatalogue(t, pool)
	billingRepo := postgres.NewBillingRepo(pool)
	entSrc := postgres.NewEntitlementSource(billingRepo)

	// Simulate Pod 1 and Pod 2 (two separate in-memory resolver instances)
	cacheEngine := infraRedis.NewCacheEngine(redisClient)
	resolverPod1 := entitlement.NewResolver(entSrc, cacheEngine)
	resolverPod2 := entitlement.NewResolver(entSrc, cacheEngine)

	broadcasterPod1 := entitlement.NewBroadcaster(infraRedis.NewPubSub(redisClient), resolverPod1)
	broadcasterPod2 := entitlement.NewBroadcaster(infraRedis.NewPubSub(redisClient), resolverPod2)

	stopPod2 := broadcasterPod2.StartSubscriber(ctx)
	defer stopPod2()

	// Provision user on free plan
	userID := uuid.New()
	userEmail := "runbook-cache@fintrack.test"
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, role)
		 VALUES ($1, $2, 'dummy', 'Runbook User', 'user')`,
		userID, userEmail); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	}()

	// Pod 2 resolves and caches user entitlements in L1
	resPod2, err := resolverPod2.Resolve(ctx, userID)
	if err != nil {
		t.Fatalf("pod2 resolve: %v", err)
	}
	if resPod2.PlanCode != "free" {
		t.Fatalf("expected initial free plan on pod2, got %s", resPod2.PlanCode)
	}

	// Now Super Admin on Pod 1 upgrades user to pro plan in DB
	now := time.Now().UTC()
	subID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id, status, current_period_start, current_period_end)
		 VALUES ($1, $2, $3, 'active', $4, $5)`,
		subID, userID, proPlanID, now.Add(-time.Hour), now.Add(30*24*time.Hour)); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	// Pod 1 broadcasts invalidation for this user via Redis Pub/Sub
	var invalidationHandled atomic.Bool
	broadcasterPod1.InvalidateUser(ctx, userID)

	// Allow Redis message propagation (< 100ms)
	time.Sleep(150 * time.Millisecond)

	// Pod 2 resolves again: L1 should have been invalidated by Redis Pub/Sub subscriber,
	// immediately picking up the new "pro" plan from DB without pod restart!
	updatedPod2, err := resolverPod2.Resolve(ctx, userID)
	if err != nil {
		t.Fatalf("pod2 resolve after invalidation: %v", err)
	}

	if updatedPod2.PlanCode != "pro" {
		t.Errorf("RUNBOOK-04 Failed! Pod2 still returned stale cached plan %q instead of %q",
			updatedPod2.PlanCode, "pro")
	} else {
		invalidationHandled.Store(true)
		t.Logf("RUNBOOK-04 Drill PASSED: Pod 2 instantly synchronized entitlement update (plan=%s) via Redis Pub/Sub without restart.",
			updatedPod2.PlanCode)
	}

	if !invalidationHandled.Load() {
		t.Fatal("Cross-pod invalidation drill did not complete successfully")
	}
}
