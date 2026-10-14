package payment

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// fakeAccountSource is an in-memory PaymentAccountSource.
type fakeAccountSource struct {
	accounts []ManualAccount
	err      error
}

func (f *fakeAccountSource) ListActive(context.Context) ([]ManualAccount, error) {
	return f.accounts, f.err
}

func TestPaymentInstructions_IncludesAllActiveAccounts(t *testing.T) {
	repo := newMemRepo()
	userID := uuid.New()
	payID := uuid.New()
	pay := &Payment{
		ID:              payID,
		UserID:          userID,
		AmountIDR:       43290,
		Status:          PaymentPending,
		PayloadSnapshot: map[string]any{"instructions": map[string]any{"amount_idr": float64(43290), "order_id": "INV/1"}},
	}
	if err := repo.CreatePayment(context.Background(), pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	svc := NewUserService(repo, newMemSettings(), nil, nil).WithPaymentAccounts(&fakeAccountSource{
		accounts: []ManualAccount{
			{Kind: "bank", Name: "BCA", AccountNumber: "123", AccountHolder: "PT FinTrack"},
			{Kind: "ewallet", Name: "DANA", AccountNumber: "0812", AccountHolder: "FinTrack"},
			{Kind: "qris", Name: "QRIS", QRISURL: "https://x/q.png"},
		},
	})

	ins, err := svc.PaymentInstructions(context.Background(), userID, payID)
	if err != nil {
		t.Fatalf("instructions: %v", err)
	}
	accounts, ok := ins["accounts"].([]ManualAccount)
	if !ok {
		t.Fatalf("expected accounts in instructions, got %#v", ins)
	}
	if len(accounts) != 3 {
		t.Fatalf("expected 3 accounts, got %d", len(accounts))
	}
	if accounts[0].Name != "BCA" || accounts[1].Name != "DANA" || accounts[2].Kind != "qris" {
		t.Fatalf("unexpected accounts: %+v", accounts)
	}
	// The amount and order id survive the overlay.
	if ins["amount_idr"] == nil || ins["order_id"] == nil {
		t.Fatalf("expected amount/order preserved, got %#v", ins)
	}
}

func TestPaymentInstructions_FallsBackToSnapshotWhenNoAccounts(t *testing.T) {
	repo := newMemRepo()
	userID := uuid.New()
	payID := uuid.New()
	pay := &Payment{
		ID:              payID,
		UserID:          userID,
		AmountIDR:       1000,
		Status:          PaymentPending,
		PayloadSnapshot: map[string]any{"instructions": map[string]any{"amount_idr": float64(1000), "order_id": "INV/9"}},
	}
	if err := repo.CreatePayment(context.Background(), pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	// No account source at all (nil) and no registry: must not panic, and must
	// still return the amount/order derived from the row.
	svc := NewUserService(repo, newMemSettings(), nil, nil)
	ins, err := svc.PaymentInstructions(context.Background(), userID, payID)
	if err != nil {
		t.Fatalf("instructions: %v", err)
	}
	if ins["order_id"] != "INV/9" {
		t.Fatalf("expected order id from snapshot, got %#v", ins)
	}
	if _, hasAccounts := ins["accounts"]; hasAccounts {
		t.Fatal("did not expect accounts key without a source")
	}
}

func TestPaymentInstructions_UnknownPayment(t *testing.T) {
	svc := NewUserService(newMemRepo(), newMemSettings(), nil, nil)
	_, err := svc.PaymentInstructions(context.Background(), uuid.New(), uuid.New())
	if err != ErrPaymentNotFound {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}
