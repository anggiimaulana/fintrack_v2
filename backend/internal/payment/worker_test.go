package payment

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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

// TestDunningWorker_RemindersAndDedup verifies H-3/H-1 reminders are emitted
// once per bucket and never duplicated.
func TestDunningWorker_RemindersAndDedup(t *testing.T) {
	repo := newMemRepo()
	notifier := &recNotifier{}
	w := NewDunningWorker(repo, notifier, time.Hour)

	// Seed an invoice due in 48h (=> H-3 "soon" bucket).
	due := time.Now().UTC().Add(48 * time.Hour)
	userID := uuid.New()
	_ = repo.CreateInvoice(context.Background(), &Invoice{ID: uuid.New(), UserID: userID, Status: InvoiceIssued, DueAt: &due, TotalIDR: 1000})

	w.RunOnce(context.Background())
	w.RunOnce(context.Background()) // second run must not re-notify

	if notifier.count() != 1 {
		t.Fatalf("expected exactly 1 reminder (deduped), got %d", notifier.count())
	}
	if notifier.sent[0].Type != "SUBSCRIPTION_EXPIRING_SOON" {
		t.Fatalf("expected SOON bucket, got %s", notifier.sent[0].Type)
	}
}

// TestDunningBucket verifies bucket boundaries.
func TestDunningBucket(t *testing.T) {
	cases := map[int]string{
		100: "", // beyond the H-3 window -> no reminder
		72:  "SUBSCRIPTION_EXPIRING_SOON",
		48:  "SUBSCRIPTION_EXPIRING_SOON",
		24:  "SUBSCRIPTION_EXPIRING_URGENT",
		5:   "SUBSCRIPTION_EXPIRING_URGENT",
		0:   "SUBSCRIPTION_EXPIRING_DUE",
		-10: "SUBSCRIPTION_EXPIRING_DUE",
	}
	for h, want := range cases {
		if got := dunningBucket(h); got != want {
			t.Errorf("dunningBucket(%d)=%s want %s", h, got, want)
		}
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

// TestRealtimePublisher_Disabled verifies graceful no-op without Redis.
func TestRealtimePublisher_Disabled(t *testing.T) {
	p := NewRealtimePublisher(nil)
	if err := p.PublishPaymentSettled(context.Background(), "u1", PaymentEvent{AmountIDR: 100}); err == nil {
		t.Fatal("expected error when pub/sub disabled")
	}
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
