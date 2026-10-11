package payment

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMidtrans_CreateTransaction verifies Snap checkout creation + basic auth.
func TestMidtrans_CreateTransaction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transactions" {
			http.NotFound(w, r)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Errorf("expected basic auth, got %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		td := body["transaction_details"].(map[string]any)
		// gross_amount must be an integer (no float artifacts).
		if _, ok := td["gross_amount"].(float64); !ok {
			t.Errorf("gross_amount should be numeric")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":        "snap-token-123",
			"redirect_url": "https://app.sandbox.midtrans.com/snap/v2/vtweb/snap-token-123",
		})
	}))
	defer srv.Close()

	g := NewMidtransGateway(MidtransConfig{ServerKey: "SB-Mid-server-xxx", SnapBaseURL: srv.URL})
	resp, err := g.CreateTransaction(context.Background(), CreateTxRequest{
		OrderID: "INV-1", AmountIDR: 39000, Description: "Pro plan",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.SnapToken != "snap-token-123" {
		t.Fatalf("expected token, got %q", resp.SnapToken)
	}
	if resp.RedirectURL == "" {
		t.Fatal("expected redirect url")
	}
	if resp.Status != PaymentPending {
		t.Fatalf("expected pending, got %s", resp.Status)
	}
}

// TestMidtrans_VerifyWebhook_ValidSignature verifies correct SHA-512 accepted.
func TestMidtrans_VerifyWebhook_ValidSignature(t *testing.T) {
	serverKey := "SB-Mid-server-test"
	orderID := "INV-20261011-00001"
	statusCode := "200"
	gross := "39000.00"

	h := sha512.New()
	h.Write([]byte(orderID + statusCode + gross + serverKey))
	sig := hex.EncodeToString(h.Sum(nil))

	payload, _ := json.Marshal(map[string]any{
		"order_id": orderID, "status_code": statusCode, "gross_amount": gross,
		"signature_key": sig, "transaction_status": "settlement", "payment_type": "qris",
	})

	g := NewMidtransGateway(MidtransConfig{ServerKey: serverKey})
	res, err := g.VerifyWebhook(context.Background(), payload, nil)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if res.Status != PaymentSettled {
		t.Fatalf("expected settled, got %s", res.Status)
	}
	if res.AmountIDR != 39000 {
		t.Fatalf("expected 39000, got %d", res.AmountIDR)
	}
}

// TestMidtrans_VerifyWebhook_InvalidSignature verifies bad signature rejected.
func TestMidtrans_VerifyWebhook_InvalidSignature(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"order_id": "INV-1", "status_code": "200", "gross_amount": "1000.00",
		"signature_key": "deadbeef", "transaction_status": "settlement",
	})
	g := NewMidtransGateway(MidtransConfig{ServerKey: "server-key"})
	if _, err := g.VerifyWebhook(context.Background(), payload, nil); err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

// TestMidtrans_CheckStatus verifies server-side status pull.
func TestMidtrans_CheckStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/status") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order_id": "INV-1", "transaction_status": "capture", "gross_amount": "39000.00",
		})
	}))
	defer srv.Close()

	g := NewMidtransGateway(MidtransConfig{ServerKey: "k", CoreBaseURL: srv.URL})
	res, err := g.CheckStatus(context.Background(), "INV-1")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if res.Status != PaymentSettled {
		t.Fatalf("expected settled, got %s", res.Status)
	}
}

// TestMidtrans_MapStatus verifies status mapping.
func TestMidtrans_MapStatus(t *testing.T) {
	cases := map[string]PaymentStatus{
		"settlement": PaymentSettled,
		"capture":    PaymentSettled,
		"pending":    PaymentPending,
		"deny":       PaymentFailed,
		"cancel":     PaymentFailed,
		"expire":     PaymentExpired,
		"refund":     PaymentRefunded,
	}
	for in, want := range cases {
		if got := mapMidtransStatus(in); got != want {
			t.Errorf("mapMidtransStatus(%q)=%s want %s", in, got, want)
		}
	}
}

// TestMidtrans_ParseAmount verifies "39000.00" -> 39000 (integer Rupiah).
func TestMidtrans_ParseAmount(t *testing.T) {
	if got := parseAmount("39000.00"); got != 39000 {
		t.Fatalf("expected 39000, got %d", got)
	}
	if got := parseAmount("0.00"); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := parseAmount(""); got != 0 {
		t.Fatalf("expected 0 for empty, got %d", got)
	}
}

// TestMidtrans_CreateWithoutKey verifies graceful config error.
func TestMidtrans_CreateWithoutKey(t *testing.T) {
	g := NewMidtransGateway(MidtransConfig{})
	if _, err := g.CreateTransaction(context.Background(), CreateTxRequest{OrderID: "x", AmountIDR: 1}); err != ErrGatewayNotConfigured {
		t.Fatalf("expected ErrGatewayNotConfigured, got %v", err)
	}
}

var _ = fmt.Sprintf
