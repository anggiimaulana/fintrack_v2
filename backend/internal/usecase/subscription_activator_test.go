package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/pkg/pagination"
)

// fakeSubRepo is a one-row SubscriptionRepository for activator tests.
type fakeSubRepo struct {
	sub *billing.Subscription
}

func (f *fakeSubRepo) GetByUserID(context.Context, uuid.UUID) (*billing.Subscription, error) {
	if f.sub == nil {
		return nil, billing.ErrSubscriptionNotFound
	}
	return f.sub, nil
}
func (f *fakeSubRepo) Create(_ context.Context, s *billing.Subscription) error { f.sub = s; return nil }
func (f *fakeSubRepo) Update(_ context.Context, s *billing.Subscription) error { f.sub = s; return nil }
func (f *fakeSubRepo) RecordEvent(context.Context, *billing.SubscriptionEvent) error {
	return nil
}
func (f *fakeSubRepo) ListEvents(context.Context, uuid.UUID, int) ([]*billing.SubscriptionEvent, error) {
	return nil, nil
}
func (f *fakeSubRepo) ListEventsPaged(context.Context, uuid.UUID, *pagination.Query) ([]*billing.SubscriptionEvent, int, error) {
	return nil, 0, nil
}

// fakePlanRepo returns plans by code.
type fakePlanRepo struct {
	byCode map[string]*billing.Plan
}

func (f *fakePlanRepo) GetPlanByCode(_ context.Context, code string) (*billing.Plan, error) {
	if p, ok := f.byCode[code]; ok {
		return p, nil
	}
	return nil, billing.ErrPlanNotFound
}
func (f *fakePlanRepo) GetPlanByID(context.Context, uuid.UUID) (*billing.Plan, error) {
	return nil, billing.ErrPlanNotFound
}
func (f *fakePlanRepo) ListActivePlans(context.Context) ([]*billing.Plan, error) { return nil, nil }
func (f *fakePlanRepo) GetEntitlements(context.Context, uuid.UUID) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (f *fakePlanRepo) GetLimits(context.Context, uuid.UUID) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func newActivatorFixture(t *testing.T, sub *billing.Subscription) (*SubscriptionActivator, *fakeSubRepo, *billing.Plan, *billing.Plan) {
	t.Helper()
	repo := &fakeSubRepo{sub: sub}
	freeID, proID := uuid.New(), uuid.New()
	plans := &fakePlanRepo{byCode: map[string]*billing.Plan{
		"free": {ID: freeID, Code: "free", BillingPeriodDays: 30},
		"pro":  {ID: proID, Code: "pro", BillingPeriodDays: 30},
	}}
	return NewSubscriptionActivator(repo, plans, nil), repo, plans.byCode["free"], plans.byCode["pro"]
}

// A same-plan payment while the cycle is running RENEWS: it stacks onto
// current_period_end so the user keeps the days they already paid for.
func TestActivateForUser_SamePlanRenewalStacks(t *testing.T) {
	now := time.Now().UTC()
	cycleEnd := now.Add(10 * 24 * time.Hour)
	sub := &billing.Subscription{
		ID: uuid.New(), UserID: uuid.New(), Status: billing.StatusActive,
		CurrentPeriodStart: now.Add(-20 * 24 * time.Hour), CurrentPeriodEnd: &cycleEnd,
	}
	act, repo, _, pro := newActivatorFixture(t, sub)
	sub.PlanID = pro.ID

	if err := act.ActivateForUser(context.Background(), sub.UserID, "pro"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	// Renewal must start at the OLD end, not now.
	if !repo.sub.CurrentPeriodStart.Equal(cycleEnd) {
		t.Fatalf("renewal must stack from the old period end: got %s want %s",
			repo.sub.CurrentPeriodStart, cycleEnd)
	}
	wantEnd := cycleEnd.AddDate(0, 0, 30)
	if repo.sub.CurrentPeriodEnd == nil || !repo.sub.CurrentPeriodEnd.Equal(wantEnd) {
		t.Fatalf("renewal end = %v, want %s", repo.sub.CurrentPeriodEnd, wantEnd)
	}
}

// A plan change mid-cycle KEEPS the cycle bounds: the user paid the prorated
// difference for the REMAINING days, so resetting would grant a free full cycle.
func TestActivateForUser_PlanChangeKeepsCycle(t *testing.T) {
	now := time.Now().UTC()
	cycleStart := now.Add(-5 * 24 * time.Hour)
	cycleEnd := now.Add(25 * 24 * time.Hour)
	sub := &billing.Subscription{
		ID: uuid.New(), UserID: uuid.New(), Status: billing.StatusActive,
		CurrentPeriodStart: cycleStart, CurrentPeriodEnd: &cycleEnd,
	}
	act, repo, free, _ := newActivatorFixture(t, sub)
	sub.PlanID = free.ID

	if err := act.ActivateForUser(context.Background(), sub.UserID, "pro"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if !repo.sub.CurrentPeriodStart.Equal(cycleStart) {
		t.Fatalf("plan change must keep the cycle start: got %s want %s",
			repo.sub.CurrentPeriodStart, cycleStart)
	}
	if repo.sub.CurrentPeriodEnd == nil || !repo.sub.CurrentPeriodEnd.Equal(cycleEnd) {
		t.Fatalf("plan change must keep the cycle end: got %v want %s",
			repo.sub.CurrentPeriodEnd, cycleEnd)
	}
}

// A payment after the cycle lapsed starts a FRESH cycle from now.
func TestActivateForUser_LapsedCycleStartsFresh(t *testing.T) {
	now := time.Now().UTC()
	lapsedEnd := now.Add(-3 * 24 * time.Hour)
	sub := &billing.Subscription{
		ID: uuid.New(), UserID: uuid.New(), Status: billing.StatusPastDue,
		CurrentPeriodStart: now.Add(-33 * 24 * time.Hour), CurrentPeriodEnd: &lapsedEnd,
	}
	act, repo, _, pro := newActivatorFixture(t, sub)
	sub.PlanID = pro.ID

	before := time.Now().UTC()
	if err := act.ActivateForUser(context.Background(), sub.UserID, "pro"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if repo.sub.CurrentPeriodStart.Before(before) {
		t.Fatalf("lapsed cycle must restart from now: got %s", repo.sub.CurrentPeriodStart)
	}
	if repo.sub.Status != billing.StatusActive {
		t.Fatalf("status = %s, want active", repo.sub.Status)
	}
}

// Paying clears a scheduled downgrade and re-activates a lapsed subscription.
func TestActivateForUser_ClearsScheduledDowngrade(t *testing.T) {
	now := time.Now().UTC()
	cycleEnd := now.Add(10 * 24 * time.Hour)
	sub := &billing.Subscription{
		ID: uuid.New(), UserID: uuid.New(), Status: billing.StatusActive,
		CurrentPeriodStart: now.Add(-20 * 24 * time.Hour), CurrentPeriodEnd: &cycleEnd,
		CancelAtPeriodEnd: true,
		Metadata:          map[string]any{"scheduled_plan_code": "free", "scheduled_plan_id": uuid.New().String()},
	}
	act, repo, _, pro := newActivatorFixture(t, sub)
	sub.PlanID = pro.ID

	if err := act.ActivateForUser(context.Background(), sub.UserID, "pro"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if repo.sub.CancelAtPeriodEnd {
		t.Fatal("paying must clear cancel_at_period_end")
	}
	if _, ok := repo.sub.Metadata["scheduled_plan_code"]; ok {
		t.Fatal("paying must clear the scheduled downgrade")
	}
}

// No subscription yet: one is created starting now.
func TestActivateForUser_CreatesSubscription(t *testing.T) {
	act, repo, _, _ := newActivatorFixture(t, nil)
	user := uuid.New()
	if err := act.ActivateForUser(context.Background(), user, "pro"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if repo.sub == nil {
		t.Fatal("subscription must be created")
	}
	if repo.sub.Status != billing.StatusActive || repo.sub.CurrentPeriodEnd == nil {
		t.Fatalf("created subscription must be active with an end: %+v", repo.sub)
	}
}
