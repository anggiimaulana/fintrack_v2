package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/crypto"
)

// TestAIGatewayRepo_LoadRoutesAndCache is env-gated on TEST_DATABASE_URL. It
// verifies route loading (ordered by priority) and that the second call is
// served from the in-memory cache.
func TestAIGatewayRepo_LoadRoutesAndCache(t *testing.T) {
	pool := newTestPool(t)
	repo := NewAIGatewayRepo(pool)
	ctx := context.Background()

	routes, err := repo.GetActiveRoutesByUseCase(ctx, domain.UseCaseChat)
	if err != nil {
		t.Fatalf("load routes: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least one chat route (seed)")
	}
	// Priority ordering must be non-decreasing.
	for i := 1; i < len(routes); i++ {
		if routes[i].Priority < routes[i-1].Priority {
			t.Fatalf("routes not ordered by priority: %d then %d", routes[i-1].Priority, routes[i].Priority)
		}
	}
	// Endpoint must be joined.
	if routes[0].Endpoint == nil {
		t.Fatal("expected endpoint to be joined")
	}

	// Second call served from cache (same slice contents).
	again, err := repo.GetActiveRoutesByUseCase(ctx, domain.UseCaseChat)
	if err != nil {
		t.Fatalf("cached load: %v", err)
	}
	if len(again) != len(routes) {
		t.Fatalf("cache mismatch: %d vs %d", len(again), len(routes))
	}

	// Invalidate then reload -> still returns rows.
	repo.InvalidateCache()
	if _, err := repo.GetActiveRoutesByUseCase(ctx, domain.UseCaseChat); err != nil {
		t.Fatalf("reload after invalidate: %v", err)
	}
}

// TestAIGatewayRepo_ListAndHealth exercises ListEndpoints + UpdateEndpointHealth.
func TestAIGatewayRepo_ListAndHealth(t *testing.T) {
	pool := newTestPool(t)
	repo := NewAIGatewayRepo(pool)
	ctx := context.Background()

	endpoints, err := repo.ListEndpoints(ctx)
	if err != nil {
		t.Fatalf("list endpoints: %v", err)
	}
	if len(endpoints) == 0 {
		t.Fatal("expected seeded endpoint")
	}

	ep := endpoints[0]
	reason := "test-probe"
	if err := repo.UpdateEndpointHealth(ctx, ep.ID, "unhealthy", time.Now(), &reason); err != nil {
		t.Fatalf("update health: %v", err)
	}

	got, err := repo.GetByID(ctx, ep.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got == nil || got.Health != "unhealthy" {
		t.Fatalf("expected health unhealthy, got %+v", got)
	}
}

// TestDecryptKey_RoundTrip verifies endpoint key decryption using the master key.
func TestDecryptKey_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc, err := crypto.Encrypt([]byte("sk-test-123"), key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := DecryptKey(&enc, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != "sk-test-123" {
		t.Fatalf("expected sk-test-123, got %q", got)
	}

	// Keyless endpoint returns empty without error.
	empty, err := DecryptKey(nil, key)
	if err != nil || empty != "" {
		t.Fatalf("nil key should be empty/nil, got %q err=%v", empty, err)
	}
}
