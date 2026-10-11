package audit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPostgresRepository_InsertBatch is env-gated on TEST_DATABASE_URL (or
// DATABASE_URL). It verifies real persistence into the append-only audit_logs
// table and that masking is applied end-to-end.
func TestPostgresRepository_InsertBatch(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set — skipping live audit integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, Config{Buffer: 16, BatchSize: 2, FlushInterval: 10 * time.Millisecond})

	marker := "audit-itest-" + uuid.NewString()
	svc.Log(ctx, Event{
		Action:   "create",
		Entity:   "itest",
		EntityID: marker,
		After:    map[string]any{"alias": "a", "api_key": "sk-should-be-masked"},
	})
	svc.Close()

	// Read back.
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE entity_id = $1`, marker).Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 persisted audit row, got %d", count)
	}

	// Verify masking persisted (api_key redacted).
	var after map[string]any
	if err := pool.QueryRow(ctx,
		`SELECT after FROM audit_logs WHERE entity_id = $1`, marker).Scan(&after); err != nil {
		t.Fatalf("scan after: %v", err)
	}
	if after["api_key"] != redacted {
		t.Fatalf("expected masked api_key, got %v", after["api_key"])
	}
	if after["alias"] != "a" {
		t.Fatalf("expected alias preserved, got %v", after["alias"])
	}
}
