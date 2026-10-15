package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// TestBillingRepo_CountAIRequestsAndVisionScans verifies the usage counters that
// back `ai_requests_per_cycle` and `ocr_scans_per_cycle`. Env-gated: skips
// without TEST_DATABASE_URL.
//
// Regression guard for GAP-01/GAP-03: before F1, vision scans never wrote to the
// ledger and there was no request counter, so both limits were bypassable.
func TestBillingRepo_CountAIRequestsAndVisionScans(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	ctx := context.Background()

	// Seed an isolated user + period so the counts are deterministic.
	userID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, role)
		VALUES ($1, $2, 'x', 'Counter Test', 'user')`,
		userID, "counter-"+userID.String()+"@test.local"); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM token_ledger WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM token_periods WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	now := time.Now().UTC()
	period, err := repo.GetOrCreatePeriod(ctx, userID, now.Add(-time.Hour).Unix(), now.Add(24*time.Hour).Unix(), 1000, 1)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Idempotency keys are globally unique; scope them to this user so repeated
	// runs (or parallel tests) never collide.
	key := func(suffix string) string { return "count-test:" + userID.String() + ":" + suffix }
	insert := func(modality string, entryType billing.LedgerEntryType, idemKey string) {
		t.Helper()
		if err := repo.InsertLedger(ctx, &billing.TokenLedgerEntry{
			PeriodID:       period.ID,
			UserID:         userID,
			EntryType:      entryType,
			Amount:         10,
			Modality:       modality,
			IdempotencyKey: idemKey,
		}); err != nil {
			t.Fatalf("insert ledger %s/%s: %v", modality, entryType, err)
		}
	}

	// Two chat reserves, one vision reserve, one vision commit.
	insert(string(billing.ModalityChat), billing.EntryReserve, key("c1"))
	insert(string(billing.ModalityChat), billing.EntryReserve, key("c2"))
	insert(string(billing.ModalityVision), billing.EntryReserve, key("v1"))
	insert(string(billing.ModalityVision), billing.EntryCommit, key("v1c"))

	// ai_requests_per_cycle counts every reserve (one per AI call).
	reqs, err := repo.CountAIRequestsInPeriod(ctx, userID, now.Unix())
	if err != nil {
		t.Fatalf("count ai requests: %v", err)
	}
	if reqs != 3 {
		t.Fatalf("ai requests = %d, want 3", reqs)
	}

	// ocr_scans_per_cycle counts vision RESERVES only (one per scan). Counting
	// every vision row would double-count (reserve + commit) — regression guard.
	scans, err := repo.CountVisionScansInPeriod(ctx, userID, now.Unix())
	if err != nil {
		t.Fatalf("count vision scans: %v", err)
	}
	if scans != 1 {
		t.Fatalf("vision scans = %d, want 1 (reserve only, not reserve+commit)", scans)
	}

	// A second completed scan (reserve + commit) must still count once.
	insert(string(billing.ModalityVision), billing.EntryReserve, key("v2"))
	insert(string(billing.ModalityVision), billing.EntryCommit, key("v2c"))
	scans, err = repo.CountVisionScansInPeriod(ctx, userID, now.Unix())
	if err != nil {
		t.Fatalf("count vision scans (2): %v", err)
	}
	if scans != 2 {
		t.Fatalf("vision scans = %d, want 2 after two completed scans", scans)
	}

	// A user with no period has zero usage (not an error).
	other := uuid.New()
	zero, err := repo.CountAIRequestsInPeriod(ctx, other, now.Unix())
	if err != nil {
		t.Fatalf("count for unknown user: %v", err)
	}
	if zero != 0 {
		t.Fatalf("unknown user requests = %d, want 0", zero)
	}
}
