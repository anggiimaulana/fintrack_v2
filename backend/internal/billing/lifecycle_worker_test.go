package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeDueLister returns a fixed due-list (or an error).
type fakeDueLister struct {
	due []DueSubscription
	err error
}

func (f *fakeDueLister) ListSubscriptionsDueForLifecycle(context.Context, time.Time) ([]DueSubscription, error) {
	return f.due, f.err
}

type transitionCall struct {
	user      uuid.UUID
	to        SubscriptionStatus
	eventType string
}

// TestLifecycleOutcome covers the end-of-cycle policy matrix.
func TestLifecycleOutcome(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	grace := 72 * time.Hour

	cases := []struct {
		name string
		due  DueSubscription
		want SubscriptionStatus
	}{
		{
			name: "unpaid inside grace -> past_due",
			due:  DueSubscription{Status: StatusActive, PeriodEnd: now.Add(-1 * time.Hour)},
			want: StatusPastDue,
		},
		{
			name: "unpaid beyond grace -> expired",
			due:  DueSubscription{Status: StatusActive, PeriodEnd: now.Add(-100 * time.Hour)},
			want: StatusExpired,
		},
		{
			name: "past_due beyond grace -> expired",
			due:  DueSubscription{Status: StatusPastDue, PeriodEnd: now.Add(-100 * time.Hour)},
			want: StatusExpired,
		},
		{
			name: "past_due inside grace -> no-op (already flagged)",
			due:  DueSubscription{Status: StatusPastDue, PeriodEnd: now.Add(-1 * time.Hour)},
			want: "",
		},
		{
			name: "user cancelled -> canceled immediately",
			due:  DueSubscription{Status: StatusActive, PeriodEnd: now.Add(-1 * time.Minute), CancelAtPeriodEnd: true},
			want: StatusCanceled,
		},
		{
			name: "scheduled plan change -> owned by downgrade worker",
			due:  DueSubscription{Status: StatusActive, PeriodEnd: now.Add(-100 * time.Hour), HasScheduledPlan: true},
			want: "",
		},
		{
			name: "already expired -> no-op",
			due:  DueSubscription{Status: StatusExpired, PeriodEnd: now.Add(-100 * time.Hour)},
			want: "",
		},
		{
			name: "already canceled -> no-op",
			due:  DueSubscription{Status: StatusCanceled, PeriodEnd: now.Add(-100 * time.Hour)},
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _, _ := lifecycleOutcome(c.due, now, grace)
			if got != c.want {
				t.Fatalf("outcome = %q, want %q", got, c.want)
			}
		})
	}
}

// TestLifecycleWorker_AppliesTransitions verifies each due subscription is
// transitioned once, with the right target.
func TestLifecycleWorker_AppliesTransitions(t *testing.T) {
	now := time.Now().UTC()
	expireMe := uuid.New()
	cancelMe := uuid.New()
	skipMe := uuid.New() // scheduled plan change -> skipped
	var calls []transitionCall

	w := NewLifecycleWorker(
		&fakeDueLister{due: []DueSubscription{
			{UserID: expireMe, Status: StatusActive, PeriodEnd: now.Add(-100 * time.Hour)},
			{UserID: cancelMe, Status: StatusActive, PeriodEnd: now.Add(-1 * time.Minute), CancelAtPeriodEnd: true},
			{UserID: skipMe, Status: StatusActive, PeriodEnd: now.Add(-100 * time.Hour), HasScheduledPlan: true},
		}},
		func(_ context.Context, uid uuid.UUID, to SubscriptionStatus, eventType, _ string) error {
			calls = append(calls, transitionCall{user: uid, to: to, eventType: eventType})
			return nil
		},
		func(context.Context) int64 { return 72 },
		LifecycleConfig{},
	)
	w.RunOnce(context.Background())

	byUser := map[uuid.UUID]SubscriptionStatus{}
	for _, c := range calls {
		byUser[c.user] = c.to
	}
	if byUser[expireMe] != StatusExpired {
		t.Fatalf("expireMe -> %q, want expired", byUser[expireMe])
	}
	if byUser[cancelMe] != StatusCanceled {
		t.Fatalf("cancelMe -> %q, want canceled", byUser[cancelMe])
	}
	if _, ok := byUser[skipMe]; ok {
		t.Fatal("scheduled plan change must be skipped by the lifecycle worker")
	}
}

// A list error is a safe no-op.
func TestLifecycleWorker_ListErrorIsSafe(t *testing.T) {
	w := NewLifecycleWorker(
		&fakeDueLister{err: errors.New("boom")},
		func(context.Context, uuid.UUID, SubscriptionStatus, string, string) error {
			t.Fatal("transition must not be called on list error")
			return nil
		},
		nil,
		LifecycleConfig{},
	)
	w.RunOnce(context.Background())
}

// One failing transition does not abort the sweep.
func TestLifecycleWorker_ContinuesAfterFailure(t *testing.T) {
	now := time.Now().UTC()
	a, b := uuid.New(), uuid.New()
	attempted := map[uuid.UUID]bool{}
	w := NewLifecycleWorker(
		&fakeDueLister{due: []DueSubscription{
			{UserID: a, Status: StatusActive, PeriodEnd: now.Add(-100 * time.Hour)},
			{UserID: b, Status: StatusActive, PeriodEnd: now.Add(-100 * time.Hour)},
		}},
		func(_ context.Context, uid uuid.UUID, _ SubscriptionStatus, _, _ string) error {
			attempted[uid] = true
			if uid == a {
				return errors.New("boom")
			}
			return nil
		},
		nil,
		LifecycleConfig{},
	)
	w.RunOnce(context.Background())
	if !attempted[a] || !attempted[b] {
		t.Fatalf("both must be attempted, got %v", attempted)
	}
}
