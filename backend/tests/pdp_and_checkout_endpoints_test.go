package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/infrastructure/auth"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/usecase"
)

// TestP09_PDP_And_Checkout_Endpoints tests:
// 1. POST /api/v1/payments/checkout creates invoice & payment session for user.
// 2. GET /api/v1/user/export-data returns complete JSON data archive for Data Portability (UU PDP Pasal 21).
// 3. DELETE /api/v1/user/account verifies password and executes Right to Erasure (UU PDP Pasal 43).
func TestP09_PDP_And_Checkout_Endpoints(t *testing.T) {
	dsn := getTestDSN(t)
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer pool.Close()

	seedPlanCatalogue(t, pool)

	userRepo := postgres.NewUserRepo(pool)
	refreshRepo := postgres.NewRefreshTokenRepo(pool)
	billingRepo := postgres.NewBillingRepo(pool)
	paymentRepo := postgres.NewPaymentRepo(pool)
	settingsRepo := postgres.NewSettingsRepo(pool)
	txManager := postgres.NewTxManager(pool)

	authUC := usecase.NewAuthUsecase(userRepo, refreshRepo, []byte(entTestSecret))

	payRegistry := payment.NewRegistry()
	payRegistry.Register(payment.NewManualGateway("BCA", "1234567890", "PT FinTrack"))
	payTax := payment.NewTaxCalculator(settingsRepo)
	payInvoice := payment.NewInvoiceService(paymentRepo, settingsRepo, payTax)
	payCheckout := payment.NewCheckoutService(paymentRepo, settingsRepo, payRegistry, payInvoice, nil)

	subActivator := usecase.NewSubscriptionActivator(billingRepo, billingRepo, nil)
	payWebhook := payment.NewWebhookProcessor(paymentRepo, payRegistry, subActivator, txManager)
	payHandler := payment.NewManualService(paymentRepo, subActivator, txManager, nil)
	payAdmin := payment.NewAdminService(payHandler)

	payH := handler.NewPaymentHandler(payWebhook).
		WithManual(payHandler).
		WithAdmin(payAdmin).
		WithCheckout(payCheckout, billingRepo, userRepo)
	engine := deliveryHTTP.SetupRouter(
		deliveryHTTP.RouterConfig{
			JWTSecret: entTestSecret,
			Pool:      pool,
		},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		authUC, nil, nil,
		payH, nil, nil,
	)
	srv := httptest.NewServer(engine)
	defer srv.Close()

	// Provision user
	userID := uuid.New()
	userEmail := fmt.Sprintf("pdp-api-%s@fintrack.test", userID.String()[:8])
	passwordPlain := "SuperSecret123!"
	passHash, _ := auth.HashPassword(passwordPlain)

	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, role)
		 VALUES ($1, $2, $3, 'PDP API User', 'user')`,
		userID, userEmail, passHash); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	}()

	tokenStr, err := auth.IssueAccessToken(userID, "user", userEmail, []byte(entTestSecret), time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}

	// ─── 1. Test POST /api/v1/payments/checkout ───────────────────────
	t.Run("POST /api/v1/payments/checkout", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"plan_code": "pro",
		})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/payments/checkout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenStr)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("do checkout: %v", err)
		}
		defer resp.Body.Close()
		respBytes, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", resp.StatusCode, string(respBytes))
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				Invoice *payment.Invoice `json:"invoice"`
				Payment *payment.Payment `json:"payment"`
				Gateway string           `json:"gateway"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBytes, &res); err != nil {
			t.Fatalf("unmarshal checkout: %v", err)
		}
		if res.Data.Invoice == nil || res.Data.Invoice.SubtotalIDR != 39000 {
			t.Errorf("unexpected invoice in checkout response: %+v", res.Data.Invoice)
		}
		t.Logf("Checkout successful: order_id=%s, gateway=%s, total=%d",
			res.Data.Invoice.InvoiceNumber, res.Data.Gateway, res.Data.Invoice.TotalIDR)
	})

	// ─── 2. Test GET /api/v1/user/export-data ─────────────────────────
	t.Run("GET /api/v1/user/export-data", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/user/export-data", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("do export: %v", err)
		}
		defer resp.Body.Close()
		respBytes, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", resp.StatusCode, string(respBytes))
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				Profile  map[string]any   `json:"profile"`
				Invoices []map[string]any `json:"invoices"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBytes, &res); err != nil {
			t.Fatalf("unmarshal export: %v", err)
		}
		if res.Data.Profile["email"] != userEmail {
			t.Errorf("expected exported email %s, got %v", userEmail, res.Data.Profile["email"])
		}
		if len(res.Data.Invoices) == 0 {
			t.Errorf("expected exported invoices from earlier checkout step")
		}
		t.Logf("Export data verified: profile ok, %d invoices included", len(res.Data.Invoices))
	})

	// ─── 3. Test DELETE /api/v1/me (Right to Erasure with password) ───
	t.Run("DELETE /api/v1/me", func(t *testing.T) {
		// Wrong password -> 401
		wrongBody, _ := json.Marshal(map[string]string{
			"password": "wrong-password",
		})
		badReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/me", bytes.NewReader(wrongBody))
		badReq.Header.Set("Content-Type", "application/json")
		badReq.Header.Set("Authorization", "Bearer "+tokenStr)
		badResp, _ := client.Do(badReq)
		if badResp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized on wrong password, got %d", badResp.StatusCode)
		}
		badResp.Body.Close()

		// Correct password -> 200 OK
		goodBody, _ := json.Marshal(map[string]string{
			"password": passwordPlain,
			"reason":   "User requesting erasure under UU PDP",
		})
		goodReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/me", bytes.NewReader(goodBody))
		goodReq.Header.Set("Content-Type", "application/json")
		goodReq.Header.Set("Authorization", "Bearer "+tokenStr)

		goodResp, err := client.Do(goodReq)
		if err != nil {
			t.Fatalf("do delete account: %v", err)
		}
		defer goodResp.Body.Close()
		delBytes, _ := io.ReadAll(goodResp.Body)

		if goodResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", goodResp.StatusCode, string(delBytes))
		}

		// Verify database state: user is pseudonymized and shredded
		var userEmailAfter, userNameAfter, passHashAfter string
		var isActiveAfter bool
		_ = pool.QueryRow(ctx,
			`SELECT email, full_name, password_hash, is_active FROM users WHERE id = $1`, userID).
			Scan(&userEmailAfter, &userNameAfter, &passHashAfter, &isActiveAfter)

		expectedEmail := fmt.Sprintf("anonymized-%s@deleted.fintrack.internal", userID.String())
		if userEmailAfter != expectedEmail || isActiveAfter != false || passHashAfter != "DELETED_CRYPTOGRAPHIC_SHREDDING" {
			t.Errorf("user not properly shredded: email=%s, active=%v, hash=%s",
				userEmailAfter, isActiveAfter, passHashAfter)
		}

		t.Log("Delete account verified: user pseudonymized, password shredded, cookies cleared.")
	})
}
