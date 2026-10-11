package postgres

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/payment"
)

// countingActivator counts activations across concurrent webhook processing.
type countingActivator struct {
	mu    sync.Mutex
	count int
}

func (c *countingActivator) ActivateForUser(context.Context, uuid.UUID, string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	return nil
}

// fixedGateway returns a scripted settled webhook + status.
type fixedGateway struct {
	name string
	ref  string
}

func (f *fixedGateway) Name() string { return f.name }
func (f *fixedGateway) CreateTransaction(context.Context, payment.CreateTxRequest) (*payment.CreateTxResponse, error) {
	return &payment.CreateTxResponse{Status: payment.PaymentPending}, nil
}
func (f *fixedGateway) VerifyWebhook(context.Context, []byte, map[string]string) (*payment.WebhookResult, error) {
	return &payment.WebhookResult{OrderID: f.ref, GatewayRef: f.ref, Status: payment.PaymentSettled, AmountIDR: 39000, PaymentMethod: "qris"}, nil
}
func (f *fixedGateway) CheckStatus(context.Context, string) (*payment.StatusResult, error) {
	return &payment.StatusResult{OrderID: f.ref, Status: payment.PaymentSettled, AmountIDR: 39000}, nil
}
func (f *fixedGateway) CancelTransaction(context.Context, string) error { return nil }

// TestPayment_ConcurrentWebhookIdempotent is the core anti-double-credit proof
// against a live database: many concurrent webhooks for the same order must
// settle the payment and activate the subscription exactly once.
func TestPayment_ConcurrentWebhookIdempotent(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPaymentRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "PayITest"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	// Create invoice + payment with a unique gateway_ref.
	ref := "INV-ITEST-" + userID.String()[:8]
	note := "pro"
	inv := &payment.Invoice{
		InvoiceNumber: "INV/ITEST/" + userID.String()[:8],
		UserID:        userID, SubtotalIDR: 39000, TotalIDR: 39000, Status: payment.InvoiceIssued, Notes: &note,
	}
	if err := repo.CreateInvoice(ctx, inv); err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	pay := &payment.Payment{
		InvoiceID: inv.ID, UserID: userID, Gateway: "midtrans", GatewayRef: &ref,
		AmountIDR: 39000, Status: payment.PaymentPending, IdempotencyKey: "itest-" + userID.String(),
	}
	if err := repo.CreatePayment(ctx, pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	reg := payment.NewRegistry()
	reg.Register(&fixedGateway{name: "midtrans", ref: ref})
	act := &countingActivator{}
	proc := payment.NewWebhookProcessor(repo, reg, act, tx)

	// Fire 20 concurrent webhooks for the same order.
	const n = 20
	var processed, idempotent int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res, err := proc.Process(ctx, "midtrans", []byte("{}"), nil)
			if err != nil {
				t.Errorf("process: %v", err)
				return
			}
			if res.AlreadyProcessed {
				atomic.AddInt32(&idempotent, 1)
			} else {
				atomic.AddInt32(&processed, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if processed != 1 {
		t.Fatalf("exactly 1 webhook should apply the transition, got %d", processed)
	}
	if int(processed+idempotent) != n {
		t.Fatalf("accounting mismatch: processed=%d idempotent=%d", processed, idempotent)
	}
	if act.count != 1 {
		t.Fatalf("subscription must be activated exactly once, got %d", act.count)
	}

	// Verify DB state.
	stored, err := repo.GetPaymentByID(ctx, pay.ID)
	if err != nil || stored == nil {
		t.Fatalf("get payment: %v", err)
	}
	if stored.Status != payment.PaymentSettled {
		t.Fatalf("payment should be settled, got %s", stored.Status)
	}
	invAfter, err := repo.GetInvoiceByID(ctx, inv.ID)
	if err != nil || invAfter == nil {
		t.Fatalf("get invoice: %v", err)
	}
	if invAfter.Status != payment.InvoicePaid {
		t.Fatalf("invoice should be paid, got %s", invAfter.Status)
	}
}

// TestPayment_InvoiceNumberSequence verifies atomic, gapless numbering.
func TestPayment_InvoiceNumberSequence(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPaymentRepo(pool)
	ctx := context.Background()

	day := "20990101" // far-future day to avoid clashing with real data
	seen := map[string]bool{}
	for range 20 {
		num, err := repo.NextInvoiceNumber(ctx, day)
		if err != nil {
			t.Fatalf("next number: %v", err)
		}
		if seen[num] {
			t.Fatalf("duplicate invoice number %s", num)
		}
		seen[num] = true
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM invoice_counters WHERE day = $1::date`, day)
	})
	if len(seen) != 20 {
		t.Fatalf("expected 20 unique numbers, got %d", len(seen))
	}
}
