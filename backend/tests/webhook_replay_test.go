package tests

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/usecase"
)

// TestP08_002_WebhookReplayIdempotency fires 10 simultaneous webhook callbacks
// carrying identical order_id and signature in < 50ms.
// Invariant: exactly 1 request processes/mutates status, exactly 9 requests are
// answered idempotently with already_processed=true, invoice is marked paid exactly
// once, and zero duplicate payment rows exist.
func TestP08_002_WebhookReplayIdempotency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping webhook replay test")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	seedPlanCatalogue(t, pool)

	paymentRepo := postgres.NewPaymentRepo(pool)
	billingRepo := postgres.NewBillingRepo(pool)
	txManager := postgres.NewTxManager(pool)

	const testServerKey = "SB-Mid-server-TEST123456"
	payRegistry := payment.NewRegistry()
	payRegistry.Register(payment.NewMidtransGateway(payment.MidtransConfig{
		ServerKey:    testServerKey,
		ClientKey:    "SB-Mid-client-TEST123456",
		IsProduction: false,
	}))

	subActivator := usecase.NewSubscriptionActivator(billingRepo, billingRepo, nil)
	payWebhook := payment.NewWebhookProcessor(paymentRepo, payRegistry, subActivator, txManager)
	payH := handler.NewPaymentHandler(payWebhook)

	engine := deliveryHTTP.SetupRouter(
		deliveryHTTP.RouterConfig{
			JWTSecret: entTestSecret,
			Pool:      pool,
		},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, payH, nil, nil,
	)
	srv := httptest.NewServer(engine)
	defer srv.Close()

	// 1. Provision user
	userID := uuid.New()
	userEmail := fmt.Sprintf("pay-%s@fintrack.test", userID.String()[:8])
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, role)
		 VALUES ($1, $2, 'dummy', 'Pay Test User', 'user')`,
		userID, userEmail); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	}()

	// 2. Provision invoice & payment in pending state
	invID := uuid.New()
	orderID := fmt.Sprintf("INV-REPLAY-%s", uuid.New().String()[:8])
	grossAmountStr := "150000.00"
	grossAmountInt := int64(150000)
	planCode := "pro"

	if err := paymentRepo.CreateInvoice(ctx, &payment.Invoice{
		ID:            invID,
		InvoiceNumber: orderID,
		UserID:        userID,
		SubtotalIDR:   grossAmountInt,
		TaxIDR:        0,
		TotalIDR:      grossAmountInt,
		Status:        payment.InvoiceIssued,
		Notes:         &planCode,
	}); err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	payID := uuid.New()
	refCopy := orderID
	if err := paymentRepo.CreatePayment(ctx, &payment.Payment{
		ID:             payID,
		InvoiceID:      invID,
		UserID:         userID,
		Gateway:        "midtrans",
		GatewayRef:     &refCopy,
		AmountIDR:      grossAmountInt,
		Status:         payment.PaymentPending,
		IdempotencyKey: "idem-" + orderID,
	}); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	// 3. Precompute valid Midtrans SHA-512 signature
	// signature = SHA512(order_id + status_code + gross_amount + server_key)
	statusCode := "200"
	rawSig := orderID + statusCode + grossAmountStr + testServerKey
	hash := sha512.Sum512([]byte(rawSig))
	signatureHex := hex.EncodeToString(hash[:])

	webhookPayload, err := json.Marshal(map[string]any{
		"order_id":           orderID,
		"status_code":        statusCode,
		"gross_amount":       grossAmountStr,
		"signature_key":      signatureHex,
		"transaction_status": "settlement",
		"payment_type":       "qris",
		"transaction_time":   "2026-10-12 08:00:00",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	const concurrency = 10
	var firstProcessCount, alreadyProcessedCount, errorCount int32
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-startBarrier

			req, err := http.NewRequestWithContext(context.Background(),
				http.MethodPost, srv.URL+"/api/v1/payments/webhook/midtrans",
				bytes.NewReader(webhookPayload))
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Idempotency-Key", "idem-"+orderID)

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				t.Logf("unexpected status %d: %s", resp.StatusCode, string(body))
				atomic.AddInt32(&errorCount, 1)
				return
			}

			var res struct {
				Success          bool   `json:"success"`
				Status           string `json:"status"`
				AlreadyProcessed bool   `json:"already_processed"`
			}
			if err := json.Unmarshal(body, &res); err != nil {
				atomic.AddInt32(&errorCount, 1)
				return
			}

			if res.AlreadyProcessed {
				atomic.AddInt32(&alreadyProcessedCount, 1)
			} else {
				atomic.AddInt32(&firstProcessCount, 1)
			}
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	t.Logf("Replay results: first_process=%d, already_processed=%d, error=%d",
		firstProcessCount, alreadyProcessedCount, errorCount)

	if errorCount != 0 {
		t.Fatalf("expected 0 errors, got %d", errorCount)
	}
	if firstProcessCount != 1 {
		t.Fatalf("expected exactly 1 first process, got %d", firstProcessCount)
	}
	if alreadyProcessedCount != 9 {
		t.Fatalf("expected exactly 9 already_processed, got %d", alreadyProcessedCount)
	}

	// 4. Verify DB state: invoice is paid, payment is settled, exactly 1 payment record
	inv, err := paymentRepo.GetInvoiceByID(ctx, invID)
	if err != nil || inv == nil {
		t.Fatalf("get invoice: %v", err)
	}
	if inv.Status != payment.InvoicePaid {
		t.Errorf("expected invoice status paid, got %s", inv.Status)
	}

	pay, err := paymentRepo.GetPaymentByID(ctx, payID)
	if err != nil || pay == nil {
		t.Fatalf("get payment: %v", err)
	}
	if pay.Status != payment.PaymentSettled {
		t.Errorf("expected payment status settled, got %s", pay.Status)
	}

	var paymentCount int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM payments WHERE gateway_ref = $1`, orderID).Scan(&paymentCount)
	if err != nil {
		t.Fatalf("query payment count: %v", err)
	}
	if paymentCount != 1 {
		t.Errorf("expected exactly 1 payment row, got %d", paymentCount)
	}
}
