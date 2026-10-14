// Package tests — API contract regression tests.
//
// These lock down the HTTP status codes for the edge cases that were wrong
// before: a missing resource answered 200/500 instead of 404, a nil dereference
// turned a "not found" into a panic-driven 500, and quota exhaustion returned a
// bespoke body instead of the standard upgrade envelope. Every case here is a
// distinct status-code contract, so a regression fails loudly.
//
// Gated on TEST_DATABASE_URL like the other integration suites:
//
//	TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/fintrack_test \
//	  go test -run TestAPIContract ./tests/...
package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/infrastructure/auth"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/usecase"
)

type contractHarness struct {
	t    *testing.T
	pool *pgxpool.Pool
	srv  *httptest.Server
}

// newContractHarness boots the real router with the use cases needed by the
// transaction / budget / category / notification / AI-session routes.
func newContractHarness(t *testing.T) *contractHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := getTestDSN(t)
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	t.Cleanup(pool.Close)

	seedPlanCatalogue(t, pool)

	txRepo := postgres.NewTransactionRepo(pool)
	txUC := usecase.NewTransactionUsecase(txRepo, nil, nil, nil, nil)
	userUC := usecase.NewUserUsecase(postgres.NewUserRepo(pool))
	budgetRepo := postgres.NewBudgetRepo(pool)
	budgetUC := usecase.NewBudgetUsecase(budgetRepo, txRepo)
	catUC := usecase.NewCategoryUsecase(
		postgres.NewIncomeCategoryRepo(pool),
		postgres.NewExpenseCategoryRepo(pool),
		postgres.NewSavingsCategoryRepo(pool),
		postgres.NewExpenseSubcategoryRepo(pool),
	)
	notifRepo := postgres.NewNotificationRepo(pool)
	notifH := handler.NewNotificationHandler(notifRepo, nil)
	aiUC := usecase.NewAIUsecase(
		postgres.NewChatSessionRepo(pool),
		postgres.NewChatMessageRepo(pool),
		txRepo, nil, nil, nil, nil, nil, nil,
	)

	engine := deliveryHTTP.SetupRouter(
		deliveryHTTP.RouterConfig{
			JWTSecret:     entTestSecret,
			Pool:          pool,
			Notifications: notifH,
		},
		txUC, catUC, userUC, nil, nil, nil, budgetUC, nil, aiUC, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	gin.SetMode(gin.TestMode)
	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)

	return &contractHarness{t: t, pool: pool, srv: srv}
}

// do issues an authenticated request and returns the status code.
func (h *contractHarness) do(method, path string, token string) int {
	h.t.Helper()
	return h.doBody(method, path, token, nil)
}

// doBody issues an authenticated request with an optional JSON body.
func (h *contractHarness) doBody(method, path, token string, body []byte) int {
	h.t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, h.srv.URL+path, reader)
	if err != nil {
		h.t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.t.Fatalf("do %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// seedUser inserts a user and returns its id and an access token.
func (h *contractHarness) seedUser(role string) (uuid.UUID, string) {
	h.t.Helper()
	id := uuid.New()
	email := fmt.Sprintf("contract-%s@fintrack.test", id.String()[:8])
	hash, _ := auth.HashPassword("SuperSecret123!")
	if _, err := h.pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password_hash, full_name, role) VALUES ($1,$2,$3,'Contract',$4)`,
		id, email, hash, role); err != nil {
		h.t.Fatalf("insert user: %v", err)
	}
	h.t.Cleanup(func() {
		_, _ = h.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	tok, err := auth.IssueAccessToken(id, role, email, []byte(entTestSecret), time.Hour)
	if err != nil {
		h.t.Fatalf("issue token: %v", err)
	}
	return id, tok
}

// TestAPIContract_NotFoundStatuses asserts that operating on a non-existent
// resource answers 404 (not 200, not 500).
func TestAPIContract_NotFoundStatuses(t *testing.T) {
	h := newContractHarness(t)
	_, tok := h.seedUser("user")
	missing := uuid.NewString()

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{"GET transaction missing", http.MethodGet, "/api/v1/transactions/" + missing, nil},
		{"DELETE transaction missing", http.MethodDelete, "/api/v1/transactions/" + missing, nil},
		{"DELETE budget missing", http.MethodDelete, "/api/v1/budgets/" + missing, nil},
		{"PUT ai session missing", http.MethodPut, "/api/v1/ai/sessions/" + missing, []byte(`{"title":"x"}`)},
		{"DELETE ai session missing", http.MethodDelete, "/api/v1/ai/sessions/" + missing, nil},
		{"GET notification missing", http.MethodGet, "/api/v1/me/notifications/" + missing, nil},
		{"DELETE notification missing", http.MethodDelete, "/api/v1/me/notifications/" + missing, nil},
		{"POST notification read missing", http.MethodPost, "/api/v1/me/notifications/" + missing + "/read", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if code := h.doBody(tc.method, tc.path, tok, tc.body); code != http.StatusNotFound {
				t.Fatalf("%s %s: got %d, want 404", tc.method, tc.path, code)
			}
		})
	}
}

// TestAPIContract_UpdateMissingIs404 asserts a well-formed PUT for a missing
// transaction answers 404 (the handler must bind the body first, then report
// the missing row).
func TestAPIContract_UpdateMissingIs404(t *testing.T) {
	h := newContractHarness(t)
	_, tok := h.seedUser("user")
	missing := uuid.NewString()

	body := []byte(`{"amount":5000}`)
	if code := h.doBody(http.MethodPut, "/api/v1/transactions/"+missing, tok, body); code != http.StatusNotFound {
		t.Fatalf("PUT missing transaction: got %d, want 404", code)
	}
}

// TestAPIContract_NotFoundRequiresValidUUID asserts a malformed id is a 400, so
// a bad id is never confused with a missing resource.
func TestAPIContract_NotFoundRequiresValidUUID(t *testing.T) {
	h := newContractHarness(t)
	_, tok := h.seedUser("user")

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"GET transaction bad id", http.MethodGet, "/api/v1/transactions/not-a-uuid"},
		{"DELETE budget bad id", http.MethodDelete, "/api/v1/budgets/not-a-uuid"},
		{"DELETE ai session bad id", http.MethodDelete, "/api/v1/ai/sessions/not-a-uuid"},
		{"GET notification bad id", http.MethodGet, "/api/v1/me/notifications/not-a-uuid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if code := h.do(tc.method, tc.path, tok); code != http.StatusBadRequest {
				t.Fatalf("%s %s: got %d, want 400", tc.method, tc.path, code)
			}
		})
	}
}

// TestAPIContract_DeleteThenDeleteIs404 asserts a second delete of the same
// resource is a 404, proving deletes report the real outcome.
func TestAPIContract_DeleteThenDeleteIs404(t *testing.T) {
	h := newContractHarness(t)
	userID, tok := h.seedUser("user")

	// Seed an expense category and a transaction owned by the user (the DB
	// enforces that an expense transaction carries an expense_category_id).
	catID := uuid.New()
	if _, err := h.pool.Exec(context.Background(),
		`INSERT INTO expense_categories (id, name, icon, color) VALUES ($1,$2,'circle','#000000')`,
		catID, "Contract Cat "+catID.String()[:8]); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.pool.Exec(context.Background(), `DELETE FROM expense_categories WHERE id = $1`, catID)
	})

	txID := uuid.New()
	if _, err := h.pool.Exec(context.Background(), `
		INSERT INTO transactions (id, user_id, type, amount, expense_category_id, description, transaction_date)
		VALUES ($1,$2,'expense',1000,$3,'contract test', now())`, txID, userID, catID); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	if code := h.do(http.MethodDelete, "/api/v1/transactions/"+txID.String(), tok); code != http.StatusOK {
		t.Fatalf("first delete: got %d, want 200", code)
	}
	if code := h.do(http.MethodDelete, "/api/v1/transactions/"+txID.String(), tok); code != http.StatusNotFound {
		t.Fatalf("second delete: got %d, want 404", code)
	}
}

// TestAPIContract_CategoryDeleteNotFound asserts deleting a missing category
// is a 404, not a 200.
func TestAPIContract_CategoryDeleteNotFound(t *testing.T) {
	h := newContractHarness(t)
	_, tok := h.seedUser("admin")
	missing := uuid.NewString()

	paths := []string{
		"/api/v1/admin/categories/income/" + missing,
		"/api/v1/admin/categories/expense/" + missing,
		"/api/v1/admin/categories/savings/" + missing,
		"/api/v1/admin/subcategories/" + missing,
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			if code := h.do(http.MethodDelete, p, tok); code != http.StatusNotFound {
				t.Fatalf("DELETE %s: got %d, want 404", p, code)
			}
		})
	}
}

// TestAPIContract_AdminSetUserStatusNotFound asserts toggling a missing user's
// status is a 404.
func TestAPIContract_AdminSetUserStatusNotFound(t *testing.T) {
	h := newContractHarness(t)
	_, tok := h.seedUser("admin")
	missing := uuid.NewString()

	body := `{"is_active":false}`
	req, _ := http.NewRequest(http.MethodPut, h.srv.URL+"/api/v1/admin/users/"+missing+"/status",
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d, want 404", resp.StatusCode)
	}
}

// TestAPIContract_SelfDeactivateIs400 asserts the self-deactivation guard stays
// a 400 (a business rule, not a missing resource).
func TestAPIContract_SelfDeactivateIs400(t *testing.T) {
	h := newContractHarness(t)
	adminID, tok := h.seedUser("admin")

	body := `{"is_active":false}`
	req, _ := http.NewRequest(http.MethodPut, h.srv.URL+"/api/v1/admin/users/"+adminID.String()+"/status",
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", resp.StatusCode)
	}
}
