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

func (f *fakeTokens) GetOrCreatePeriod(_ context.Context, _ uuid.UUID, _, _ int64, quota, ratio int64) (*TokenPeriod, error) {
	f.period.QuotaTokens, f.period.QuotaRatio = quota, ratio
	return f.period, nil
}
func (f *fakeTokens) GetPeriodForUpdate(context.Context, uuid.UUID, int64) (*TokenPeriod, error) {
	return f.period, nil
}
func (f *fakeTokens) GetPeriodByIDForUpdate(_ context.Context, id uuid.UUID) (*TokenPeriod, error) {
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
func (f *fakeTokens) InsertLedger(_ context.Context, e *TokenLedgerEntry) error {
	if _, ok := f.byKey[e.IdempotencyKey]; ok {
		return ErrDuplicateIdempotencyKey
	}
	f.ledger = append(f.ledger, e)
	f.byKey[e.IdempotencyKey] = e
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
