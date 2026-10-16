package billing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeTokens is an in-memory TokenRepository so the ledger's conversion logic
// can be tested without a database. It implements just enough to exercise
// reserve → commit/release.
type fakeTokens struct {
	period   *TokenPeriod
	ledger   []*TokenLedgerEntry
	byKey    map[string]*TokenLedgerEntry
	resolved map[uuid.UUID]bool
	// onLock runs inside GetPeriodByIDForUpdate, i.e. AFTER the caller's
	// pre-transaction resolved-check but BEFORE the in-transaction one. Tests
	// use it to simulate a concurrent commit winning the race in that window.
	onLock func()
}

func newFakeTokens(quota, ratio int64) *fakeTokens {
	return &fakeTokens{
		period: &TokenPeriod{
			ID: uuid.New(), PeriodStart: time.Now().Add(-time.Hour), PeriodEnd: time.Now().Add(24 * time.Hour),
			QuotaTokens: quota, QuotaRatio: ratio,
		},
		byKey:    map[string]*TokenLedgerEntry{},
		resolved: map[uuid.UUID]bool{},
	}
}

func (f *fakeTokens) GetOrCreatePeriod(_ context.Context, _ uuid.UUID, start, end int64, quota, ratio int64) (*TokenPeriod, error) {
	// Mirror the repository: insert-or-return the row for this exact window.
	// A row for a different window does not satisfy this one.
	ps := time.Unix(start, 0).UTC()
	pe := time.Unix(end, 0).UTC()
	if f.period != nil && f.period.PeriodStart.Equal(ps) {
		return f.period, nil
	}
	f.period = &TokenPeriod{ID: uuid.New(), PeriodStart: ps, PeriodEnd: pe, QuotaTokens: quota, QuotaRatio: ratio}
	return f.period, nil
}

// GetPeriodForUpdate mirrors the real query: it only returns a row whose
// [period_start, period_end] covers `nowUnix`, so a stale window yields nil.
func (f *fakeTokens) GetPeriodForUpdate(_ context.Context, _ uuid.UUID, nowUnix int64) (*TokenPeriod, error) {
	now := time.Unix(nowUnix, 0).UTC()
	if f.period == nil || f.period.PeriodStart.After(now) || f.period.PeriodEnd.Before(now) {
		return nil, nil
	}
	return f.period, nil
}
func (f *fakeTokens) GetPeriodByIDForUpdate(_ context.Context, id uuid.UUID) (*TokenPeriod, error) {
	if f.onLock != nil {
		f.onLock()
	}
	if f.period != nil && f.period.ID == id {
		return f.period, nil
	}
	return f.period, nil
}
func (f *fakeTokens) GetCurrentPeriod(context.Context, uuid.UUID, int64) (*TokenPeriod, error) {
	return f.period, nil
}
func (f *fakeTokens) AdjustPeriod(_ context.Context, _ uuid.UUID, usedDelta, reservedDelta int64) error {
	f.period.UsedTokens += usedDelta
	f.period.ReservedTokens += reservedDelta
	return nil
}
func (f *fakeTokens) UpdatePeriodQuota(_ context.Context, _ uuid.UUID, quota, ratio int64) error {
	f.period.QuotaTokens, f.period.QuotaRatio = quota, ratio
	return nil
}
func (f *fakeTokens) InsertLedger(_ context.Context, e *TokenLedgerEntry) error {
	if _, ok := f.byKey[e.IdempotencyKey]; ok {
		return ErrDuplicateIdempotencyKey
	}
	f.ledger = append(f.ledger, e)
	f.byKey[e.IdempotencyKey] = e
	// Mirror the DB trigger: a commit/release resolves its reservation, so a
	// later resolution attempt is detected.
	if e.ReservationID != nil && (e.EntryType == EntryCommit || e.EntryType == EntryRelease) {
		f.resolved[*e.ReservationID] = true
	}
	return nil
}
func (f *fakeTokens) GetLedgerByIdempotencyKey(_ context.Context, key string) (*TokenLedgerEntry, error) {
	return f.byKey[key], nil
}
func (f *fakeTokens) GetLedgerByID(_ context.Context, id uuid.UUID) (*TokenLedgerEntry, error) {
	for _, e := range f.ledger {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}
func (f *fakeTokens) IsReservationResolved(_ context.Context, id uuid.UUID) (bool, error) {
	return f.resolved[id], nil
}
func (f *fakeTokens) ListExpiredReservations(context.Context, time.Time) ([]*TokenLedgerEntry, error) {
	return nil, nil
}

// fakeTx runs the callback inline (no real transaction).
type fakeTx struct{}

func (fakeTx) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

// staticRounding returns a fixed rounding policy.
type staticRounding struct {
	mode RoundingMode
	min  int64
}

func (s staticRounding) Mode(context.Context) RoundingMode { return s.mode }
func (s staticRounding) MinCharge(context.Context) int64   { return s.min }

func newTestLedger(quota, ratio int64) (*LedgerService, *fakeTokens) {
	tokens := newFakeTokens(quota, ratio)
	quotaFn := QuotaResolverFunc(func(_ context.Context, _ uuid.UUID, at time.Time) (time.Time, time.Time, int64, int64, error) {
		start := at.Truncate(24 * time.Hour)
		return start, start.AddDate(0, 0, 30), quota, ratio, nil
	})
	svc := NewLedgerService(tokens, fakeTx{}, quotaFn, nil).
		WithRounding(staticRounding{mode: RoundCeil, min: 1})
	return svc, tokens
}

// I-6 + core conversion: raw tokens are divided by the ratio before charging,
// while the ledger records the raw count separately for cost.
func TestLedger_ReserveCommit_ConvertsRawToUnits(t *testing.T) {
	svc, tokens := newTestLedger(5000, 5)
	ctx := context.Background()
	user := uuid.New()

	res, err := svc.Reserve(ctx, user, 3809, ModalityChat, "r1")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	// ceil(3809/5) = 762 units
	if res.Amount != 762 {
		t.Fatalf("reserve units = %d, want 762", res.Amount)
	}
	if tokens.period.ReservedTokens != 762 {
		t.Fatalf("reserved = %d, want 762", tokens.period.ReservedTokens)
	}

	if err := svc.Commit(ctx, user, res.ID, 9945, "c1"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	// ceil(9945/5) = 1989 units
	if tokens.period.UsedTokens != 1989 {
		t.Fatalf("used units = %d, want 1989", tokens.period.UsedTokens)
	}
	if tokens.period.ReservedTokens != 0 {
		t.Fatalf("reserved after commit = %d, want 0", tokens.period.ReservedTokens)
	}
	if tokens.period.Available() != 3011 {
		t.Fatalf("available = %d, want 3011", tokens.period.Available())
	}

	commit := tokens.byKey["c1"]
	if commit.Amount != 1989 {
		t.Fatalf("ledger units = %d, want 1989", commit.Amount)
	}
	if commit.RawTokens != 9945 {
		t.Fatalf("ledger raw_tokens = %d, want 9945", commit.RawTokens)
	}
}

// I-8 / backward compatibility: ratio 1 is the identity.
func TestLedger_RatioOneIsIdentity(t *testing.T) {
	svc, tokens := newTestLedger(100000, 1)
	ctx := context.Background()
	user := uuid.New()

	res, err := svc.Reserve(ctx, user, 500, ModalityChat, "r1")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if res.Amount != 500 {
		t.Fatalf("ratio 1 identity: got %d want 500", res.Amount)
	}
	if err := svc.Commit(ctx, user, res.ID, 500, "c1"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if tokens.period.UsedTokens != 500 {
		t.Fatalf("used = %d, want 500", tokens.period.UsedTokens)
	}
}

// I-4: Release returns the ENTIRE held unit amount.
func TestLedger_ReleaseReturnsFullHold(t *testing.T) {
	svc, tokens := newTestLedger(5000, 5)
	ctx := context.Background()
	user := uuid.New()

	res, err := svc.Reserve(ctx, user, 9945, ModalityChat, "r1")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if res.Amount != 1989 {
		t.Fatalf("held = %d, want 1989", res.Amount)
	}
	if err := svc.Release(ctx, user, res.ID, "rel1"); err != nil {
		t.Fatalf("release: %v", err)
	}
	if tokens.period.ReservedTokens != 0 {
		t.Fatalf("reserved after release = %d, want 0", tokens.period.ReservedTokens)
	}
	if tokens.period.UsedTokens != 0 {
		t.Fatalf("release must not charge: used = %d", tokens.period.UsedTokens)
	}
}

// I-1: the charge is clamped so used never exceeds quota.
func TestLedger_CommitClampsToQuota(t *testing.T) {
	svc, tokens := newTestLedger(1000, 5)
	ctx := context.Background()
	user := uuid.New()

	res, err := svc.Reserve(ctx, user, 100, ModalityChat, "r1")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	// 100000 raw / 5 = 20000 units, far above the 1000-unit quota.
	if err := svc.Commit(ctx, user, res.ID, 100000, "c1"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if tokens.period.UsedTokens != 1000 {
		t.Fatalf("used must clamp to quota: got %d want 1000", tokens.period.UsedTokens)
	}
	if tokens.period.UsedTokens > tokens.period.QuotaTokens {
		t.Fatalf("used (%d) exceeds quota (%d)", tokens.period.UsedTokens, tokens.period.QuotaTokens)
	}
}

// Insufficient quota is rejected before any mutation.
func TestLedger_ReserveInsufficientQuota(t *testing.T) {
	svc, tokens := newTestLedger(100, 1)
	ctx := context.Background()
	user := uuid.New()

	_, err := svc.Reserve(ctx, user, 500, ModalityChat, "r1")
	if err != ErrInsufficientQuota {
		t.Fatalf("want ErrInsufficientQuota, got %v", err)
	}
	if tokens.period.ReservedTokens != 0 {
		t.Fatalf("no hold should be taken on rejection: reserved = %d", tokens.period.ReservedTokens)
	}
}

// Refund is symmetric with Commit: raw → units, clamped to what was charged.
func TestLedger_RefundConvertsAndClamps(t *testing.T) {
	svc, tokens := newTestLedger(5000, 5)
	ctx := context.Background()
	user := uuid.New()

	res, _ := svc.Reserve(ctx, user, 9945, ModalityChat, "r1")
	_ = svc.Commit(ctx, user, res.ID, 9945, "c1") // used = 1989
	if err := svc.Refund(ctx, user, 9945, ModalityChat, "ref1"); err != nil {
		t.Fatalf("refund: %v", err)
	}
	if tokens.period.UsedTokens != 0 {
		t.Fatalf("after full refund used = %d, want 0", tokens.period.UsedTokens)
	}
}

// floor mode can round to 0; the minimum charge must still bill at least 1 unit.
func TestLedger_MinChargeCoversFloorRounding(t *testing.T) {
	tokens := newFakeTokens(5000, 5)
	quotaFn := QuotaResolverFunc(func(_ context.Context, _ uuid.UUID, at time.Time) (time.Time, time.Time, int64, int64, error) {
		start := at.Truncate(24 * time.Hour)
		return start, start.AddDate(0, 0, 30), 5000, 5, nil
	})
	svc := NewLedgerService(tokens, fakeTx{}, quotaFn, nil).
		WithRounding(staticRounding{mode: RoundFloor, min: 1})
	ctx := context.Background()
	user := uuid.New()

	res, err := svc.Reserve(ctx, user, 4, ModalityChat, "r1") // floor(4/5) = 0 -> min 1
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if res.Amount != 1 {
		t.Fatalf("min charge must bill 1 unit: got %d", res.Amount)
	}
}

// A reservation must be committed at most once, even when two commits race
// between the caller's pre-transaction check and the period lock. Regression
// guard for the double-charge bug: the old code checked `resolved` only BEFORE
// the transaction, so both commits charged and the ledger ended up with two
// commit rows for one reservation.
func TestLedger_CommitResolvesReservationOnce(t *testing.T) {
	svc, tokens := newTestLedger(10000, 1)
	ctx := context.Background()
	user := uuid.New()

	res, err := svc.Reserve(ctx, user, 100, ModalityChat, "r1")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	// Simulate the loser of the race: while this commit is inside the
	// transaction (holding the period lock), another commit for the SAME
	// reservation lands and resolves it first.
	var onLockOnce bool
	tokens.onLock = func() {
		if onLockOnce {
			return
		}
		onLockOnce = true
		tokens.resolved[res.ID] = true
		tokens.period.UsedTokens += 100
		tokens.period.ReservedTokens -= 100
	}

	err = svc.Commit(ctx, user, res.ID, 100, "c-loser")
	if err != ErrReservationAlreadyResolved {
		t.Fatalf("want ErrReservationAlreadyResolved, got %v", err)
	}
	// The losing commit must not have charged a second time.
	if tokens.period.UsedTokens != 100 {
		t.Fatalf("double charge! used = %d, want 100", tokens.period.UsedTokens)
	}
}

// Same guard for Release: it must not return the hold twice.
func TestLedger_ReleaseResolvesReservationOnce(t *testing.T) {
	svc, tokens := newTestLedger(10000, 1)
	ctx := context.Background()
	user := uuid.New()

	res, err := svc.Reserve(ctx, user, 100, ModalityChat, "r1")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	var onLockOnce bool
	tokens.onLock = func() {
		if onLockOnce {
			return
		}
		onLockOnce = true
		tokens.resolved[res.ID] = true
		tokens.period.ReservedTokens -= 100
	}

	err = svc.Release(ctx, user, res.ID, "rel-loser")
	if err != ErrReservationAlreadyResolved {
		t.Fatalf("want ErrReservationAlreadyResolved, got %v", err)
	}
	if tokens.period.ReservedTokens != 0 {
		t.Fatalf("reserved = %d, want 0 (no double release)", tokens.period.ReservedTokens)
	}
}

// A lapsed cycle window (subscription still "active" but current_period_end in
// the past, no renewal yet) must not panic. Regression guard: the resolver
// returned a PAST window, GetOrCreatePeriod inserted it verbatim, the
// current-time lookup then found no covering row, and Reserve dereferenced the
// nil period (nil-pointer panic → HTTP 500).
func TestLedger_LapsedWindowDoesNotPanic(t *testing.T) {
	tokens := newFakeTokens(5000, 1)
	// Start with NO period row: force the resolve+create path.
	tokens.period = nil
	now := time.Now().UTC()
	// Resolver returns a window entirely in the past.
	quotaFn := QuotaResolverFunc(func(_ context.Context, _ uuid.UUID, _ time.Time) (time.Time, time.Time, int64, int64, error) {
		return now.Add(-60 * 24 * time.Hour), now.Add(-30 * 24 * time.Hour), 5000, 1, nil
	})
	svc := NewLedgerService(tokens, fakeTx{}, quotaFn, nil).
		WithRounding(staticRounding{mode: RoundCeil, min: 1})

	res, err := svc.Reserve(context.Background(), uuid.New(), 100, ModalityChat, "lapsed-r1")
	if err != nil {
		t.Fatalf("reserve on lapsed window must succeed, got %v", err)
	}
	if res == nil {
		t.Fatal("reserve returned nil reservation")
	}
	if tokens.period == nil {
		t.Fatal("period must not be nil after reserve")
	}
	// The rolled-forward window must cover now.
	if tokens.period.PeriodStart.After(now) || tokens.period.PeriodEnd.Before(now) {
		t.Fatalf("period window must cover now: [%s, %s]",
			tokens.period.PeriodStart, tokens.period.PeriodEnd)
	}
}

// A running period keeps its quota snapshot: the ledger does NOT reopen the
// cycle on a quota change. Syncing the running period is the activator's job
// (see QuotaSyncer), so this asserts the ledger leaves it alone.
func TestLedger_RunningPeriodKeepsSnapshot(t *testing.T) {
	tokens := newFakeTokens(5000, 1)
	windowStart := time.Now().UTC().Truncate(time.Hour)
	quota := int64(5000)
	quotaFn := QuotaResolverFunc(func(_ context.Context, _ uuid.UUID, _ time.Time) (time.Time, time.Time, int64, int64, error) {
		return windowStart, windowStart.AddDate(0, 0, 30), quota, 1, nil
	})
	svc := NewLedgerService(tokens, fakeTx{}, quotaFn, nil).
		WithRounding(staticRounding{mode: RoundCeil, min: 1})
	ctx := context.Background()
	user := uuid.New()

	if _, err := svc.Reserve(ctx, user, 100, ModalityChat, "s1"); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	firstPeriodID := tokens.period.ID
	reservedBefore := tokens.period.ReservedTokens

	// The plan quota changes, window unchanged: the ledger must not reopen the
	// cycle (the activator syncs the running period instead).
	quota = 150000
	if _, err := svc.Reserve(ctx, user, 100, ModalityChat, "s2"); err != nil {
		t.Fatalf("reserve after quota change: %v", err)
	}
	if tokens.period.ID != firstPeriodID {
		t.Fatal("the ledger must not open a new cycle on a quota change")
	}
	if tokens.period.QuotaTokens != 5000 {
		t.Fatalf("cycle quota must stay snapshotted: got %d want 5000", tokens.period.QuotaTokens)
	}
	if tokens.period.ReservedTokens <= reservedBefore {
		t.Fatalf("usage must accumulate on the same period: reserved=%d", tokens.period.ReservedTokens)
	}
}

// A new window (renewal) DOES open a new cycle with the new quota.
func TestLedger_NewWindowOpensNewCycle(t *testing.T) {
	tokens := newFakeTokens(5000, 1)
	// A movable clock: the first cycle covers [T0, T0+30d]; after "time passes"
	// beyond it, the resolver returns the next cycle and the ledger must open it.
	t0 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	now := t0
	quotaFn := QuotaResolverFunc(func(_ context.Context, _ uuid.UUID, at time.Time) (time.Time, time.Time, int64, int64, error) {
		// The window containing `at`: either the first or the second cycle.
		if at.Before(t0.AddDate(0, 0, 30)) {
			return t0, t0.AddDate(0, 0, 30), 5000, 1, nil
		}
		return t0.AddDate(0, 0, 30), t0.AddDate(0, 0, 60), 150000, 1, nil
	})
	svc := NewLedgerService(tokens, fakeTx{}, quotaFn, nil).
		WithRounding(staticRounding{mode: RoundCeil, min: 1}).
		WithClock(func() time.Time { return now })
	ctx := context.Background()
	user := uuid.New()

	if _, err := svc.Reserve(ctx, user, 100, ModalityChat, "n1"); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	firstPeriodID := tokens.period.ID

	// Time passes beyond the first cycle: a renewal opens the second.
	now = t0.AddDate(0, 0, 31)
	if _, err := svc.Reserve(ctx, user, 100, ModalityChat, "n2"); err != nil {
		t.Fatalf("reserve after rollover: %v", err)
	}
	if tokens.period.ID == firstPeriodID {
		t.Fatal("a new window must open a new period")
	}
	if tokens.period.QuotaTokens != 150000 {
		t.Fatalf("new cycle quota = %d, want 150000", tokens.period.QuotaTokens)
	}
}
