package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/quota"
)

func TestQuotaRepo_GetOrCreateAndIncrement(t *testing.T) {
	pool := newTestPool(t)
	repo := NewQuotaRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, role)
		VALUES ($1, $2, 'x', 'Quota User', 'user')`,
		userID, "quota-"+userID.String()+"@test.local"); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM usage_counters WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	now := time.Now().UTC().Truncate(time.Second)
	start := now
	end := now.Add(7 * 24 * time.Hour)

	// 1. First call: inserts counter
	c1, err := repo.GetOrCreateCounter(ctx, userID, "ai_tokens_per_cycle", quota.CycleTypeWeekly, start, end)
	if err != nil {
		t.Fatalf("first GetOrCreateCounter: %v", err)
	}
	if c1.Used != 0 {
		t.Errorf("expected used 0, got %d", c1.Used)
	}

	// 2. Second call: retrieves existing counter
	c2, err := repo.GetOrCreateCounter(ctx, userID, "ai_tokens_per_cycle", quota.CycleTypeWeekly, start, end)
	if err != nil {
		t.Fatalf("second GetOrCreateCounter: %v", err)
	}
	if c1.ID != c2.ID {
		t.Errorf("expected same ID, got %s vs %s", c1.ID, c2.ID)
	}

	// 3. Increment within limit
	if err := repo.IncrementUsage(ctx, c1.ID, 50, 100); err != nil {
		t.Fatalf("increment within limit: %v", err)
	}

	// 4. Increment exceeding limit -> fails
	if err := repo.IncrementUsage(ctx, c1.ID, 60, 100); err == nil {
		t.Errorf("expected error when exceeding limit, got nil")
	}

	// 5. Increment degraded usage
	if err := repo.IncrementDegradedUsage(ctx, c1.ID, 10, 20); err != nil {
		t.Fatalf("increment degraded: %v", err)
	}
	if err := repo.IncrementDegradedUsage(ctx, c1.ID, 15, 20); err == nil {
		t.Errorf("expected error when exceeding max degraded cap, got nil")
	}
}

func TestQuotaRepo_PlanCyclesAndThreshold(t *testing.T) {
	pool := newTestPool(t)
	repo := NewQuotaRepo(pool)
	ctx := context.Background()

	planID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO plans (id, code, name, price_monthly_idr, price_yearly_idr)
		VALUES ($1, $2, 'Test Plan', 50000, 500000)`,
		planID, "tp-"+planID.String()[:8]); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM plan_limit_cycles WHERE plan_id = $1`, planID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM quota_notification_config WHERE plan_id = $1`, planID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM plans WHERE id = $1`, planID)
	})

	weeklyLimit := int64(25000)
	degMode := quota.DegradedModelDowngrade
	model := "gpt-4o-mini"
	cycles := []quota.PlanLimitCycle{
		{
			PlanID:        planID,
			LimitKey:      "ai_tokens_per_cycle",
			CycleType:     quota.CycleWeeklyMonthly,
			WeeklyLimit:   &weeklyLimit,
			DegradedMode:  &degMode,
			FallbackModel: &model,
		},
	}

	if err := repo.UpsertPlanCycles(ctx, planID, cycles); err != nil {
		t.Fatalf("UpsertPlanCycles: %v", err)
	}

	cycle, err := repo.GetPlanCycle(ctx, planID, "ai_tokens_per_cycle")
	if err != nil {
		t.Fatalf("GetPlanCycle: %v", err)
	}
	if cycle == nil || cycle.WeeklyLimit == nil || *cycle.WeeklyLimit != 25000 {
		t.Fatalf("expected weekly limit 25000, got %v", cycle)
	}

	// Warning threshold defaults to global 80
	thresh, err := repo.GetWarningThreshold(ctx, planID)
	if err != nil {
		t.Fatalf("GetWarningThreshold default: %v", err)
	}
	if thresh != 80 {
		t.Errorf("expected default 80, got %d", thresh)
	}

	// Insert override for this plan
	_, err = pool.Exec(ctx, `
		INSERT INTO quota_notification_config (plan_id, warning_threshold)
		VALUES ($1, 90)
		ON CONFLICT (plan_id) DO UPDATE SET warning_threshold = EXCLUDED.warning_threshold
	`, planID)
	if err != nil {
		t.Fatalf("insert plan threshold override: %v", err)
	}

	thresh, err = repo.GetWarningThreshold(ctx, planID)
	if err != nil {
		t.Fatalf("GetWarningThreshold override: %v", err)
	}
	if thresh != 90 {
		t.Errorf("expected overridden 90, got %d", thresh)
	}
}
