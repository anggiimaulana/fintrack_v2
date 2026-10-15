package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// TestLedger_RatioConversion proves the core of the quota-ratio feature: raw
// provider tokens are divided by the cycle's snapshot ratio before being charged
// to the plan quota, while cost stays based on raw tokens.
func TestLedger_RatioConversion(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@ratio.local", "Ratio"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	// Quota 5000 units, ratio 5.
	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: 5000, days: 30, ratio: 5}, nil)

	// I-4: reserve returns UNITs (ceil(3809/5) = 762), not raw tokens.
	res, err := ledger.Reserve(ctx, userID, 3809, billing.ModalityChat, "ratio-res-"+userID.String())
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if res.Amount != 762 {
		t.Fatalf("reserve must be in UNITS: got %d want 762", res.Amount)
	}

	// Commit 9945 raw tokens -> ceil(9945/5) = 1989 units.
	if err := ledger.Commit(ctx, userID, res.ID, 9945, "ratio-com-"+userID.String()); err != nil {
		t.Fatalf("commit: %v", err)
	}

	period, err := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if err != nil || period == nil {
		t.Fatalf("period lookup: %v", err)
	}
	if period.QuotaRatio != 5 {
		t.Fatalf("period ratio snapshot = %d, want 5", period.QuotaRatio)
	}
	if period.UsedTokens != 1989 {
		t.Fatalf("used units = %d, want 1989", period.UsedTokens)
	}
	if period.ReservedTokens != 0 {
		t.Fatalf("reserved after commit = %d, want 0", period.ReservedTokens)
	}
	// The user can still chat: 5000 - 1989 = 3011 units left.
	if period.Available() != 3011 {
		t.Fatalf("available = %d, want 3011", period.Available())
	}

	// The ledger row records BOTH units and raw tokens.
	entry, err := repo.GetLedgerByIdempotencyKey(ctx, "ratio-com-"+userID.String())
	if err != nil || entry == nil {
		t.Fatalf("ledger lookup: %v", err)
	}
	if entry.Amount != 1989 {
		t.Fatalf("entry units = %d, want 1989", entry.Amount)
	}
	if entry.RawTokens != 9945 {
		t.Fatalf("entry raw_tokens = %d, want 9945", entry.RawTokens)
	}
}

// TestLedger_RatioOneIsIdentity proves backward compatibility: with ratio 1 the
// ledger behaves exactly as before the feature (units == tokens).
func TestLedger_RatioOneIsIdentity(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@id1.local", "Identity"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: 100000, days: 30, ratio: 1}, nil)

	res, err := ledger.Reserve(ctx, userID, 500, billing.ModalityChat, "id1-res-"+userID.String())
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if res.Amount != 500 {
		t.Fatalf("ratio 1 must be identity: reserve %d want 500", res.Amount)
	}
	if err := ledger.Commit(ctx, userID, res.ID, 500, "id1-com-"+userID.String()); err != nil {
		t.Fatalf("commit: %v", err)
	}
	period, _ := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if period.UsedTokens != 500 {
		t.Fatalf("used = %d, want 500", period.UsedTokens)
	}
}

// TestLedger_ReleaseFullHoldWithRatio proves I-4: Release returns the ENTIRE
// held unit amount (which was computed from raw tokens).
func TestLedger_ReleaseFullHoldWithRatio(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@rel.local", "Release"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: 5000, days: 30, ratio: 5}, nil)
	res, err := ledger.Reserve(ctx, userID, 9945, billing.ModalityChat, "rel-res-"+userID.String())
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if res.Amount != 1989 {
		t.Fatalf("held units = %d, want 1989", res.Amount)
	}
	before, _ := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if before.ReservedTokens != 1989 {
		t.Fatalf("reserved = %d, want 1989", before.ReservedTokens)
	}

	if err := ledger.Release(ctx, userID, res.ID, "rel-rel-"+userID.String()); err != nil {
		t.Fatalf("release: %v", err)
	}
	after, _ := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if after.ReservedTokens != 0 {
		t.Fatalf("reserved after release = %d, want 0", after.ReservedTokens)
	}
	if after.UsedTokens != 0 {
		t.Fatalf("release must not charge: used = %d, want 0", after.UsedTokens)
	}
}
