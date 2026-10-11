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

	// Reserve and backdate the ledger entry so it looks stale.
	res, err := ledger.Reserve(ctx, userID, 250, billing.ModalityChat, "itest-clean-"+userID.String())
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	// token_ledger is append-only; we cannot UPDATE created_at. Instead, use a
	// zero maxAge so everything is immediately "expired".
	worker := billing.NewCleanerWorker(repo, ledger, billing.CleanerConfig{Interval: time.Hour, MaxAge: time.Nanosecond})
	worker.SweepOnce(ctx)

	period, _ := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if period.ReservedTokens != 0 {
		t.Fatalf("cleaner should have released the reservation; reserved=%d (res=%s)", period.ReservedTokens, res.ID)
	}
}
