package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// fixedQuota returns a resolver with a fixed cycle for tests.
type fixedQuota struct {
	quota int64
	days  int
}

func (f fixedQuota) ResolveQuota(_ context.Context, _ uuid.UUID, at time.Time) (time.Time, time.Time, int64, error) {
	start := at.Truncate(24 * time.Hour)
	return start, start.AddDate(0, 0, f.days), f.quota, nil
}

// TestLedger_ReserveCommitRelease is env-gated. It exercises the full 2-phase
// ledger against a live database.
func TestLedger_ReserveCommitRelease(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)

	rule, err := repo.GetActiveRule(context.Background())
	if err != nil {
		t.Fatalf("pricing rule: %v", err)
	}
	pricing := billing.NewPricingEngine(rule)

	// Unique user to avoid clashing with other tests.
	userID := uuid.New()
	// Insert a throwaway user row (FK requires it).
	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "ITest")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
	})

	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: 1000, days: 30}, pricing)
	ctx := context.Background()

	// 1) Reserve 300 tokens.
	res, err := ledger.Reserve(ctx, userID, 300, billing.ModalityChat, "itest-res-"+userID.String())
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if res.Amount != 300 {
		t.Fatalf("expected reserved 300, got %d", res.Amount)
	}

	period, err := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if err != nil || period == nil {
		t.Fatalf("period lookup: %v", err)
	}
	if period.ReservedTokens != 300 {
		t.Fatalf("expected reserved_tokens=300, got %d", period.ReservedTokens)
	}

	// 2) Commit 120 actual tokens -> reserved released, used = 120.
	if err := ledger.Commit(ctx, userID, res.ID, 120, "itest-com-"+res.ID.String()); err != nil {
		t.Fatalf("commit: %v", err)
	}
	period, _ = repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if period.UsedTokens != 120 {
		t.Fatalf("expected used=120, got %d", period.UsedTokens)
	}
	if period.ReservedTokens != 0 {
		t.Fatalf("expected reserved=0 after commit, got %d", period.ReservedTokens)
	}

	// 3) Commit again is a no-op (idempotent).
	if err := ledger.Commit(ctx, userID, res.ID, 120, "itest-com-"+res.ID.String()); err != nil {
		t.Fatalf("idempotent commit: %v", err)
	}
	period, _ = repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if period.UsedTokens != 120 {
		t.Fatalf("idempotent commit changed used to %d", period.UsedTokens)
	}

	// 4) Reserve then release.
	res2, err := ledger.Reserve(ctx, userID, 200, billing.ModalityVision, "itest-res2-"+userID.String())
	if err != nil {
		t.Fatalf("reserve2: %v", err)
	}
	if err := ledger.Release(ctx, userID, res2.ID, "itest-rel-"+res2.ID.String()); err != nil {
		t.Fatalf("release: %v", err)
	}
	period, _ = repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if period.ReservedTokens != 0 {
		t.Fatalf("expected reserved=0 after release, got %d", period.ReservedTokens)
	}
	if period.UsedTokens != 120 {
		t.Fatalf("release must not change used, got %d", period.UsedTokens)
	}
}

// TestLedger_RefundReturnsChargedTokens verifies a refund reduces used_tokens
// (returning tokens for work that failed after commit) and is idempotent, and
// that it can never push used_tokens below zero.
func TestLedger_RefundReturnsChargedTokens(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "ITest")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID) })

	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: 1000, days: 30}, nil)

	// Charge 400 tokens: reserve 500, commit 400.
	res, err := ledger.Reserve(ctx, userID, 500, billing.ModalityVision, "itest-refund-res-"+userID.String())
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if err := ledger.Commit(ctx, userID, res.ID, 400, "itest-refund-com-"+userID.String()); err != nil {
		t.Fatalf("commit: %v", err)
	}
	period, _ := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if period.UsedTokens != 400 {
		t.Fatalf("expected used=400 after commit, got %d", period.UsedTokens)
	}

	// Refund 400 -> used back to 0.
	refundKey := "itest-refund-" + userID.String()
	if err := ledger.Refund(ctx, userID, 400, billing.ModalityVision, refundKey); err != nil {
		t.Fatalf("refund: %v", err)
	}
	period, _ = repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if period.UsedTokens != 0 {
		t.Fatalf("expected used=0 after refund, got %d", period.UsedTokens)
	}

	// Idempotent: a repeat with the same key must not double-refund.
	if err := ledger.Refund(ctx, userID, 400, billing.ModalityVision, refundKey); err != nil {
		t.Fatalf("idempotent refund: %v", err)
	}
	period, _ = repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if period.UsedTokens != 0 {
		t.Fatalf("idempotent refund changed used to %d", period.UsedTokens)
	}
}

// TestLedger_RefundClampedToUsed verifies a refund larger than what was charged
// only returns the charged amount, so the balance never goes negative.
func TestLedger_RefundClampedToUsed(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "ITest")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID) })

	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: 1000, days: 30}, nil)

	res, err := ledger.Reserve(ctx, userID, 300, billing.ModalityChat, "itest-clamp-res-"+userID.String())
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if err := ledger.Commit(ctx, userID, res.ID, 100, "itest-clamp-com-"+userID.String()); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Ask to refund 500 though only 100 was charged -> clamp to 100.
	if err := ledger.Refund(ctx, userID, 500, billing.ModalityChat, "itest-clamp-refund-"+userID.String()); err != nil {
		t.Fatalf("refund: %v", err)
	}
	period, _ := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if period.UsedTokens != 0 {
		t.Fatalf("expected used clamped to 0, got %d", period.UsedTokens)
	}
}

// TestLedger_InsufficientQuota verifies reserve is rejected when quota is low.
func TestLedger_InsufficientQuota(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "ITest")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID) })

	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: 100, days: 30}, nil)

	if _, err := ledger.Reserve(ctx, userID, 500, billing.ModalityChat, "itest-big-"+userID.String()); err != billing.ErrInsufficientQuota {
		t.Fatalf("expected ErrInsufficientQuota, got %v", err)
	}
}

// TestLedger_CleanerReleasesStale verifies the cleaner releases an old reserve.
func TestLedger_CleanerReleasesStale(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "ITest")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID) })

	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: 1000, days: 30}, nil)

	// token_ledger is append-only (UPDATE/DELETE blocked), but INSERT with an
	// explicit created_at is allowed. Insert a reserve dated well in the past so
	// it is unambiguously stale, then reserve a fresh one to prove the cleaner
	// only touches the old entry.
	period, err := repo.GetOrCreatePeriod(ctx, userID, time.Now().Add(-48*time.Hour).Unix(), time.Now().Add(30*24*time.Hour).Unix(), 1000)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}
	oldReserveID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO token_ledger (id, period_id, user_id, reservation_id, entry_type, amount, modality, idempotency_key, created_at)
		VALUES ($1,$2,$3,$1,'reserve',250,'chat',$4, now() - interval '2 hours')`,
		oldReserveID, period.ID, userID, "itest-clean-old-"+userID.String())
	if err != nil {
		t.Fatalf("insert old reserve: %v", err)
	}
	// Account for the reservation in the period so we can assert it is released.
	if err := repo.AdjustPeriod(ctx, period.ID, 0, 250); err != nil {
		t.Fatalf("adjust period: %v", err)
	}

	// A fresh reserve (younger than MaxAge) must survive the sweep.
	fresh, err := ledger.Reserve(ctx, userID, 100, billing.ModalityChat, "itest-clean-fresh-"+userID.String())
	if err != nil {
		t.Fatalf("fresh reserve: %v", err)
	}

	worker := billing.NewCleanerWorker(repo, ledger, billing.CleanerConfig{Interval: time.Hour, MaxAge: 30 * time.Minute})
	worker.SweepOnce(ctx)

	period, _ = repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	// Old reserve released (250), fresh reserve still held (100) → reserved = 100.
	if period.ReservedTokens != 100 {
		t.Fatalf("cleaner should have released only the stale reservation; reserved=%d (want 100; old=%s fresh=%s)",
			period.ReservedTokens, oldReserveID, fresh.ID)
	}
}
