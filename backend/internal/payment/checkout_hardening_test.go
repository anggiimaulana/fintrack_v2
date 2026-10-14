package payment

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// newCheckoutForTest builds a checkout service with a manual gateway and no tax.
func newCheckoutForTest(t *testing.T) (*CheckoutService, *memRepo, SettingsRepository) {
	t.Helper()
	repo := newMemRepo()
	s := newMemSettings()
	s.m["tax_enabled"] = "false"
	s.m["tax_rate_percent"] = int64(0)
	reg := NewRegistry()
	reg.Register(NewManualGateway("BCA", "123", "PT F"))
	invoices := NewInvoiceService(repo, s, NewTaxCalculator(s))
	return NewCheckoutService(repo, s, reg, invoices, nil), repo, s
}

// TestCheckout_BlocksSecondOpenInvoice verifies a user with an unpaid invoice
// cannot create another — the core "no invoice spam" rule.
func TestCheckout_BlocksSecondOpenInvoice(t *testing.T) {
	svc, _, _ := newCheckoutForTest(t)
	user := uuid.New()

	if _, err := svc.Checkout(context.Background(), CheckoutRequest{
		UserID: user, PlanCode: "pro", PlanName: "Pro", AmountIDR: 39000,
	}); err != nil {
		t.Fatalf("first checkout: %v", err)
	}

	_, err := svc.Checkout(context.Background(), CheckoutRequest{
		UserID: user, PlanCode: "pro", PlanName: "Pro", AmountIDR: 39000,
	})
	if !errors.Is(err, ErrOpenInvoiceExists) {
		t.Fatalf("expected ErrOpenInvoiceExists, got %v", err)
	}
}

// TestCheckout_AllowsAfterCancel verifies cancelling the open invoice frees the
// slot for a new checkout.
func TestCheckout_AllowsAfterCancel(t *testing.T) {
	svc, repo, _ := newCheckoutForTest(t)
	user := uuid.New()

	first, err := svc.Checkout(context.Background(), CheckoutRequest{
		UserID: user, PlanCode: "pro", PlanName: "Pro", AmountIDR: 39000,
	})
	if err != nil {
		t.Fatalf("first checkout: %v", err)
	}

	// Cancel it (as the user-facing service would).
	changed, err := repo.CloseInvoice(context.Background(), first.Invoice.ID, InvoiceVoided, "cancelled_by_user")
	if err != nil || !changed {
		t.Fatalf("close invoice: changed=%v err=%v", changed, err)
	}

	if _, err := svc.Checkout(context.Background(), CheckoutRequest{
		UserID: user, PlanCode: "pro", PlanName: "Pro", AmountIDR: 39000,
	}); err != nil {
		t.Fatalf("checkout after cancel should succeed, got %v", err)
	}
}

// TestCheckout_DifferentUsersIndependent verifies the guard is per-user.
func TestCheckout_DifferentUsersIndependent(t *testing.T) {
	svc, _, _ := newCheckoutForTest(t)
	if _, err := svc.Checkout(context.Background(), CheckoutRequest{UserID: uuid.New(), PlanCode: "pro", AmountIDR: 39000}); err != nil {
		t.Fatalf("user A: %v", err)
	}
	if _, err := svc.Checkout(context.Background(), CheckoutRequest{UserID: uuid.New(), PlanCode: "pro", AmountIDR: 39000}); err != nil {
		t.Fatalf("user B should not be blocked by user A: %v", err)
	}
}

// TestUserService_CancelInvoice verifies the owner-scoped cancel path and that a
// non-owner cannot cancel someone else's invoice.
func TestUserService_CancelInvoice(t *testing.T) {
	repo := newMemRepo()
	owner := uuid.New()
	other := uuid.New()
	inv := &Invoice{
		ID:            uuid.New(),
		InvoiceNumber: "INV/T/1",
		UserID:        owner,
		SubtotalIDR:   1000,
		TotalIDR:      1000,
		Status:        InvoiceIssued,
	}
	if err := repo.CreateInvoice(context.Background(), inv); err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	svc := NewUserService(repo, newMemSettings(), nil, nil)

	// A non-owner gets not-found (no existence leak).
	if _, err := svc.CancelInvoice(context.Background(), other, inv.ID); !errors.Is(err, ErrInvoiceNotFound) {
		t.Fatalf("non-owner cancel should be ErrInvoiceNotFound, got %v", err)
	}

	// The owner cancels successfully and the invoice is voided with a reason.
	closed, err := svc.CancelInvoice(context.Background(), owner, inv.ID)
	if err != nil {
		t.Fatalf("owner cancel: %v", err)
	}
	if closed.Status != InvoiceVoided {
		t.Fatalf("status = %s, want voided", closed.Status)
	}
	if closed.ClosedReason == nil || *closed.ClosedReason != "cancelled_by_user" {
		t.Fatalf("closed reason = %v", closed.ClosedReason)
	}

	// A second cancel is rejected (already closed).
	if _, err := svc.CancelInvoice(context.Background(), owner, inv.ID); !errors.Is(err, ErrInvoiceNotCancellable) {
		t.Fatalf("second cancel should be ErrInvoiceNotCancellable, got %v", err)
	}
}

// TestUserService_CancelPaidInvoiceRejected verifies a paid invoice is terminal.
func TestUserService_CancelPaidInvoiceRejected(t *testing.T) {
	repo := newMemRepo()
	user := uuid.New()
	inv := &Invoice{ID: uuid.New(), InvoiceNumber: "INV/T/2", UserID: user, SubtotalIDR: 1, TotalIDR: 1, Status: InvoicePaid}
	if err := repo.CreateInvoice(context.Background(), inv); err != nil {
		t.Fatalf("create: %v", err)
	}
	svc := NewUserService(repo, newMemSettings(), nil, nil)
	if _, err := svc.CancelInvoice(context.Background(), user, inv.ID); !errors.Is(err, ErrInvoiceNotCancellable) {
		t.Fatalf("paid invoice must not be cancellable, got %v", err)
	}
}

// TestExpiryWorker_ExpiresPastDueInvoice verifies the worker closes an overdue
// invoice and expires its pending payment.
func TestExpiryWorker_ExpiresPastDueInvoice(t *testing.T) {
	repo := newMemRepo()
	user := uuid.New()
	past := time.Now().Add(-1 * time.Hour)
	inv := &Invoice{
		ID:            uuid.New(),
		InvoiceNumber: "INV/T/3",
		UserID:        user,
		SubtotalIDR:   1000,
		TotalIDR:      1000,
		Status:        InvoiceIssued,
		DueAt:         &past,
	}
	if err := repo.CreateInvoice(context.Background(), inv); err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	pay := &Payment{ID: uuid.New(), InvoiceID: inv.ID, UserID: user, AmountIDR: 1000, Status: PaymentPending, IdempotencyKey: "k1"}
	if err := repo.CreatePayment(context.Background(), pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	worker := NewExpiryWorker(repo, time.Minute)
	worker.RunOnce(context.Background())

	stored, _ := repo.GetInvoiceByID(context.Background(), inv.ID)
	if stored.Status != InvoiceExpired {
		t.Fatalf("invoice status = %s, want expired", stored.Status)
	}
	if stored.ExpiredAt == nil {
		t.Fatal("expected expired_at to be set")
	}
	gotPay, _ := repo.GetPaymentByID(context.Background(), pay.ID)
	if gotPay.Status != PaymentExpired {
		t.Fatalf("payment status = %s, want expired", gotPay.Status)
	}
}

// TestExpiryWorker_LeavesFutureInvoiceOpen verifies a not-yet-due invoice is
// untouched.
func TestExpiryWorker_LeavesFutureInvoiceOpen(t *testing.T) {
	repo := newMemRepo()
	future := time.Now().Add(24 * time.Hour)
	inv := &Invoice{ID: uuid.New(), InvoiceNumber: "INV/T/4", UserID: uuid.New(), SubtotalIDR: 1, TotalIDR: 1, Status: InvoiceIssued, DueAt: &future}
	if err := repo.CreateInvoice(context.Background(), inv); err != nil {
		t.Fatalf("create: %v", err)
	}
	NewExpiryWorker(repo, time.Minute).RunOnce(context.Background())
	stored, _ := repo.GetInvoiceByID(context.Background(), inv.ID)
	if stored.Status != InvoiceIssued {
		t.Fatalf("future invoice must stay issued, got %s", stored.Status)
	}
}

// TestExpiryWorker_PaidInvoiceNotExpired verifies a settled invoice is never
// expired even if its due date passed (the conditional UPDATE protects this).
func TestExpiryWorker_PaidInvoiceNotExpired(t *testing.T) {
	repo := newMemRepo()
	past := time.Now().Add(-2 * time.Hour)
	inv := &Invoice{ID: uuid.New(), InvoiceNumber: "INV/T/5", UserID: uuid.New(), SubtotalIDR: 1, TotalIDR: 1, Status: InvoicePaid, DueAt: &past}
	if err := repo.CreateInvoice(context.Background(), inv); err != nil {
		t.Fatalf("create: %v", err)
	}
	NewExpiryWorker(repo, time.Minute).RunOnce(context.Background())
	stored, _ := repo.GetInvoiceByID(context.Background(), inv.ID)
	if stored.Status != InvoicePaid {
		t.Fatalf("paid invoice must not be expired, got %s", stored.Status)
	}
}

// TestCheckout_ConcurrentRace_Serialized reproduces the production race through
// the transactional path: the tx runner serialises the check-then-insert (like
// the real per-user advisory lock inside a DB transaction), so exactly one
// checkout wins and the rest get a clean conflict.
//
// The in-memory fake has no unique index, so the *serialisation* is what this
// asserts; the DB partial unique index is the additional production backstop,
// verified separately by the integration tests.
func TestCheckout_ConcurrentRace_Serialized(t *testing.T) {
	repo := newMemRepo()
	s := newMemSettings()
	s.m["tax_enabled"] = "false"
	reg := NewRegistry()
	reg.Register(NewManualGateway("BCA", "123", "PT F"))
	invoices := NewInvoiceService(repo, s, NewTaxCalculator(s))
	svc := NewCheckoutService(repo, s, reg, invoices, nil).WithTx(&serialTxRunner{})

	user := uuid.New()
	const attempts = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, blocked, other := 0, 0, 0
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Checkout(context.Background(), CheckoutRequest{UserID: user, PlanCode: "pro", AmountIDR: 39000})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case errors.Is(err, ErrOpenInvoiceExists):
				blocked++
			default:
				other++
			}
		}()
	}
	wg.Wait()

	if ok != 1 {
		t.Fatalf("expected exactly 1 success through the tx runner, got %d (blocked=%d other=%d)", ok, blocked, other)
	}
	if other != 0 {
		t.Fatalf("expected only clean conflicts, got %d unexpected errors", other)
	}
}

// serialTxRunner mimics the real TxManager: RunInTx holds a lock for the whole
// transaction, so the guard's check-then-insert is atomic exactly as it is under
// a DB transaction with a per-user advisory lock.
type serialTxRunner struct {
	mu sync.Mutex
}

func (r *serialTxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn(ctx)
}

// AdvisoryXactLock is a no-op: RunInTx already serialises the whole section, so
// the per-key lock has nothing left to do in this single-process fake.
func (r *serialTxRunner) AdvisoryXactLock(context.Context, string) error { return nil }
func TestInvoiceService_DueDateFromHoursAndMinutes(t *testing.T) {
	repo := newMemRepo()
	s := newMemSettings()
	s.m["invoice_expiry_hours"] = int64(1)
	s.m["invoice_expiry_minutes"] = int64(30)
	svc := NewInvoiceService(repo, s, NewTaxCalculator(s))

	before := time.Now()
	inv, err := svc.Create(context.Background(), CreateInvoiceDTO{UserID: uuid.New(), SubtotalIDR: 1000, Notes: "pro"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if inv.DueAt == nil {
		t.Fatal("expected a due date")
	}
	// The window is 1h30m = 90 minutes; allow a generous tolerance for the
	// business-timezone conversion.
	got := inv.DueAt.Sub(before)
	if got < 85*time.Minute || got > 95*time.Minute {
		t.Fatalf("due window = %v, want ~90m", got)
	}
}
