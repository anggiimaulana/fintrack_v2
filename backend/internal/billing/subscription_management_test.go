package billing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/pagination"
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

// fakeLifecycleSubRepo is a one-row SubscriptionRepository for lifecycle tests.
type fakeLifecycleSubRepo struct {
	sub    *Subscription
	events []*SubscriptionEvent
}

func (f *fakeLifecycleSubRepo) GetByUserID(context.Context, uuid.UUID) (*Subscription, error) {
	if f.sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	return f.sub, nil
}
func (f *fakeLifecycleSubRepo) Create(_ context.Context, s *Subscription) error { f.sub = s; return nil }
func (f *fakeLifecycleSubRepo) Update(_ context.Context, s *Subscription) error { f.sub = s; return nil }
func (f *fakeLifecycleSubRepo) RecordEvent(_ context.Context, ev *SubscriptionEvent) error {
	f.events = append(f.events, ev)
	return nil
}
func (f *fakeLifecycleSubRepo) ListEvents(context.Context, uuid.UUID, int) ([]*SubscriptionEvent, error) {
	return f.events, nil
}
func (f *fakeLifecycleSubRepo) ListEventsPaged(context.Context, uuid.UUID, *pagination.Query) ([]*SubscriptionEvent, int, error) {
	return f.events, len(f.events), nil
}

// TestTransitionByUserID_ExpiresLapsedSubscription is the regression guard for
// "nothing moves a non-renewed subscription out of active": the worker's
// entry point must expire it (and record the event).
func TestTransitionByUserID_ExpiresLapsedSubscription(t *testing.T) {
	repo := &fakeLifecycleSubRepo{sub: &Subscription{
		ID: uuid.New(), UserID: uuid.New(), Status: StatusActive,
	}}
	svc := NewLifecycleService(repo, fakeTx{}, nil)

	if err := svc.TransitionByUserID(context.Background(), repo.sub.UserID, StatusExpired, "expired", "cycle ended"); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if repo.sub.Status != StatusExpired {
		t.Fatalf("status = %s, want expired", repo.sub.Status)
	}
	if len(repo.events) != 1 || repo.events[0].EventType != "expired" {
		t.Fatalf("expected one 'expired' event, got %+v", repo.events)
	}
}

// A canceled-at-period-end subscription becomes `canceled`.
func TestTransitionByUserID_CancelsAtPeriodEnd(t *testing.T) {
	repo := &fakeLifecycleSubRepo{sub: &Subscription{
		ID: uuid.New(), UserID: uuid.New(), Status: StatusActive, CancelAtPeriodEnd: true,
	}}
	svc := NewLifecycleService(repo, fakeTx{}, nil)

	if err := svc.TransitionByUserID(context.Background(), repo.sub.UserID, StatusCanceled, "canceled", "cancel_at_period_end"); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if repo.sub.Status != StatusCanceled {
		t.Fatalf("status = %s, want canceled", repo.sub.Status)
	}
	if repo.sub.CanceledAt == nil {
		t.Fatal("canceled_at must be set")
	}
}

// Idempotent: re-applying the same transition is a no-op, not an error (a sweep
// may observe a subscription another worker just settled).
func TestTransitionByUserID_Idempotent(t *testing.T) {
	repo := &fakeLifecycleSubRepo{sub: &Subscription{
		ID: uuid.New(), UserID: uuid.New(), Status: StatusExpired,
	}}
	svc := NewLifecycleService(repo, fakeTx{}, nil)
	if err := svc.TransitionByUserID(context.Background(), repo.sub.UserID, StatusExpired, "expired", ""); err != nil {
		t.Fatalf("re-applying a terminal status must be a no-op, got %v", err)
	}
}

// A missing subscription is a no-op, not an error.
func TestTransitionByUserID_NoSubscriptionIsNoop(t *testing.T) {
	repo := &fakeLifecycleSubRepo{}
	svc := NewLifecycleService(repo, fakeTx{}, nil)
	if err := svc.TransitionByUserID(context.Background(), uuid.New(), StatusExpired, "expired", ""); err != nil {
		t.Fatalf("missing subscription must be a no-op, got %v", err)
	}
}
