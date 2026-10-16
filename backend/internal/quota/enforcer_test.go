package quota_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/quota"
)

type mockCycleStore struct {
	cycles     map[string]*quota.PlanLimitCycle
	thresholds map[uuid.UUID]int
	globalThresh int
}

func (m *mockCycleStore) GetPlanCycle(_ context.Context, planID uuid.UUID, limitKey string) (*quota.PlanLimitCycle, error) {
	key := planID.String() + ":" + limitKey
	return m.cycles[key], nil
}

func (m *mockCycleStore) GetWarningThreshold(_ context.Context, planID uuid.UUID) (int, error) {
	if t, ok := m.thresholds[planID]; ok && t > 0 {
		return t, nil
	}
	if m.globalThresh > 0 {
		return m.globalThresh, nil
	}
	return 80, nil
}

type mockUsageCounterStore struct {
	counters map[string]*quota.UsageCounter
}

func newMockUsageCounterStore() *mockUsageCounterStore {
	return &mockUsageCounterStore{counters: make(map[string]*quota.UsageCounter)}
}

func (m *mockUsageCounterStore) GetOrCreateCounter(
	_ context.Context,
	userID uuid.UUID,
	limitKey, cycleType string,
	start, end time.Time,
) (*quota.UsageCounter, error) {
	key := userID.String() + ":" + limitKey + ":" + cycleType + ":" + start.Format(time.RFC3339)
	if c, ok := m.counters[key]; ok {
		return c, nil
	}
	c := &quota.UsageCounter{
		ID:         uuid.New(),
		UserID:     userID,
		LimitKey:   limitKey,
		CycleType:  cycleType,
		CycleStart: start,
		CycleEnd:   end,
		Used:       0,
	}
	m.counters[key] = c
	return c, nil
}

func (m *mockUsageCounterStore) IncrementUsage(_ context.Context, counterID uuid.UUID, delta int64, hardLimit int64) error {
	for _, c := range m.counters {
		if c.ID == counterID {
			c.Used += delta
			return nil
		}
	}
	return nil
}

func (m *mockUsageCounterStore) IncrementDegradedUsage(_ context.Context, counterID uuid.UUID, delta int64, maxDegraded int64) error {
	for _, c := range m.counters {
		if c.ID == counterID {
			c.DegradedUsed += delta
			return nil
		}
	}
	return nil
}

func (m *mockUsageCounterStore) GetCurrentCounters(_ context.Context, userID uuid.UUID, now time.Time) ([]quota.UsageCounter, error) {
	var list []quota.UsageCounter
	for _, c := range m.counters {
		if c.UserID == userID && !now.Before(c.CycleStart) && now.Before(c.CycleEnd) {
			list = append(list, *c)
		}
	}
	return list, nil
}

func TestEnforcer_MonthlyOnly_WithinLimit(t *testing.T) {
	store := &mockCycleStore{cycles: make(map[string]*quota.PlanLimitCycle)}
	usageStore := newMockUsageCounterStore()
	enforcer := quota.NewDualCycleEnforcer(store, usageStore)

	userID := uuid.New()
	planID := uuid.New()
	now := time.Date(2026, 10, 16, 14, 30, 0, 0, time.UTC)
	periodStart := now.Add(-5 * 24 * time.Hour)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	// User has used 20 out of 100
	counter, _ := usageStore.GetOrCreateCounter(context.Background(), userID, "max_transactions", quota.CycleTypeMonthly, periodStart, periodEnd)
	counter.Used = 20

	res, err := enforcer.Evaluate(context.Background(), quota.EvalParams{
		UserID:             userID,
		PlanID:             planID,
		LimitKey:           "max_transactions",
		MonthlyLimit:       100,
		MonthlyPeriodStart: periodStart,
		MonthlyPeriodEnd:   periodEnd,
		Now:                now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Allowed {
		t.Errorf("expected allowed=true, got false")
	}
	if res.MonthlyRemaining != 80 {
		t.Errorf("expected monthly remaining 80, got %d", res.MonthlyRemaining)
	}
	if res.Effective != 80 {
		t.Errorf("expected effective 80, got %d", res.Effective)
	}
	if res.Status != quota.StatusNormal {
		t.Errorf("expected status normal, got %s", res.Status)
	}
}

func TestEnforcer_MonthlyOnly_Exhausted(t *testing.T) {
	store := &mockCycleStore{cycles: make(map[string]*quota.PlanLimitCycle)}
	usageStore := newMockUsageCounterStore()
	enforcer := quota.NewDualCycleEnforcer(store, usageStore)

	userID := uuid.New()
	planID := uuid.New()
	now := time.Date(2026, 10, 16, 14, 30, 0, 0, time.UTC)
	periodStart := now.Add(-5 * 24 * time.Hour)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	counter, _ := usageStore.GetOrCreateCounter(context.Background(), userID, "max_transactions", quota.CycleTypeMonthly, periodStart, periodEnd)
	counter.Used = 100

	res, err := enforcer.Evaluate(context.Background(), quota.EvalParams{
		UserID:             userID,
		PlanID:             planID,
		LimitKey:           "max_transactions",
		MonthlyLimit:       100,
		MonthlyPeriodStart: periodStart,
		MonthlyPeriodEnd:   periodEnd,
		Now:                now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Allowed {
		t.Errorf("expected allowed=false, got true")
	}
	if res.BlockReason != quota.BlockReasonMonthlyExhausted {
		t.Errorf("expected block reason monthly_exhausted, got %s", res.BlockReason)
	}
	if res.Status != quota.StatusBlocked {
		t.Errorf("expected status blocked, got %s", res.Status)
	}
}

func TestEnforcer_WeeklyMonthly_MonthlyCapsWeekly(t *testing.T) {
	planID := uuid.New()
	userID := uuid.New()
	limitKey := "ai_tokens_per_cycle"

	weeklyLimit := int64(50000)
	degradedMode := quota.DegradedModelDowngrade
	fallbackModel := "gpt-4o-mini"

	store := &mockCycleStore{
		cycles: map[string]*quota.PlanLimitCycle{
			planID.String() + ":" + limitKey: {
				PlanID:        planID,
				LimitKey:      limitKey,
				CycleType:     quota.CycleWeeklyMonthly,
				WeeklyLimit:   &weeklyLimit,
				DegradedMode:  &degradedMode,
				FallbackModel: &fallbackModel,
			},
		},
	}
	usageStore := newMockUsageCounterStore()
	enforcer := quota.NewDualCycleEnforcer(store, usageStore)

	now := time.Date(2026, 10, 16, 14, 30, 0, 0, time.UTC)
	anchor := now.Add(-2 * 24 * time.Hour) // 2 days elapsed in weekly cycle
	periodStart := now.Add(-20 * 24 * time.Hour)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	// Weekly: used 10.000 / 50.000 -> 40.000 left
	// Monthly: used 170.000 / 180.000 -> 10.000 left
	// Effective remaining MUST be 10.000 (monthly caps weekly)
	weeklyStart := anchor
	weeklyEnd := anchor.Add(7 * 24 * time.Hour)
	wCounter, _ := usageStore.GetOrCreateCounter(context.Background(), userID, limitKey, quota.CycleTypeWeekly, weeklyStart, weeklyEnd)
	wCounter.Used = 10000

	mCounter, _ := usageStore.GetOrCreateCounter(context.Background(), userID, limitKey, quota.CycleTypeMonthly, periodStart, periodEnd)
	mCounter.Used = 170000

	res, err := enforcer.Evaluate(context.Background(), quota.EvalParams{
		UserID:             userID,
		PlanID:             planID,
		LimitKey:           limitKey,
		MonthlyLimit:       180000,
		WeeklyAnchor:       &anchor,
		MonthlyPeriodStart: periodStart,
		MonthlyPeriodEnd:   periodEnd,
		Now:                now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Allowed {
		t.Errorf("expected allowed=true, got false")
	}
	if res.WeeklyRemaining != 40000 {
		t.Errorf("expected weekly remaining 40000, got %d", res.WeeklyRemaining)
	}
	if res.MonthlyRemaining != 10000 {
		t.Errorf("expected monthly remaining 10000, got %d", res.MonthlyRemaining)
	}
	if res.Effective != 10000 {
		t.Errorf("expected effective remaining 10000, got %d", res.Effective)
	}
}

func TestEnforcer_WeeklyMonthly_WeeklyExhausted_ModelDowngrade(t *testing.T) {
	planID := uuid.New()
	userID := uuid.New()
	limitKey := "ai_tokens_per_cycle"

	weeklyLimit := int64(50000)
	degradedMode := quota.DegradedModelDowngrade
	fallbackModel := "gpt-4o-mini"
	maxDegraded := int64(10000)

	store := &mockCycleStore{
		cycles: map[string]*quota.PlanLimitCycle{
			planID.String() + ":" + limitKey: {
				PlanID:           planID,
				LimitKey:         limitKey,
				CycleType:        quota.CycleWeeklyMonthly,
				WeeklyLimit:      &weeklyLimit,
				DegradedMode:     &degradedMode,
				FallbackModel:    &fallbackModel,
				MaxDegradedUsage: &maxDegraded,
			},
		},
	}
	usageStore := newMockUsageCounterStore()
	enforcer := quota.NewDualCycleEnforcer(store, usageStore)

	now := time.Date(2026, 10, 16, 14, 30, 0, 0, time.UTC)
	anchor := now.Add(-3 * 24 * time.Hour)
	periodStart := now.Add(-10 * 24 * time.Hour)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	weeklyStart := anchor
	weeklyEnd := anchor.Add(7 * 24 * time.Hour)
	wCounter, _ := usageStore.GetOrCreateCounter(context.Background(), userID, limitKey, quota.CycleTypeWeekly, weeklyStart, weeklyEnd)
	wCounter.Used = 50000 // Weekly limit reached
	wCounter.DegradedUsed = 2000

	mCounter, _ := usageStore.GetOrCreateCounter(context.Background(), userID, limitKey, quota.CycleTypeMonthly, periodStart, periodEnd)
	mCounter.Used = 80000 // Monthly still has 100.000

	res, err := enforcer.Evaluate(context.Background(), quota.EvalParams{
		UserID:             userID,
		PlanID:             planID,
		LimitKey:           limitKey,
		MonthlyLimit:       180000,
		WeeklyAnchor:       &anchor,
		MonthlyPeriodStart: periodStart,
		MonthlyPeriodEnd:   periodEnd,
		Now:                now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Allowed {
		t.Errorf("expected allowed=true in degraded mode, got false")
	}
	if !res.Degraded {
		t.Errorf("expected degraded=true, got false")
	}
	if res.DegradedMode != quota.DegradedModelDowngrade {
		t.Errorf("expected degraded_mode model_downgrade, got %s", res.DegradedMode)
	}
	if res.FallbackModel != "gpt-4o-mini" {
		t.Errorf("expected fallback model gpt-4o-mini, got %s", res.FallbackModel)
	}
	if res.Status != quota.StatusDegradedDowngrade {
		t.Errorf("expected status degraded_downgrade, got %s", res.Status)
	}
}

func TestEnforcer_WeeklyMonthly_WeeklyExhausted_HardBlock(t *testing.T) {
	planID := uuid.New()
	userID := uuid.New()
	limitKey := "ocr_scans_per_cycle"

	weeklyLimit := int64(30)
	degradedMode := quota.DegradedHardBlock

	store := &mockCycleStore{
		cycles: map[string]*quota.PlanLimitCycle{
			planID.String() + ":" + limitKey: {
				PlanID:       planID,
				LimitKey:     limitKey,
				CycleType:    quota.CycleWeeklyMonthly,
				WeeklyLimit:  &weeklyLimit,
				DegradedMode: &degradedMode,
			},
		},
	}
	usageStore := newMockUsageCounterStore()
	enforcer := quota.NewDualCycleEnforcer(store, usageStore)

	now := time.Date(2026, 10, 16, 14, 30, 0, 0, time.UTC)
	anchor := now.Add(-4 * 24 * time.Hour)
	periodStart := now.Add(-10 * 24 * time.Hour)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	weeklyStart := anchor
	weeklyEnd := anchor.Add(7 * 24 * time.Hour)
	wCounter, _ := usageStore.GetOrCreateCounter(context.Background(), userID, limitKey, quota.CycleTypeWeekly, weeklyStart, weeklyEnd)
	wCounter.Used = 30 // Weekly limit reached

	mCounter, _ := usageStore.GetOrCreateCounter(context.Background(), userID, limitKey, quota.CycleTypeMonthly, periodStart, periodEnd)
	mCounter.Used = 40 // Monthly still has quota

	res, err := enforcer.Evaluate(context.Background(), quota.EvalParams{
		UserID:             userID,
		PlanID:             planID,
		LimitKey:           limitKey,
		MonthlyLimit:       100,
		WeeklyAnchor:       &anchor,
		MonthlyPeriodStart: periodStart,
		MonthlyPeriodEnd:   periodEnd,
		Now:                now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Allowed {
		t.Errorf("expected allowed=false for hard_block, got true")
	}
	if res.BlockReason != quota.BlockReasonWeeklyExhausted {
		t.Errorf("expected block reason weekly_exhausted, got %s", res.BlockReason)
	}
	if res.Status != quota.StatusBlocked {
		t.Errorf("expected status blocked, got %s", res.Status)
	}
}

func TestEnforcer_WeeklyMonthly_DegradedCapReached(t *testing.T) {
	planID := uuid.New()
	userID := uuid.New()
	limitKey := "ai_requests_per_cycle"

	weeklyLimit := int64(200)
	degradedMode := quota.DegradedThrottle
	rateAmt := 5
	rateWindow := 60
	maxDegraded := int64(50)

	store := &mockCycleStore{
		cycles: map[string]*quota.PlanLimitCycle{
			planID.String() + ":" + limitKey: {
				PlanID:                 planID,
				LimitKey:               limitKey,
				CycleType:              quota.CycleWeeklyMonthly,
				WeeklyLimit:            &weeklyLimit,
				DegradedMode:           &degradedMode,
				ThrottleRateAmount:     &rateAmt,
				ThrottleRateWindowSecs: &rateWindow,
				MaxDegradedUsage:       &maxDegraded,
			},
		},
	}
	usageStore := newMockUsageCounterStore()
	enforcer := quota.NewDualCycleEnforcer(store, usageStore)

	now := time.Date(2026, 10, 16, 14, 30, 0, 0, time.UTC)
	anchor := now.Add(-5 * 24 * time.Hour)
	periodStart := now.Add(-10 * 24 * time.Hour)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	weeklyStart := anchor
	weeklyEnd := anchor.Add(7 * 24 * time.Hour)
	wCounter, _ := usageStore.GetOrCreateCounter(context.Background(), userID, limitKey, quota.CycleTypeWeekly, weeklyStart, weeklyEnd)
	wCounter.Used = 200
	wCounter.DegradedUsed = 50 // Cap reached!

	res, err := enforcer.Evaluate(context.Background(), quota.EvalParams{
		UserID:             userID,
		PlanID:             planID,
		LimitKey:           limitKey,
		MonthlyLimit:       1000,
		WeeklyAnchor:       &anchor,
		MonthlyPeriodStart: periodStart,
		MonthlyPeriodEnd:   periodEnd,
		Now:                now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Allowed {
		t.Errorf("expected allowed=false when degraded cap reached, got true")
	}
	if res.BlockReason != quota.BlockReasonDegradedCap {
		t.Errorf("expected block reason degraded_cap, got %s", res.BlockReason)
	}
}

func TestEnforcer_UnlimitedBypass(t *testing.T) {
	store := &mockCycleStore{cycles: make(map[string]*quota.PlanLimitCycle)}
	usageStore := newMockUsageCounterStore()
	enforcer := quota.NewDualCycleEnforcer(store, usageStore)

	userID := uuid.New()
	planID := uuid.New()
	now := time.Now().UTC()

	res, err := enforcer.Evaluate(context.Background(), quota.EvalParams{
		UserID:             userID,
		PlanID:             planID,
		LimitKey:           "max_transactions",
		MonthlyLimit:       -1, // Unlimited
		MonthlyPeriodStart: now.Add(-24 * time.Hour),
		MonthlyPeriodEnd:   now.Add(24 * time.Hour),
		Now:                now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Allowed {
		t.Errorf("expected allowed=true for unlimited, got false")
	}
	if res.Status != quota.StatusNormal {
		t.Errorf("expected status normal, got %s", res.Status)
	}
}
func TestEnforcer_RecordWeeklyUsage(t *testing.T) {
	planID := uuid.New()
	userID := uuid.New()
	limitKey := "max_transactions"
	weeklyLimit := int64(200)
	degMode := quota.DegradedHardBlock

	store := &mockCycleStore{
		cycles: map[string]*quota.PlanLimitCycle{
			planID.String() + ":" + limitKey: {
				PlanID:       planID,
				LimitKey:     limitKey,
				CycleType:    quota.CycleWeeklyMonthly,
				WeeklyLimit:  &weeklyLimit,
				DegradedMode: &degMode,
			},
		},
	}
	usageStore := newMockUsageCounterStore()
	enforcer := quota.NewDualCycleEnforcer(store, usageStore)

	now := time.Now().UTC()
	anchor := now.Add(-2 * 24 * time.Hour)

	// Record 5 usages
	err := enforcer.RecordWeeklyUsage(context.Background(), userID, planID, limitKey, 5, &anchor, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	counters, err := usageStore.GetCurrentCounters(context.Background(), userID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(counters) != 1 {
		t.Fatalf("expected 1 counter, got %d", len(counters))
	}
	if counters[0].Used != 5 {
		t.Errorf("expected used 5, got %d", counters[0].Used)
	}
}
