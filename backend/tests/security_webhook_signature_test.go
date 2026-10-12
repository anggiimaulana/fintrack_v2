package tests

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/payment"
)

type noopTxRunner struct{}

func (noopTxRunner) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type mockPaymentRepo struct {
	payment.Repository
}

func (m *mockPaymentRepo) GetPaymentByGatewayRefForUpdate(_ context.Context, _ string) (*payment.Payment, error) {
	return nil, nil // not found -> ignored
}

func (m *mockPaymentRepo) GetPaymentByGatewayRef(_ context.Context, _ string) (*payment.Payment, error) {
	return nil, nil
}

func (m *mockPaymentRepo) UpdatePaymentStatus(_ context.Context, _ uuid.UUID, _ payment.PaymentStatus, _ *string, _ *string, _ map[string]any) error {
	return nil
}

// TestP08_007_CryptographicWebhookSignatureValidation verifies that invalid, forged,
// or tampered webhook signatures are immediately rejected with HTTP 401 Unauthorized,
// preventing spoofed payment activations and timing attacks.
func TestP08_007_CryptographicWebhookSignatureValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const serverKey = "SB-Mid-server-SECRET-KEY-999"
	reg := payment.NewRegistry()
	reg.Register(payment.NewMidtransGateway(payment.MidtransConfig{
		ServerKey:    serverKey,
		IsProduction: false,
	}))

	processor := payment.NewWebhookProcessor(&mockPaymentRepo{}, reg, nil, noopTxRunner{})
	payH := handler.NewPaymentHandler(processor)

	engine := deliveryHTTP.SetupRouter(
		deliveryHTTP.RouterConfig{JWTSecret: entTestSecret},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, payH, nil, nil,
	)

	orderID := "INV-SEC-TEST-001"
	statusCode := "200"
	grossAmount := "150000.00"

	// Compute legitimate signature
	h := sha512.New()
	h.Write([]byte(orderID + statusCode + grossAmount + serverKey))
	validSig := hex.EncodeToString(h.Sum(nil))

	cases := []struct {
		name         string
		payload      map[string]any
		expectedCode int
	}{
		{
			name: "Completely invalid/garbage signature -> HTTP 401",
			payload: map[string]any{
				"order_id":           orderID,
				"status_code":        statusCode,
				"gross_amount":       grossAmount,
				"signature_key":      "deadbeef0102030405060708090a0b0c0d0e0f",
				"transaction_status": "settlement",
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Tampered gross amount (150000 -> 1000) with original signature -> HTTP 401",
			payload: map[string]any{
				"order_id":           orderID,
				"status_code":        statusCode,
				"gross_amount":       "1000.00",
				"signature_key":      validSig,
				"transaction_status": "settlement",
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Tampered order ID with original signature -> HTTP 401",
			payload: map[string]any{
				"order_id":           "INV-HACKED-999",
				"status_code":        statusCode,
				"gross_amount":       grossAmount,
				"signature_key":      validSig,
				"transaction_status": "settlement",
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Tampered status code with original signature -> HTTP 401",
			payload: map[string]any{
				"order_id":           orderID,
				"status_code":        "201",
				"gross_amount":       grossAmount,
				"signature_key":      validSig,
				"transaction_status": "settlement",
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Valid signature (payment not in DB) -> HTTP 200 ignored",
			payload: map[string]any{
				"order_id":           orderID,
				"status_code":        statusCode,
				"gross_amount":       grossAmount,
				"signature_key":      validSig,
				"transaction_status": "settlement",
				"payment_type":       "qris",
			},
			expectedCode: http.StatusOK, // Passes cryptographic verification, ignores unknown order
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook/midtrans", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			engine.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Fatalf("expected HTTP %d, got %d (body: %s)", tc.expectedCode, w.Code, w.Body.String())
			}
		})
	}
}
