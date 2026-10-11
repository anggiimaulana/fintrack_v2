package postgres

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// TestLedger_ConcurrentReserveNoOverspend is the core anti-double-spend proof:
// many goroutines reserve concurrently against a small quota; the number that
// succeed must never exceed quota/amount, and reserved_tokens must never exceed
// the quota.
func TestLedger_ConcurrentReserveNoOverspend(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "Concurrency"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	const quota = 1000
	const perReserve = 100 // => at most 10 successes
	const goroutines = 40

	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: quota, days: 30}, nil)

	var success, insufficient, other int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := range goroutines {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start // release all at once to maximise contention
			key := fmt.Sprintf("conc-%s-%d", userID.String(), n)
			_, err := ledger.Reserve(ctx, userID, perReserve, billing.ModalityChat, key)
			switch {
			case err == nil:
				atomic.AddInt32(&success, 1)
			case err == billing.ErrInsufficientQuota:
				atomic.AddInt32(&insufficient, 1)
			default:
				atomic.AddInt32(&other, 1)
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	maxSuccess := int32(quota / perReserve)
	if success > maxSuccess {
		t.Fatalf("over-spend! success=%d exceeds max %d", success, maxSuccess)
	}
	if other != 0 {
		t.Fatalf("expected no unexpected errors, got %d", other)
	}
	if success+insufficient != goroutines {
		t.Fatalf("accounting mismatch: success=%d insufficient=%d total=%d", success, insufficient, goroutines)
	}

	// The DB invariant must hold: reserved <= quota.
	period, err := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if err != nil || period == nil {
		t.Fatalf("period lookup: %v", err)
	}
	if period.ReservedTokens > quota {
		t.Fatalf("reserved %d exceeds quota %d", period.ReservedTokens, quota)
	}
	if period.ReservedTokens != int64(success)*perReserve {
		t.Fatalf("reserved=%d does not match success*amount=%d", period.ReservedTokens, int64(success)*perReserve)
	}
	t.Logf("success=%d insufficient=%d reserved=%d (quota=%d)", success, insufficient, period.ReservedTokens, quota)
}

// TestLedger_ConcurrentCommitNoNegative verifies that concurrent commits never
// drive used_tokens negative and each reservation resolves exactly once.
func TestLedger_ConcurrentCommitNoNegative(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "Concurrency"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	ledger := billing.NewLedgerService(repo, tx, fixedQuota{quota: 10000, days: 30}, nil)

	// Create 10 reservations sequentially (avoids quota contention here).
	const n = 10
	reservations := make([]uuid.UUID, 0, n)
	for i := range n {
		r, err := ledger.Reserve(ctx, userID, 100, billing.ModalityChat, fmt.Sprintf("cc-%s-%d", userID.String(), i))
		if err != nil {
			t.Fatalf("reserve %d: %v", i, err)
		}
		reservations = append(reservations, r.ID)
	}

	// Commit each reservation twice, concurrently: exactly one must take effect.
	var wg sync.WaitGroup
	for _, id := range reservations {
		for dup := range 2 {
			wg.Add(1)
			go func(rid uuid.UUID, d int) {
				defer wg.Done()
				// Second attempt uses a different idempotency key to try to
				// double-charge the same reservation.
				_ = ledger.Commit(ctx, userID, rid, 100, fmt.Sprintf("commit-%s-%d", rid.String(), d))
			}(id, dup)
		}
	}
	wg.Wait()

	period, err := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if err != nil || period == nil {
		t.Fatalf("period lookup: %v", err)
	}
	if period.UsedTokens < 0 || period.ReservedTokens < 0 {
		t.Fatalf("balances must never be negative: used=%d reserved=%d", period.UsedTokens, period.ReservedTokens)
	}
	// Exactly one commit per reservation: used should be n*100 = 1000, and
	// reserved back to 0.
	if period.UsedTokens != int64(n)*100 {
		t.Fatalf("expected used=%d (one commit each), got %d", n*100, period.UsedTokens)
	}
	if period.ReservedTokens != 0 {
		t.Fatalf("expected reserved=0 after all commits, got %d", period.ReservedTokens)
	}
}
