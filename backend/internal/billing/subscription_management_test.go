package billing

import (
	"testing"
	"time"
)

// TestCanTransition covers the legal/illegal state matrix.
func TestCanTransition(t *testing.T) {
	legal := [][2]SubscriptionStatus{
		{StatusTrialing, StatusActive},
		{StatusTrialing, StatusExpired},
		{StatusActive, StatusPastDue},
		{StatusActive, StatusCanceled},
		{StatusPastDue, StatusActive},
		{StatusPastDue, StatusGrace},
		{StatusPastDue, StatusExpired},
		{StatusGrace, StatusActive},
		{StatusGrace, StatusExpired},
		{StatusCanceled, StatusExpired},
	}
	for _, c := range legal {
		if !CanTransition(c[0], c[1]) {
			t.Errorf("expected %s -> %s legal", c[0], c[1])
		}
	}

	illegal := [][2]SubscriptionStatus{
		{StatusExpired, StatusPastDue}, // terminal
		{StatusExpired, StatusActive},
		{StatusCanceled, StatusActive},
		{StatusActive, StatusGrace}, // must go via past_due
		{StatusTrialing, StatusPastDue},
	}
	for _, c := range illegal {
		if CanTransition(c[0], c[1]) {
			t.Errorf("expected %s -> %s illegal", c[0], c[1])
		}
	}
}

// TestCalculateUpgradeProration verifies integer floor math.
func TestCalculateUpgradeProration(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 30)
	sub := &Subscription{CurrentPeriodStart: start, CurrentPeriodEnd: &end}
	oldPlan := &Plan{PriceMonthlyIDR: 39000}
	newPlan := &Plan{PriceMonthlyIDR: 129000}

	// Halfway through the cycle (15 days remaining).
	now := start.AddDate(0, 0, 15)
	p := CalculateUpgradeProration(sub, oldPlan, newPlan, now)

	if p.DaysTotal != 30 || p.DaysRemaining != 15 {
		t.Fatalf("expected 30/15 days, got %d/%d", p.DaysTotal, p.DaysRemaining)
	}
	// oldCredit = 39000*15/30 = 19500; newCost = 129000*15/30 = 64500; due = 45000.
	if p.OldCreditIDR != 19500 {
		t.Fatalf("expected old credit 19500, got %d", p.OldCreditIDR)
	}
	if p.NewCostIDR != 64500 {
		t.Fatalf("expected new cost 64500, got %d", p.NewCostIDR)
	}
	if p.AmountDueIDR != 45000 {
		t.Fatalf("expected amount due 45000, got %d", p.AmountDueIDR)
	}
}

// TestCalculateUpgradeProration_NeverNegative verifies amount due floors at 0.
func TestCalculateUpgradeProration_NeverNegative(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 30)
	sub := &Subscription{CurrentPeriodStart: start, CurrentPeriodEnd: &end}
	// Downgrade scenario: new plan cheaper -> credit exceeds cost.
	oldPlan := &Plan{PriceMonthlyIDR: 129000}
	newPlan := &Plan{PriceMonthlyIDR: 39000}
	now := start.AddDate(0, 0, 20)
	p := CalculateUpgradeProration(sub, oldPlan, newPlan, now)
	if p.AmountDueIDR != 0 {
		t.Fatalf("amount due must never be negative, got %d", p.AmountDueIDR)
	}
}

// TestCalculateUpgradeProration_FullRemaining verifies no time elapsed.
func TestCalculateUpgradeProration_FullRemaining(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 30)
	sub := &Subscription{CurrentPeriodStart: start, CurrentPeriodEnd: &end}
	oldPlan := &Plan{PriceMonthlyIDR: 0}
	newPlan := &Plan{PriceMonthlyIDR: 39000}
	p := CalculateUpgradeProration(sub, oldPlan, newPlan, start)
	if p.AmountDueIDR != 39000 {
		t.Fatalf("expected full price 39000 at cycle start, got %d", p.AmountDueIDR)
	}
}

// TestCalculateUpgradeProration_NoPeriodEnd verifies nil period end is safe.
func TestCalculateUpgradeProration_NoPeriodEnd(t *testing.T) {
	p := CalculateUpgradeProration(&Subscription{}, nil, &Plan{PriceMonthlyIDR: 100}, time.Now())
	if p.AmountDueIDR != 0 {
		t.Fatalf("nil period end must yield 0, got %d", p.AmountDueIDR)
	}
}
