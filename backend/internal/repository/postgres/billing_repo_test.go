package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// TestBillingRepo_PlansAndEntitlements is env-gated. Verifies seeded plans,
// entitlements, limits, and pricing load correctly.
func TestBillingRepo_PlansAndEntitlements(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	ctx := context.Background()

	plans, err := repo.ListActivePlans(ctx)
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	if len(plans) < 3 {
		t.Fatalf("expected >= 3 seeded plans, got %d", len(plans))
	}

	free, err := repo.GetPlanByCode(ctx, "free")
	if err != nil {
		t.Fatalf("get free plan: %v", err)
	}
	if free.PriceMonthlyIDR != 0 {
		t.Fatalf("free plan must be Rp0, got %d", free.PriceMonthlyIDR)
	}
	if free.BillingPeriodDays != 30 {
		t.Fatalf("expected 30-day cycle, got %d", free.BillingPeriodDays)
	}

	ent, err := repo.GetEntitlements(ctx, free.ID)
	if err != nil {
		t.Fatalf("entitlements: %v", err)
	}
	if !ent["ai_chat"] {
		t.Fatal("free plan must allow ai_chat")
	}
	if ent["ocr_receipt"] {
		t.Fatal("free plan must NOT allow ocr_receipt")
	}

	limits, err := repo.GetLimits(ctx, free.ID)
	if err != nil {
		t.Fatalf("limits: %v", err)
	}
	if limits["ai_tokens_per_cycle"] != 5000 {
		t.Fatalf("expected free quota 5000, got %d", limits["ai_tokens_per_cycle"])
	}

	rule, err := repo.GetActiveRule(ctx)
	if err != nil {
		t.Fatalf("pricing rule: %v", err)
	}
	if rule.BasePer1kIDR != 300 {
		t.Fatalf("expected base 300, got %d", rule.BasePer1kIDR)
	}
	if rule.ModalityMultipliers["vision"] != 2.5 {
		t.Fatalf("expected vision 2.5, got %v", rule.ModalityMultipliers["vision"])
	}
}

// TestBillingRepo_TokenPeriodAndLedger verifies period creation, ledger insert,
// idempotency, and append-only enforcement at the repository level.
func TestBillingRepo_TokenPeriodAndLedger(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	ctx := context.Background()

	// Use a deterministic seeded user.
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	now := time.Now().UTC().Truncate(time.Second)

	period, err := repo.GetOrCreatePeriod(ctx, userID, now.Unix(), now.Add(30*24*time.Hour).Unix(), 1000)
	if err != nil {
		t.Fatalf("get/create period: %v", err)
	}
	if period.QuotaTokens != 1000 {
		t.Fatalf("expected quota 1000, got %d", period.QuotaTokens)
	}

	// Idempotent re-get returns the same period id.
	again, err := repo.GetOrCreatePeriod(ctx, userID, now.Unix(), now.Add(30*24*time.Hour).Unix(), 1000)
	if err != nil {
		t.Fatalf("re-get period: %v", err)
	}
	if again.ID != period.ID {
		t.Fatal("period creation must be idempotent")
	}

	// Insert a reserve entry.
	key := "itest-" + period.ID.String()
	entry := &billing.TokenLedgerEntry{
		PeriodID: period.ID, UserID: userID, EntryType: billing.EntryReserve,
		Amount: 100, Modality: "chat", IdempotencyKey: key,
	}
	if err := repo.InsertLedger(ctx, entry); err != nil {
		t.Fatalf("insert ledger: %v", err)
	}

	// Duplicate idempotency key must fail.
	dup := &billing.TokenLedgerEntry{
		PeriodID: period.ID, UserID: userID, EntryType: billing.EntryReserve,
		Amount: 100, Modality: "chat", IdempotencyKey: key,
	}
	if err := repo.InsertLedger(ctx, dup); err != billing.ErrDuplicateIdempotencyKey {
		t.Fatalf("expected ErrDuplicateIdempotencyKey, got %v", err)
	}

	// Replay lookup returns the original.
	got, err := repo.GetLedgerByIdempotencyKey(ctx, key)
	if err != nil || got == nil {
		t.Fatalf("expected ledger entry, got %v err=%v", got, err)
	}
	if got.Amount != 100 {
		t.Fatalf("expected amount 100, got %d", got.Amount)
	}

	// Adjust period tokens.
	if err := repo.AdjustPeriod(ctx, period.ID, 0, 100); err != nil {
		t.Fatalf("adjust period: %v", err)
	}
	locked, err := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
	if err != nil {
		t.Fatalf("get period for update: %v", err)
	}
	if locked == nil || locked.ReservedTokens != 100 {
		t.Fatalf("expected reserved 100, got %+v", locked)
	}
}
