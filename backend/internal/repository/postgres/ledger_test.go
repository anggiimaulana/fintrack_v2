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
	// ratio is the token→unit ratio; 0 means 1 (identity), so pre-existing
	// tests keep working unchanged.
	ratio int64
}

func (f fixedQuota) ResolveQuota(_ context.Context, _ uuid.UUID, at time.Time) (time.Time, time.Time, int64, int64, error) {
	start := at.Truncate(24 * time.Hour)
	r := f.ratio
	if r < 1 {
		r = 1
	}
	return start, start.AddDate(0, 0, f.days), f.quota, r, nil
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
	period, err := repo.GetOrCreatePeriod(ctx, userID, time.Now().Add(-48*time.Hour).Unix(), time.Now().Add(30*24*time.Hour).Unix(), 1000, 1)
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

// TestLedger_DoubleResolutionRejectedAtDB proves the database is the final
// authority on "one resolution per reservation". The application fix checks
// inside the transaction, but this test bypasses the service and inserts two
// commit rows directly: the token_ledger_single_resolution trigger must reject
// the second, and the repository must surface it as
// ErrReservationAlreadyResolved (not a raw P0001).
func TestLedger_DoubleResolutionRejectedAtDB(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "DoubleRes"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	period, err := repo.GetOrCreatePeriod(ctx, userID, time.Now().Add(-time.Hour).Unix(), time.Now().Add(24*time.Hour).Unix(), 10000, 1)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	reservationID := uuid.New()
	base := billing.TokenLedgerEntry{
		PeriodID: period.ID, UserID: userID, ReservationID: &reservationID,
		Amount: 100, Modality: "chat",
	}
	first := base
	first.ID = uuid.New()
	first.EntryType = billing.EntryCommit
	first.IdempotencyKey = "dup-first-" + userID.String()
	if err := repo.InsertLedger(ctx, &first); err != nil {
		t.Fatalf("first commit: %v", err)
	}

	second := base
	second.ID = uuid.New()
	second.EntryType = billing.EntryCommit
	second.IdempotencyKey = "dup-second-" + userID.String() // different key, same reservation
	err = repo.InsertLedger(ctx, &second)
	if err != billing.ErrReservationAlreadyResolved {
		t.Fatalf("want ErrReservationAlreadyResolved, got %v", err)
	}

	// A release for an already-committed reservation is likewise rejected.
	rel := base
	rel.ID = uuid.New()
	rel.EntryType = billing.EntryRelease
	rel.Amount = -100
	rel.IdempotencyKey = "dup-release-" + userID.String()
	if err := repo.InsertLedger(ctx, &rel); err != billing.ErrReservationAlreadyResolved {
		t.Fatalf("release after commit: want ErrReservationAlreadyResolved, got %v", err)
	}
}

// TestLedger_LapsedSubscriptionNoPanic is the regression guard for the nil
// period panic: an `active` subscription whose current_period_end has lapsed
// (the cycle only advances on payment, and nothing renews it automatically)
// used to resolve to a past window, insert a row that never covers `now`, then
// hand a nil period to Reserve — a nil-pointer panic (HTTP 500). The ledger must
// roll the window forward so the cycle always covers now.
func TestLedger_LapsedSubscriptionNoPanic(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "Lapsed"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	pastStart := time.Now().UTC().Add(-60 * 24 * time.Hour)
	pastEnd := time.Now().UTC().Add(-30 * 24 * time.Hour)
	quota := billing.QuotaResolverFunc(func(_ context.Context, _ uuid.UUID, _ time.Time) (time.Time, time.Time, int64, int64, error) {
		return pastStart, pastEnd, 1000, 1, nil
	})
	ledger := billing.NewLedgerService(repo, tx, quota, nil)

	res, err := ledger.Reserve(ctx, userID, 100, billing.ModalityChat, "lapsed-res-"+userID.String())
	if err != nil {
		t.Fatalf("reserve must not fail on a lapsed window, got %v", err)
	}
	if res == nil {
		t.Fatal("nil reservation")
	}
	cur, err := repo.GetCurrentPeriod(ctx, userID, time.Now().UTC().Unix())
	if err != nil {
		t.Fatalf("current period: %v", err)
	}
	if cur == nil {
		t.Fatal("no period covers now after reserve (the nil -> panic bug)")
	}
	if cur.PeriodEnd.Before(time.Now().UTC()) {
		t.Fatalf("period must cover now: end=%s", cur.PeriodEnd)
	}
}

// TestQuotaSyncer_AppliesUpgradeToRunningPeriodDB is the regression guard for
// the "upgraded but still capped at the old plan" bug, at the DB level. A plan
// change must reach the RUNNING token period: QuotaSyncer updates its quota in
// place, preserving recorded usage and NOT opening a second (overlapping) period.
func TestQuotaSyncer_AppliesUpgradeToRunningPeriodDB(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "MidUpgrade"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	// Running Free cycle with usage.
	freeStart := time.Now().UTC().Add(-10 * 24 * time.Hour)
	free, err := repo.GetOrCreatePeriod(ctx, userID, freeStart.Unix(), freeStart.Add(30*24*time.Hour).Unix(), 5000, 1)
	if err != nil {
		t.Fatalf("free period: %v", err)
	}
	if err := repo.AdjustPeriod(ctx, free.ID, 4000, 0); err != nil {
		t.Fatalf("usage: %v", err)
	}

	// Upgrade: sync the running period to the Pro quota.
	proPlan, err := repo.GetPlanByCode(ctx, "pro")
	if err != nil {
		t.Fatalf("pro plan: %v", err)
	}
	limits, err := repo.GetLimits(ctx, proPlan.ID)
	if err != nil {
		t.Fatalf("pro limits: %v", err)
	}
	wantQuota := limits["ai_tokens_per_cycle"]
	if err := repo.UpdatePeriodQuota(ctx, free.ID, wantQuota, proPlan.TokenQuotaRatio); err != nil {
		t.Fatalf("sync quota: %v", err)
	}

	p, err := repo.GetPeriodByIDForUpdate(ctx, free.ID)
	if err != nil || p == nil {
		t.Fatalf("period lookup: %v", err)
	}
	if p.QuotaTokens != wantQuota {
		t.Fatalf("upgrade must apply to the running period: quota=%d want %d", p.QuotaTokens, wantQuota)
	}
	if p.UsedTokens != 4000 {
		t.Fatalf("usage must be preserved across a plan change: used=%d want 4000", p.UsedTokens)
	}

	// And no second period was created (the exclusion constraint would reject an
	// overlapping one anyway).
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM token_periods WHERE user_id=$1`, userID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("plan change must not open a second period: got %d", n)
	}
}

// TestTokenPeriod_OverlapRejectedAtDB proves the exclusion constraint is the
// final authority: two periods covering the same instant cannot both exist.
func TestTokenPeriod_OverlapRejectedAtDB(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "Overlap"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	base := time.Now().UTC().Add(-10 * 24 * time.Hour)
	if _, err := repo.GetOrCreatePeriod(ctx, userID, base.Unix(), base.Add(30*24*time.Hour).Unix(), 1000, 1); err != nil {
		t.Fatalf("first period: %v", err)
	}
	// Overlapping window: must be rejected by token_periods_no_overlap.
	overlapStart := base.Add(5 * 24 * time.Hour)
	_, err := repo.GetOrCreatePeriod(ctx, userID, overlapStart.Unix(), overlapStart.Add(30*24*time.Hour).Unix(), 2000, 1)
	if err == nil {
		t.Fatal("overlapping period must be rejected by the exclusion constraint")
	}
}

// TestLifecycle_LapsedSubscriptionExpires proves the end-of-cycle worker's
// query + transition work against the real schema: an `active` subscription
// whose cycle ended (and grace lapsed) is listed as due and can be expired.
func TestLifecycle_LapsedSubscriptionExpires(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "Lifecycle"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	freePlan, err := repo.GetPlanByCode(ctx, "free")
	if err != nil {
		t.Fatalf("free plan: %v", err)
	}
	// Active subscription whose cycle ended 100h ago (beyond the 72h grace).
	pastEnd := time.Now().UTC().Add(-100 * time.Hour)
	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (user_id, plan_id, status, current_period_start, current_period_end)
		 VALUES ($1,$2,'active',$3,$4)`,
		userID, freePlan.ID, pastEnd.Add(-30*24*time.Hour), pastEnd); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	due, err := repo.ListSubscriptionsDueForLifecycle(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("list due: %v", err)
	}
	found := false
	for _, d := range due {
		if d.UserID == userID {
			found = true
		}
	}
	if !found {
		t.Fatal("lapsed active subscription must be listed as due")
	}

	// A cancelled subscription due at its boundary is also listed.
	if _, err := pool.Exec(ctx, `UPDATE subscriptions SET cancel_at_period_end=true WHERE user_id=$1`, userID); err != nil {
		t.Fatalf("set cancel: %v", err)
	}
	due2, _ := repo.ListSubscriptionsDueForLifecycle(ctx, time.Now().UTC())
	var cancelFlag bool
	for _, d := range due2 {
		if d.UserID == userID {
			cancelFlag = d.CancelAtPeriodEnd
		}
	}
	if !cancelFlag {
		t.Fatal("cancel_at_period_end must be surfaced to the worker")
	}
}

// TestSetSubscriptionStatus_ProtectsTerminalStates proves the grace worker
// cannot resurrect an expired/canceled subscription: a status setter must not
// move a terminal state back to a non-terminal one, or paid access would
// silently return after the lifecycle worker expired it.
func TestSetSubscriptionStatus_ProtectsTerminalStates(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "Terminal"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	proPlan, _ := repo.GetPlanByCode(ctx, "pro")
	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (user_id, plan_id, status, current_period_start, current_period_end)
		 VALUES ($1,$2,'expired', now() - interval '60 days', now() - interval '30 days')`,
		userID, proPlan.ID); err != nil {
		t.Fatalf("insert sub: %v", err)
	}

	// The grace worker tries to set past_due — must be ignored.
	if err := repo.SetSubscriptionStatus(ctx, userID, billing.StatusPastDue); err != nil {
		t.Fatalf("set status: %v", err)
	}
	sub, _ := repo.GetByUserID(ctx, userID)
	if sub.Status != billing.StatusExpired {
		t.Fatalf("expired must be protected from regression: got %s", sub.Status)
	}

	// A non-terminal status can still be updated normally.
	if _, err := pool.Exec(ctx, `UPDATE subscriptions SET status='active' WHERE user_id=$1`, userID); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := repo.SetSubscriptionStatus(ctx, userID, billing.StatusPastDue); err != nil {
		t.Fatalf("set status: %v", err)
	}
	sub, _ = repo.GetByUserID(ctx, userID)
	if sub.Status != billing.StatusPastDue {
		t.Fatalf("active -> past_due must still work: got %s", sub.Status)
	}
}
