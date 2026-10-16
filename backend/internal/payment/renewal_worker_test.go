package payment

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeRenewalSource returns a fixed candidate list.
type fakeRenewalSource struct {
	due []RenewalCandidate
	err error
}

func (f *fakeRenewalSource) ListRenewalCandidates(context.Context, time.Time) ([]RenewalCandidate, error) {
	return f.due, f.err
}

// recRenewalNotifier captures renewal notifications.
type recRenewalNotifier struct {
	mu            sync.Mutex
	sent          []string
	autoCharged   []string
}

func (r *recRenewalNotifier) RenewalInvoiceIssued(_ context.Context, _ uuid.UUID, inv *Invoice, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, inv.InvoiceNumber)
	return nil
}

func (r *recRenewalNotifier) RenewalAutoCharged(_ context.Context, _ uuid.UUID, inv *Invoice, _, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.autoCharged = append(r.autoCharged, inv.InvoiceNumber)
	return nil
}

func (r *recRenewalNotifier) count() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.sent) }
func (r *recRenewalNotifier) autoChargedCount() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.autoCharged) }
func newRenewalWorker(repo *memRepo, src *fakeRenewalSource, notif *recRenewalNotifier) *RenewalWorker {
	invoices := NewInvoiceService(repo, nil, nil)
	return NewRenewalWorker(src, repo, invoices, notif, RenewalConfig{})
}

// TestRenewalWorker_IssuesInvoiceOnce verifies a due subscription gets exactly
// one renewal invoice, and a second sweep does not double-bill it.
func TestRenewalWorker_IssuesInvoiceOnce(t *testing.T) {
	repo := newMemRepo()
	notif := &recRenewalNotifier{}
	userID := uuid.New()
	subID := uuid.New()
	src := &fakeRenewalSource{due: []RenewalCandidate{{
		UserID: userID, SubscriptionID: subID, PlanCode: "pro", PlanName: "Pro",
		PriceIDR: 39000, PeriodEnd: time.Now().UTC().Add(-time.Hour),
	}}}
	w := newRenewalWorker(repo, src, notif)

	w.RunOnce(context.Background())
	if notif.count() != 1 {
		t.Fatalf("expected 1 renewal invoice, got %d", notif.count())
	}

	// Second sweep: the marker exists -> no new invoice.
	w.RunOnce(context.Background())
	if notif.count() != 1 {
		t.Fatalf("renewal must be idempotent per cycle, got %d invoices", notif.count())
	}
}

// A free plan (price 0) is never invoiced.
func TestRenewalWorker_SkipsFreePlan(t *testing.T) {
	repo := newMemRepo()
	notif := &recRenewalNotifier{}
	src := &fakeRenewalSource{due: []RenewalCandidate{{
		UserID: uuid.New(), SubscriptionID: uuid.New(), PlanCode: "free",
		PriceIDR: 0, PeriodEnd: time.Now().UTC().Add(-time.Hour),
	}}}
	w := newRenewalWorker(repo, src, notif)
	w.RunOnce(context.Background())
	if notif.count() != 0 {
		t.Fatalf("free plan must not be invoiced, got %d", notif.count())
	}
}

// A user who already has an open invoice is not billed again (the DB allows at
// most one open invoice per user).
func TestRenewalWorker_SkipsExistingOpenInvoice(t *testing.T) {
	repo := newMemRepo()
	notif := &recRenewalNotifier{}
	userID := uuid.New()
	due := time.Now().UTC().Add(24 * time.Hour)
	if err := repo.CreateInvoice(context.Background(), &Invoice{
		ID: uuid.New(), InvoiceNumber: "INV-EXISTING", UserID: userID,
		Status: InvoiceIssued, DueAt: &due, TotalIDR: 1000,
	}); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	src := &fakeRenewalSource{due: []RenewalCandidate{{
		UserID: userID, SubscriptionID: uuid.New(), PlanCode: "pro",
		PriceIDR: 39000, PeriodEnd: time.Now().UTC().Add(-time.Hour),
	}}}
	w := newRenewalWorker(repo, src, notif)
	w.RunOnce(context.Background())
	if notif.count() != 0 {
		t.Fatalf("must not issue a second open invoice, got %d", notif.count())
	}
}

// A new cycle (different period end) is billed again: the marker is per cycle.
func TestRenewalWorker_NewCycleIsBilledAgain(t *testing.T) {
	repo := newMemRepo()
	notif := &recRenewalNotifier{}
	userID := uuid.New()
	subID := uuid.New()
	periodEnd := time.Now().UTC().Add(-time.Hour)
	src := &fakeRenewalSource{due: []RenewalCandidate{{
		UserID: userID, SubscriptionID: subID, PlanCode: "pro", PlanName: "Pro",
		PriceIDR: 39000, PeriodEnd: periodEnd,
	}}}
	w := newRenewalWorker(repo, src, notif)
	w.RunOnce(context.Background())

	// Next cycle: the previous invoice was paid, so no open invoice blocks it.
	for _, inv := range repo.invoices {
		inv.Status = InvoicePaid
	}

	src.due = []RenewalCandidate{{
		UserID: userID, SubscriptionID: subID, PlanCode: "pro", PlanName: "Pro",
		PriceIDR: 39000, PeriodEnd: periodEnd.AddDate(0, 0, 30),
	}}
	w.RunOnce(context.Background())
	if notif.count() != 2 {
		t.Fatalf("a new cycle must be billed again, got %d invoices", notif.count())
	}
}

// TestRenewalInvoice_PlanCodeIsNotTheMarker is the regression guard for the
// activation bug: the renewal worker writes an idempotency marker into `notes`,
// and activation used to read the plan code FROM `notes`. A paid renewal invoice
// then tried to activate a plan named "renewal:<id>:<ts>" and failed, so the user
// paid and got nothing. The plan must live in the dedicated `plan_code` column.
func TestRenewalInvoice_PlanCodeIsNotTheMarker(t *testing.T) {
	repo := newMemRepo()
	notif := &recRenewalNotifier{}
	subID := uuid.New()
	src := &fakeRenewalSource{due: []RenewalCandidate{{
		UserID: uuid.New(), SubscriptionID: subID, PlanCode: "pro", PlanName: "Pro",
		PriceIDR: 39000, PeriodEnd: time.Now().UTC().Add(-time.Hour),
	}}}
	w := newRenewalWorker(repo, src, notif)
	w.RunOnce(context.Background())

	var inv *Invoice
	for _, i := range repo.invoices {
		inv = i
	}
	if inv == nil {
		t.Fatal("no invoice issued")
	}
	if inv.PlanCode == nil || *inv.PlanCode != "pro" {
		t.Fatalf("invoice plan_code = %v, want \"pro\"", inv.PlanCode)
	}
	if inv.Notes == nil || (*inv.Notes)[:8] != "renewal:" {
		t.Fatalf("invoice notes must hold the renewal marker, got %v", inv.Notes)
	}
	// The activation path must resolve the real plan, not the marker.
	if got := planCodeFromInvoice(inv); got != "pro" {
		t.Fatalf("planCodeFromInvoice = %q, want \"pro\"", got)
	}
}

type fakeCardCharger struct {
	status PaymentStatus
	err    error
}

func (f *fakeCardCharger) ChargeSavedCard(_ context.Context, _ ChargeSavedCardRequest) (*ChargeCardResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &ChargeCardResponse{
		GatewayRef:    "test-ref",
		Status:        f.status,
		PaymentMethod: "credit_card",
		MaskedCard:    "481111-1114",
		Bank:          "bca",
	}, nil
}

func TestRenewalWorker_AutoChargeWithSavedCard(t *testing.T) {
	repo := newMemRepo()
	notif := &recRenewalNotifier{}
	userID := uuid.New()
	subID := uuid.New()

	// User has a saved card.
	card := &CardToken{
		ID:         uuid.New(),
		UserID:     userID,
		Gateway:    "midtrans",
		TokenID:    "tok_12345",
		MaskedCard: "481111-1114",
		Bank:       "bca",
		IsDefault:  true,
	}
	repo.savedCards = append(repo.savedCards, card)

	src := &fakeRenewalSource{due: []RenewalCandidate{{
		UserID: userID, SubscriptionID: subID, PlanCode: "pro", PlanName: "Pro",
		PriceIDR: 50000, PeriodEnd: time.Now().UTC().Add(-time.Hour),
	}}}

	act := &countingActivator{}
	charger := &fakeCardCharger{status: PaymentSettled}
	autocharger := NewAutoCharger(repo, charger, act, noopTx{})

	invoices := NewInvoiceService(repo, nil, nil)
	w := NewRenewalWorker(src, repo, invoices, notif, RenewalConfig{}).WithAutoCharger(autocharger)
	w.RunOnce(context.Background())

	// Should have charged automatically, activating subscription and emitting autoCharged event
	if notif.autoChargedCount() != 1 {
		t.Fatalf("expected 1 auto-charge notification, got %d", notif.autoChargedCount())
	}
	if notif.count() != 0 {
		t.Fatalf("expected 0 manual invoice notifications, got %d", notif.count())
	}
	if act.count != 1 {
		t.Fatalf("subscription should be activated automatically, got %d", act.count)
	}

	// Verify invoice is paid
	var inv *Invoice
	for _, i := range repo.invoices {
		inv = i
	}
	if inv == nil || inv.Status != InvoicePaid {
		t.Fatalf("invoice should be marked paid, got %v", inv)
	}
}
