package payment

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/contracts"
)

// recNotifier captures notifications.
type recNotifier struct {
	mu   sync.Mutex
	sent []Notification
}

func (r *recNotifier) Notify(_ context.Context, n Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, n)
	return nil
}
func (r *recNotifier) count() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.sent) }

// TestDunningWorker_RemindsSubscriptions verifies reminders follow the
// subscription period (default H-7 onto the SOON bucket), not the invoice due
// date, and carry a stable per-period dedupe key.
func TestDunningWorker_RemindsSubscriptions(t *testing.T) {
	repo := newMemRepo()
	notifier := &recNotifier{}
	w := NewDunningWorker(repo, nil, notifier, time.Hour)

	userID := uuid.New()
	end := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	repo.subs = []SubscriptionExpiry{{UserID: userID, PeriodEnd: end}}

	w.RunOnce(context.Background())

	if notifier.count() != 1 {
		t.Fatalf("expected 1 reminder, got %d", notifier.count())
	}
	got := notifier.sent[0]
	if got.Type != "SUBSCRIPTION_EXPIRING_SOON" {
		t.Fatalf("expected SOON bucket, got %s", got.Type)
	}
	want := "subscription:" + userID.String() + ":" + strconv.FormatInt(end.Unix(), 10) + ":SUBSCRIPTION_EXPIRING_SOON"
	if got.DedupeKey != want {
		t.Fatalf("dedupe key = %q, want %q", got.DedupeKey, want)
	}
}

// TestDunningWorker_EndedSubscriptionIsIgnored verifies a period that already
// ended never produces a reminder (the "still notified after expiry" bug).
func TestDunningWorker_EndedSubscriptionIsIgnored(t *testing.T) {
	repo := newMemRepo()
	notifier := &recNotifier{}
	w := NewDunningWorker(repo, nil, notifier, time.Hour)
	repo.subs = []SubscriptionExpiry{{UserID: uuid.New(), PeriodEnd: time.Now().UTC().Add(-time.Hour)}}
	w.RunOnce(context.Background())
	if notifier.count() != 0 {
		t.Fatalf("ended subscription must not be reminded, got %d", notifier.count())
	}
}

// TestDunningBucket verifies bucket boundaries (soon defaults to H-7).
func TestDunningBucket(t *testing.T) {
	cases := map[int]string{
		200: "", // beyond the H-7 window -> no reminder
		168: "SUBSCRIPTION_EXPIRING_SOON",
		48:  "SUBSCRIPTION_EXPIRING_SOON",
		24:  "SUBSCRIPTION_EXPIRING_URGENT",
		5:   "SUBSCRIPTION_EXPIRING_URGENT",
		0:   "SUBSCRIPTION_EXPIRING_URGENT",
		-1:  "",
	}
	for h, want := range cases {
		if got := dunningBucket(h, 24, 168); got != want {
			t.Errorf("dunningBucket(%d)=%s want %s", h, got, want)
		}
	}
}

// TestDunningWorker_ConfigurableThresholds verifies the reminder offsets come
// from settings, so an operator can retune the cadence without a redeploy.
func TestDunningWorker_ConfigurableThresholds(t *testing.T) {
	repo := newMemRepo()
	notifier := &recNotifier{}
	settings := newMemSettings()
	settings.m["dunning_hours_urgent"] = int64(12)
	settings.m["dunning_hours_soon"] = int64(36)
	w := NewDunningWorker(repo, settings, notifier, time.Hour)

	// A period ending in 30h is SOON (>12h urgent, <=36h soon).
	repo.subs = []SubscriptionExpiry{{UserID: uuid.New(), PeriodEnd: time.Now().UTC().Add(30 * time.Hour)}}
	w.RunOnce(context.Background())
	if notifier.count() != 1 || notifier.sent[0].Type != "SUBSCRIPTION_EXPIRING_SOON" {
		t.Fatalf("expected SOON reminder under custom thresholds, got %+v", notifier.sent)
	}

	// With a 20h horizon the 30h subscription must NOT be picked up at all.
	notifier2 := &recNotifier{}
	settings.m["dunning_hours_soon"] = int64(20)
	w2 := NewDunningWorker(repo, settings, notifier2, time.Hour)
	w2.RunOnce(context.Background())
	if notifier2.count() != 0 {
		t.Fatalf("expected no reminder with 20h horizon, got %d", notifier2.count())
	}
}

// TestDunningThresholds_Monotonic verifies urgent is clamped to soon so the
// buckets can never invert (which would strand subscriptions in an unreachable band).
func TestDunningThresholds_Monotonic(t *testing.T) {
	settings := newMemSettings()
	settings.m["dunning_hours_urgent"] = int64(90)
	settings.m["dunning_hours_soon"] = int64(24)
	w := NewDunningWorker(newMemRepo(), settings, nil, time.Hour)
	urgent, soon := w.dunningThresholds(context.Background())
	if urgent != soon {
		t.Fatalf("expected urgent clamped to soon (%d), got urgent=%d soon=%d", soon, urgent, soon)
	}
}

// TestDunningWorker_NextPeriodGetsNewDedupeKey verifies a renewed subscription
// (new period end) is reminded again instead of being silenced forever.
func TestDunningWorker_NextPeriodGetsNewDedupeKey(t *testing.T) {
	repo := newMemRepo()
	notifier := &recNotifier{}
	w := NewDunningWorker(repo, nil, notifier, time.Hour)
	userID := uuid.New()

	repo.subs = []SubscriptionExpiry{{UserID: userID, PeriodEnd: time.Now().UTC().Add(48 * time.Hour)}}
	w.RunOnce(context.Background())
	firstKey := notifier.sent[0].DedupeKey

	repo.subs = []SubscriptionExpiry{{UserID: userID, PeriodEnd: time.Now().UTC().Add(100 * time.Hour)}}
	w.RunOnce(context.Background())
	if notifier.count() != 2 {
		t.Fatalf("expected the renewed period to be reminded too, got %d", notifier.count())
	}
	if notifier.sent[1].DedupeKey == firstKey {
		t.Fatal("a new period must produce a new dedupe key")
	}
}

// TestGraceWorker_Transitions verifies grace -> past_due transitions.
func TestGraceWorker_Transitions(t *testing.T) {
	repo := newMemRepo()
	s := newMemSettings()
	s.m["grace_period_hours"] = int64(72)

	var mu sync.Mutex
	statuses := map[string]string{}
	setStatus := func(_ context.Context, userID, status string) error {
		mu.Lock()
		statuses[userID] = status
		mu.Unlock()
		return nil
	}
	w := NewGraceWorker(repo, s, setStatus, time.Hour)

	// Within grace (10h past due).
	userA := uuid.New()
	dueA := time.Now().UTC().Add(-10 * time.Hour)
	_ = repo.CreateInvoice(context.Background(), &Invoice{ID: uuid.New(), UserID: userA, Status: InvoiceIssued, DueAt: &dueA, TotalIDR: 1000})

	// Beyond grace (100h past due).
	userB := uuid.New()
	dueB := time.Now().UTC().Add(-100 * time.Hour)
	_ = repo.CreateInvoice(context.Background(), &Invoice{ID: uuid.New(), UserID: userB, Status: InvoiceIssued, DueAt: &dueB, TotalIDR: 1000})

	w.RunOnce(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if statuses[userA.String()] != "grace" {
		t.Fatalf("user A (within grace) expected 'grace', got %q", statuses[userA.String()])
	}
	if statuses[userB.String()] != "past_due" {
		t.Fatalf("user B (beyond grace) expected 'past_due', got %q", statuses[userB.String()])
	}
}

// TestNoopEmitter_IsSafe verifies emitting without a wired transport is a no-op
// (the default state in tests and when REALTIME_ENABLED=off).
func TestNoopEmitter_IsSafe(t *testing.T) {
	var e RealtimeEmitter = noopEmitter{}
	e.PublishUser(context.Background(), uuid.New(), contracts.Event{Event: contracts.EventPaymentReceived})
	e.PublishAdmins(context.Background(), contracts.Event{Event: contracts.EventManualApprovalNeeded})
}

// TestAuditInterceptor_NilSafe verifies nil recorder is safe.
func TestAuditInterceptor_NilSafe(t *testing.T) {
	a := NewAuditInterceptor(nil)
	// Must not panic.
	a.PaymentCreate(context.Background(), uuid.New(), 100)
	a.ManualReview(context.Background(), uuid.New(), uuid.New(), true, "")
}

// TestAuditInterceptor_Records verifies events reach the recorder.
func TestAuditInterceptor_Records(t *testing.T) {
	rec := &recAudit{}
	a := NewAuditInterceptor(rec)
	pid := uuid.New()
	a.PaymentSettled(context.Background(), pid, 39000)
	if len(rec.events) != 1 || rec.events[0].Action != "PAYMENT_SETTLED" {
		t.Fatalf("expected 1 PAYMENT_SETTLED event, got %+v", rec.events)
	}
}

type recAudit struct{ events []AuditEvent }

func (r *recAudit) Record(_ context.Context, e AuditEvent) error {
	r.events = append(r.events, e)
	return nil
}
