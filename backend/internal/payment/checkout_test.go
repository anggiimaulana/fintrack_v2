package payment

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestCheckoutService_ManualFlow verifies invoice + payment creation and the
// gateway lock, for the manual gateway.
func TestCheckoutService_ManualFlow(t *testing.T) {
	repo := newMemRepo()
	s := newMemSettings()
	s.m["tax_enabled"] = "false"
	s.m["tax_rate_percent"] = int64(0)
	reg := NewRegistry()
	reg.Register(NewManualGateway("BCA", "1234567890", "PT FinTrack"))

	invoices := NewInvoiceService(repo, s, NewTaxCalculator(s))
	svc := NewCheckoutService(repo, s, reg, invoices, nil)

	res, err := svc.Checkout(context.Background(), CheckoutRequest{
		UserID: uuid.New(), PlanCode: "pro", PlanName: "Pro", AmountIDR: 39000,
	})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if res.Gateway != "manual" {
		t.Fatalf("expected manual, got %s", res.Gateway)
	}
	if res.Invoice.TotalIDR != 39000 {
		t.Fatalf("expected total 39000, got %d", res.Invoice.TotalIDR)
	}
	if res.Payment.Status != PaymentPending {
		t.Fatalf("expected pending, got %s", res.Payment.Status)
	}
	// Manual gateway should return transfer instructions.
	if res.Instructions == nil {
		t.Fatal("expected transfer instructions for manual gateway")
	}
	if res.Instructions["account_number"] != "1234567890" {
		t.Fatalf("unexpected instructions: %v", res.Instructions)
	}
	// Payment row must lock the gateway name.
	stored, _ := repo.GetPaymentByID(context.Background(), res.Payment.ID)
	if stored.Gateway != "manual" {
		t.Fatalf("payment must lock gateway 'manual', got %s", stored.Gateway)
	}
}

// TestCheckoutService_TaxApplied verifies tax is included in the total.
func TestCheckoutService_TaxApplied(t *testing.T) {
	repo := newMemRepo()
	s := newMemSettings()
	s.m["tax_enabled"] = "true"
	s.m["tax_rate_percent"] = int64(11)
	reg := NewRegistry()
	reg.Register(NewManualGateway("BCA", "1", "X"))

	invoices := NewInvoiceService(repo, s, NewTaxCalculator(s))
	svc := NewCheckoutService(repo, s, reg, invoices, nil)

	res, err := svc.Checkout(context.Background(), CheckoutRequest{
		UserID: uuid.New(), PlanCode: "pro", PlanName: "Pro", AmountIDR: 100000,
	})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if res.Invoice.TaxIDR != 11000 {
		t.Fatalf("expected tax 11000, got %d", res.Invoice.TaxIDR)
	}
	if res.Payment.AmountIDR != 111000 {
		t.Fatalf("payment amount must include tax (111000), got %d", res.Payment.AmountIDR)
	}
}
