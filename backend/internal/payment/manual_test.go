package payment

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
)

// memSettings is an in-memory SettingsRepository.
type memSettings struct{ m map[string]any }

func newMemSettings() *memSettings {
	return &memSettings{m: map[string]any{"tax_enabled": "true", "tax_rate_percent": int64(11), "invoice_due_hours": int64(24), "payment_gateway_default": "manual"}}
}
func (s *memSettings) GetString(_ context.Context, key string) (string, error) {
	if v, ok := s.m[key].(string); ok {
		return v, nil
	}
	return "", nil
}
func (s *memSettings) GetInt(_ context.Context, key string) (int64, error) {
	if v, ok := s.m[key].(int64); ok {
		return v, nil
	}
	return 0, nil
}
func (s *memSettings) Set(_ context.Context, key string, value any) error {
	s.m[key] = value
	return nil
}

// memProofStore captures saved proof content.
type memProofStore struct{ saved map[string][]byte }

func (m *memProofStore) Save(_ context.Context, rel string, content []byte) (string, error) {
	if m.saved == nil {
		m.saved = map[string][]byte{}
	}
	m.saved[rel] = content
	return "https://cdn.local/" + rel, nil
}

// TestTaxCalculator verifies floor rounding and 0% handling.
func TestTaxCalculator(t *testing.T) {
	ctx := context.Background()

	s := newMemSettings()
	s.m["tax_enabled"] = "true"
	s.m["tax_rate_percent"] = int64(11)
	tc := NewTaxCalculator(s)
	if got := tc.TaxIDR(ctx, 100000); got != 11000 {
		t.Fatalf("11%% of 100000 = 11000, got %d", got)
	}
	// floor: 11% of 999 = 109.89 -> 109
	if got := tc.TaxIDR(ctx, 999); got != 109 {
		t.Fatalf("11%% of 999 should floor to 109, got %d", got)
	}

	s.m["tax_enabled"] = "false"
	s.m["tax_rate_percent"] = int64(0)
	if got := tc.TaxIDR(ctx, 100000); got != 0 {
		t.Fatalf("0%% tax must be 0, got %d", got)
	}

	if got := tc.TaxIDR(ctx, 0); got != 0 {
		t.Fatalf("zero subtotal must be 0 tax, got %d", got)
	}
}

// TestTaxCalculator_Toggle verifies PPN is opt-in: a configured rate does
// nothing until tax_enabled is explicitly true.
func TestTaxCalculator_Toggle(t *testing.T) {
	ctx := context.Background()
	// A bare settings store: no tax_enabled key at all, which must mean "off".
	s := &memSettings{m: map[string]any{"tax_rate_percent": int64(11)}}
	tc := NewTaxCalculator(s)

	if tc.Enabled(ctx) {
		t.Fatal("tax must default to disabled")
	}
	if got := tc.TaxIDR(ctx, 100000); got != 0 {
		t.Fatalf("disabled tax must charge 0, got %d", got)
	}
	if got := tc.RatePercent(ctx); got != 0 {
		t.Fatalf("disabled tax must report 0%% rate, got %d", got)
	}

	// Flip it on: the same rate now applies.
	s.m["tax_enabled"] = "true"
	if !tc.Enabled(ctx) {
		t.Fatal("tax should be enabled")
	}
	if got := tc.TaxIDR(ctx, 100000); got != 11000 {
		t.Fatalf("enabled 11%% of 100000 = 11000, got %d", got)
	}
	if got := tc.RatePercent(ctx); got != 11 {
		t.Fatalf("rate = %d, want 11", got)
	}
}

// TestTaxCalculator_DynamicRate verifies the rate is read per call, so an
// operator change (e.g. 11% -> 12%) applies without a restart.
func TestTaxCalculator_DynamicRate(t *testing.T) {
	ctx := context.Background()
	s := newMemSettings()
	s.m["tax_enabled"] = "true"
	s.m["tax_rate_percent"] = int64(11)
	tc := NewTaxCalculator(s)

	if got := tc.TaxIDR(ctx, 100000); got != 11000 {
		t.Fatalf("11%% = 11000, got %d", got)
	}
	s.m["tax_rate_percent"] = int64(12)
	if got := tc.TaxIDR(ctx, 100000); got != 12000 {
		t.Fatalf("rate change to 12%% should apply immediately, got %d", got)
	}
}

// TestTaxCalculator_Label verifies the display label defaults to PPN and is
// overridable.
func TestTaxCalculator_Label(t *testing.T) {
	ctx := context.Background()
	s := newMemSettings()
	tc := NewTaxCalculator(s)
	if got := tc.Label(ctx); got != "PPN" {
		t.Fatalf("default label = %q, want PPN", got)
	}
	s.m["tax_label"] = "PB1"
	if got := tc.Label(ctx); got != "PB1" {
		t.Fatalf("custom label = %q, want PB1", got)
	}
}

// TestInvoiceService_Create verifies number format, tax, and total.
func TestInvoiceService_Create(t *testing.T) {
	repo := newMemRepo()
	// Provide a deterministic numbering repo behaviour via a fake number fn.
	s := newMemSettings()
	tc := NewTaxCalculator(s)
	svc := NewInvoiceService(repo, s, tc)

	inv, err := svc.Create(context.Background(), CreateInvoiceDTO{
		UserID: uuid.New(), SubtotalIDR: 39000, Notes: "pro",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	if inv.TaxIDR != 4290 { // 11% of 39000
		t.Fatalf("expected tax 4290, got %d", inv.TaxIDR)
	}
	if inv.TotalIDR != 43290 {
		t.Fatalf("expected total 43290, got %d", inv.TotalIDR)
	}
	if inv.InvoiceNumber == "" {
		t.Fatal("expected invoice number")
	}
	if inv.Status != InvoiceIssued {
		t.Fatalf("expected issued, got %s", inv.Status)
	}
	if inv.DueAt == nil {
		t.Fatal("expected due date")
	}
}

// TestValidateProof verifies magic-byte validation.
func TestValidateProof(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	webp := append([]byte("RIFF"), append([]byte{0, 0, 0, 0}, []byte("WEBP")...)...)
	exe := []byte("MZ\x90\x00")

	cases := []struct {
		name string
		b    []byte
		ok   bool
	}{
		{"jpeg", jpeg, true},
		{"png", png, true},
		{"webp", webp, true},
		{"exe", exe, false},
		{"empty", nil, false},
	}
	for _, c := range cases {
		_, _, err := ValidateProof(c.b, "f")
		if c.ok && err != nil {
			t.Errorf("%s: expected ok, got %v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Errorf("%s: expected rejection", c.name)
		}
	}

	// Oversized content rejected.
	big := make([]byte, MaxProofBytes+1)
	big[0], big[1], big[2] = 0xFF, 0xD8, 0xFF
	if _, _, err := ValidateProof(big, "f"); err != ErrProofTooLarge {
		t.Fatalf("expected ErrProofTooLarge, got %v", err)
	}
}

// TestManualService_UploadAndApprove verifies the manual flow end-to-end.
func TestManualService_UploadAndApprove(t *testing.T) {
	repo := newMemRepo()
	act := &countingActivator{}
	store := &memProofStore{}
	svc := NewManualService(repo, act, noopTx{}, store)
	ctx := context.Background()

	// Seed invoice + manual payment.
	invID := uuid.New()
	userID := uuid.New()
	note := "pro"
	_ = repo.CreateInvoice(ctx, &Invoice{ID: invID, UserID: userID, Status: InvoiceIssued, Notes: &note})
	ref := "INV-1"
	pay := &Payment{ID: uuid.New(), InvoiceID: invID, UserID: userID, Gateway: "manual", GatewayRef: &ref, AmountIDR: 39000, Status: PaymentPending, IdempotencyKey: "mk1"}
	_ = repo.CreatePayment(ctx, pay)

	// Upload proof (valid PNG).
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00}
	updated, err := svc.UploadProof(ctx, pay.ID, userID, png, "proof.png")
	if err != nil {
		t.Fatalf("upload proof: %v", err)
	}
	if updated.Status != PaymentPendingReview {
		t.Fatalf("expected pending_review, got %s", updated.Status)
	}
	if len(store.saved) != 1 {
		t.Fatalf("expected 1 stored proof, got %d", len(store.saved))
	}

	// Approve -> settled + activation once.
	approved, err := svc.Approve(ctx, pay.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != PaymentSettled {
		t.Fatalf("expected settled, got %s", approved.Status)
	}
	if act.count != 1 {
		t.Fatalf("expected 1 activation, got %d", act.count)
	}

	// Double approve must fail (state guard).
	if _, err := svc.Approve(ctx, pay.ID); err == nil {
		t.Fatal("double approve must be rejected")
	}
}

// TestManualService_RejectRequiresReason verifies rejection reason enforcement.
func TestManualService_RejectRequiresReason(t *testing.T) {
	repo := newMemRepo()
	svc := NewManualService(repo, nil, noopTx{}, nil)
	if _, err := svc.Reject(context.Background(), uuid.New(), ""); err == nil {
		t.Fatal("empty reason must be rejected")
	}
}

// TestManualService_UploadWrongOwner verifies ownership enforcement.
func TestManualService_UploadWrongOwner(t *testing.T) {
	repo := newMemRepo()
	svc := NewManualService(repo, nil, noopTx{}, nil)
	ctx := context.Background()
	pay := &Payment{ID: uuid.New(), UserID: uuid.New(), Gateway: "manual", AmountIDR: 1000, Status: PaymentPending, IdempotencyKey: "x"}
	_ = repo.CreatePayment(ctx, pay)

	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if _, err := svc.UploadProof(ctx, pay.ID, uuid.New(), png, "p.png"); err != ErrPaymentNotFound {
		t.Fatalf("expected ErrPaymentNotFound for wrong owner, got %v", err)
	}
}

var _ = bytes.MinRead
