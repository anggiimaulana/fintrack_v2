package billing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakePlans is a minimal PlanRepository for syncer tests.
type fakePlans struct {
	byCode map[string]*Plan
	limits map[uuid.UUID]map[string]int64
}

func (f *fakePlans) GetPlanByCode(_ context.Context, code string) (*Plan, error) {
	if p, ok := f.byCode[code]; ok {
		return p, nil
	}
	return nil, ErrPlanNotFound
}
func (f *fakePlans) GetPlanByID(context.Context, uuid.UUID) (*Plan, error) {
	return nil, ErrPlanNotFound
}
func (f *fakePlans) ListActivePlans(context.Context) ([]*Plan, error) { return nil, nil }
func (f *fakePlans) GetEntitlements(context.Context, uuid.UUID) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (f *fakePlans) GetLimits(_ context.Context, planID uuid.UUID) (map[string]int64, error) {
	return f.limits[planID], nil
}

// TestQuotaSyncer_UpdatesRunningPeriod is the regression guard for "upgraded but
// still capped at the old plan": after a mid-cycle plan change the RUNNING
// period's quota must be updated in place (usage preserved, no new period).
func TestQuotaSyncer_UpdatesRunningPeriod(t *testing.T) {
	tokens := newFakeTokens(5000, 1) // running Free period, still covering now
	user := uuid.New()
	if _, err := tokens.GetOrCreatePeriod(context.Background(), user, tokens.period.PeriodStart.Unix(), tokens.period.PeriodEnd.Unix(), 5000, 1); err != nil {
		t.Fatalf("seed period: %v", err)
	}
	tokens.period.UsedTokens = 4000
	periodID := tokens.period.ID

	proID := uuid.New()
	plans := &fakePlans{
		byCode: map[string]*Plan{"pro": {ID: proID, Code: "pro", TokenQuotaRatio: 1}},
		limits: map[uuid.UUID]map[string]int64{proID: {"ai_tokens_per_cycle": 150000}},
	}

	syncer := NewQuotaSyncer(tokens, plans)
	if err := syncer.SyncRunningPeriod(context.Background(), user, "pro"); err != nil {
		t.Fatalf("sync: %v", err)
	}

	if tokens.period.ID != periodID {
		t.Fatal("sync must update the running period in place, not create a new one")
	}
	if tokens.period.QuotaTokens != 150000 {
		t.Fatalf("quota = %d, want 150000", tokens.period.QuotaTokens)
	}
	if tokens.period.UsedTokens != 4000 {
		t.Fatalf("usage must be preserved: used = %d, want 4000", tokens.period.UsedTokens)
	}
}

// No running period => nothing to sync (the next AI call creates it).
func TestQuotaSyncer_NoPeriodIsNoop(t *testing.T) {
	tokens := newFakeTokens(5000, 1)
	tokens.period = nil
	proID := uuid.New()
	plans := &fakePlans{
		byCode: map[string]*Plan{"pro": {ID: proID, Code: "pro", TokenQuotaRatio: 1}},
		limits: map[uuid.UUID]map[string]int64{proID: {"ai_tokens_per_cycle": 150000}},
	}
	if err := NewQuotaSyncer(tokens, plans).SyncRunningPeriod(context.Background(), uuid.New(), "pro"); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if tokens.period != nil {
		t.Fatal("sync must not create a period when none exists")
	}
}

// An unlimited plan (-1) syncs to the ledger's sentinel, not a negative quota.
func TestQuotaSyncer_UnlimitedUsesSentinel(t *testing.T) {
	tokens := newFakeTokens(5000, 1)
	user := uuid.New()
	bizID := uuid.New()
	plans := &fakePlans{
		byCode: map[string]*Plan{"business": {ID: bizID, Code: "business", TokenQuotaRatio: 1}},
		limits: map[uuid.UUID]map[string]int64{bizID: {"ai_tokens_per_cycle": -1}},
	}
	if err := NewQuotaSyncer(tokens, plans).SyncRunningPeriod(context.Background(), user, "business"); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if tokens.period.QuotaTokens != defaultUnlimitedSentinel {
		t.Fatalf("unlimited quota = %d, want sentinel %d", tokens.period.QuotaTokens, defaultUnlimitedSentinel)
	}
}

// The syncer is a no-op when the running period already matches the plan.
func TestQuotaSyncer_AlreadyMatchingIsNoop(t *testing.T) {
	tokens := newFakeTokens(150000, 1)
	user := uuid.New()
	proID := uuid.New()
	plans := &fakePlans{
		byCode: map[string]*Plan{"pro": {ID: proID, Code: "pro", TokenQuotaRatio: 1}},
		limits: map[uuid.UUID]map[string]int64{proID: {"ai_tokens_per_cycle": 150000}},
	}
	periodID := tokens.period.ID
	if err := NewQuotaSyncer(tokens, plans).SyncRunningPeriod(context.Background(), user, "pro"); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if tokens.period.ID != periodID || tokens.period.QuotaTokens != 150000 {
		t.Fatal("no-op expected when the period already matches the plan")
	}
}

// Sanity: the movable-clock window helper used elsewhere stays consistent.
func TestQuotaSyncer_RatioSynced(t *testing.T) {
	tokens := newFakeTokens(5000, 1)
	user := uuid.New()
	proID := uuid.New()
	plans := &fakePlans{
		byCode: map[string]*Plan{"pro": {ID: proID, Code: "pro", TokenQuotaRatio: 5}},
		limits: map[uuid.UUID]map[string]int64{proID: {"ai_tokens_per_cycle": 150000}},
	}
	_ = time.Now()
	if err := NewQuotaSyncer(tokens, plans).SyncRunningPeriod(context.Background(), user, "pro"); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if tokens.period.QuotaRatio != 5 {
		t.Fatalf("ratio = %d, want 5", tokens.period.QuotaRatio)
	}
}

// TestQuotaSyncer_DowngradeToFreeReclaimsPaidQuota is the regression guard for
// the revenue leak: when a subscription lapses, the RUNNING period still held
// the paid quota, so the expired user kept Pro-level usage until the period
// ended. Syncing to the fallback plan must drop the ceiling immediately.
func TestQuotaSyncer_DowngradeToFreeReclaimsPaidQuota(t *testing.T) {
	tokens := newFakeTokens(150000, 1) // running Pro period, covering now
	user := uuid.New()
	tokens.period.UsedTokens = 10000

	freeID := uuid.New()
	plans := &fakePlans{
		byCode: map[string]*Plan{"free": {ID: freeID, Code: "free", TokenQuotaRatio: 1}},
		limits: map[uuid.UUID]map[string]int64{freeID: {"ai_tokens_per_cycle": 5000}},
	}

	if err := NewQuotaSyncer(tokens, plans).SyncRunningPeriod(context.Background(), user, "free"); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if tokens.period.QuotaTokens != 5000 {
		t.Fatalf("lapsed user must drop to the Free quota: got %d want 5000", tokens.period.QuotaTokens)
	}
	if tokens.period.UsedTokens != 10000 {
		t.Fatalf("usage must be preserved: got %d want 10000", tokens.period.UsedTokens)
	}
	// Usage above the new ceiling: the ledger clamps on commit, but available
	// must never read positive by accident.
	if tokens.period.Available() > 0 {
		t.Fatalf("available must not be positive when usage exceeds the new quota: %d", tokens.period.Available())
	}
}
