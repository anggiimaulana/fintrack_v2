package payment

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// memRepo is an in-memory Repository for webhook tests.
type memRepo struct {
	mu       sync.Mutex
	invoices map[uuid.UUID]*Invoice
	payments map[uuid.UUID]*Payment
	byRef    map[string]*Payment
	paidCall int
}

func newMemRepo() *memRepo {
	return &memRepo{invoices: map[uuid.UUID]*Invoice{}, payments: map[uuid.UUID]*Payment{}, byRef: map[string]*Payment{}}
}

func (m *memRepo) CreateInvoice(_ context.Context, inv *Invoice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invoices[inv.ID] = inv
	return nil
}
func (m *memRepo) GetInvoiceByID(_ context.Context, id uuid.UUID) (*Invoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.invoices[id], nil
}
func (m *memRepo) GetInvoiceByNumber(context.Context, string) (*Invoice, error) { return nil, nil }
func (m *memRepo) UpdateInvoiceStatus(context.Context, uuid.UUID, InvoiceStatus, *string, *string) error {
	return nil
}
func (m *memRepo) MarkInvoicePaid(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paidCall++
	if inv := m.invoices[id]; inv != nil {
		inv.Status = InvoicePaid
	}
	return nil
}
func (m *memRepo) NextInvoiceNumber(context.Context, string) (string, error) {
	return "INV/TEST/00001", nil
}

func (m *memRepo) CreatePayment(_ context.Context, p *Payment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payments[p.ID] = p
	if p.GatewayRef != nil {
		m.byRef[*p.GatewayRef] = p
	}
	return nil
}
func (m *memRepo) GetPaymentByID(_ context.Context, id uuid.UUID) (*Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.payments[id], nil
}
func (m *memRepo) GetPaymentByGatewayRef(_ context.Context, ref string) (*Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byRef[ref], nil
}
func (m *memRepo) GetPaymentByGatewayRefForUpdate(_ context.Context, ref string) (*Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byRef[ref], nil
}
func (m *memRepo) UpdatePaymentStatus(_ context.Context, id uuid.UUID, status PaymentStatus, method, proofURL *string, _ map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p := m.payments[id]; p != nil {
		p.Status = status
		if method != nil {
			p.PaymentMethod = method
		}
	}
	return nil
}
func (m *memRepo) SetPaymentProof(context.Context, uuid.UUID, string) error { return nil }
func (m *memRepo) ListPaymentsPendingReview(context.Context, int) ([]*Payment, error) {
	return nil, nil
}
func (m *memRepo) ListInvoicesDueBefore(context.Context, int64) ([]*Invoice, error) {
	return nil, nil
}

// noopTx runs fn directly (no real transaction in unit tests).
type noopTx struct{}

func (noopTx) RunInTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// fakeGateway returns a scripted webhook result and status.
type fakeGateway struct {
	name       string
	verifyErr  error
	webhookRes *WebhookResult
	statusRes  *StatusResult
}

func (f *fakeGateway) Name() string { return f.name }
func (f *fakeGateway) CreateTransaction(context.Context, CreateTxRequest) (*CreateTxResponse, error) {
	return &CreateTxResponse{Status: PaymentPending}, nil
}
func (f *fakeGateway) VerifyWebhook(context.Context, []byte, map[string]string) (*WebhookResult, error) {
	if f.verifyErr != nil {
		return nil, f.verifyErr
	}
	return f.webhookRes, nil
}
func (f *fakeGateway) CheckStatus(context.Context, string) (*StatusResult, error) {
	return f.statusRes, nil
}
func (f *fakeGateway) CancelTransaction(context.Context, string) error { return nil }

// countingActivator counts activations to prove idempotency.
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

// TestWebhookProcessor_IdempotentReplay is the core anti-double-credit proof:
// processing the same settled webhook N times activates the subscription and
// marks the invoice paid exactly once.
func TestWebhookProcessor_IdempotentReplay(t *testing.T) {
	repo := newMemRepo()
	reg := NewRegistry()
	ref := "INV-20261011-00001"
	reg.Register(&fakeGateway{
		name:       "midtrans",
		webhookRes: &WebhookResult{OrderID: ref, GatewayRef: ref, Status: PaymentSettled, AmountIDR: 39000, PaymentMethod: "qris"},
		statusRes:  &StatusResult{OrderID: ref, Status: PaymentSettled, AmountIDR: 39000},
	})
	act := &countingActivator{}
	proc := NewWebhookProcessor(repo, reg, act, noopTx{})

	// Seed invoice + payment.
	invID := uuid.New()
	userID := uuid.New()
	note := "pro"
	_ = repo.CreateInvoice(context.Background(), &Invoice{ID: invID, UserID: userID, Status: InvoiceIssued, Notes: &note})
	refCopy := ref
	pay := &Payment{ID: uuid.New(), InvoiceID: invID, UserID: userID, Gateway: "midtrans", GatewayRef: &refCopy, AmountIDR: 39000, Status: PaymentPending, IdempotencyKey: "k1"}
	_ = repo.CreatePayment(context.Background(), pay)

	// Process the same webhook 5 times.
	for i := range 5 {
		res, err := proc.Process(context.Background(), "midtrans", []byte("{}"), nil)
		if err != nil {
			t.Fatalf("process %d: %v", i, err)
		}
		if i == 0 && res.AlreadyProcessed {
			t.Fatal("first process should not be marked already-processed")
		}
		if i > 0 && !res.AlreadyProcessed {
			t.Fatalf("process %d should be idempotent (already processed)", i)
		}
	}

	if act.count != 1 {
		t.Fatalf("subscription must be activated exactly once, got %d", act.count)
	}
	if repo.paidCall != 1 {
		t.Fatalf("invoice must be marked paid exactly once, got %d", repo.paidCall)
	}
	if repo.payments[pay.ID].Status != PaymentSettled {
		t.Fatalf("payment should be settled, got %s", repo.payments[pay.ID].Status)
	}
}

// TestWebhookProcessor_InvalidSignature verifies signature errors propagate.
func TestWebhookProcessor_InvalidSignature(t *testing.T) {
	repo := newMemRepo()
	reg := NewRegistry()
	reg.Register(&fakeGateway{name: "midtrans", verifyErr: ErrInvalidSignature})
	proc := NewWebhookProcessor(repo, reg, nil, noopTx{})

	if _, err := proc.Process(context.Background(), "midtrans", []byte("{}"), nil); err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

// TestWebhookProcessor_UnknownGateway verifies missing gateway handling.
func TestWebhookProcessor_UnknownGateway(t *testing.T) {
	proc := NewWebhookProcessor(newMemRepo(), NewRegistry(), nil, noopTx{})
	if _, err := proc.Process(context.Background(), "nope", []byte("{}"), nil); err != ErrGatewayNotConfigured {
		t.Fatalf("expected ErrGatewayNotConfigured, got %v", err)
	}
}
