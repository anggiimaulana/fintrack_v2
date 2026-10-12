// Package tests — entitlement enforcement end-to-end.
//
// This suite boots the REAL router (deliveryHTTP.SetupRouter) over real
// middleware, handlers, repositories and PostgreSQL. That matters: the original
// P07 defect was middleware that existed, passed its unit tests, and was wired
// to nothing. Only a test through the actual router can catch that class of bug,
// so the assertions here are "a Free user is blocked on a gated route" rather
// than "the middleware function returns 403".
//
// Gated on TEST_DATABASE_URL so `go test ./...` stays green without a database.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/internal/infrastructure/auth"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/usecase"
)

const entTestSecret = "test-secret-at-least-32-characters-long"

// Plan IDs are fixed by the seed data (database/seeds/0002_subscription_token_seeds.sql).
var (
	freePlanID     = uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	proPlanID      = uuid.MustParse("a0000000-0000-0000-0000-000000000002")
	businessPlanID = uuid.MustParse("a0000000-0000-0000-0000-000000000003")
)

type entHarness struct {
	t    *testing.T
	pool *pgxpool.Pool
	srv  *httptest.Server
}

// newEntHarness wires the real router with entitlement enforcement enabled.
//
// The test database is deliberately seed-free (see database/README.md), so the
// plan catalogue is provisioned here rather than assumed. Values mirror
// database/seeds/0002_subscription_token_seeds.sql.
func newEntHarness(t *testing.T) *entHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping entitlement enforcement tests")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(pool.Close)

	seedPlanCatalogue(t, pool)

	txRepo := postgres.NewTransactionRepo(pool)
	billingRepo := postgres.NewBillingRepo(pool)
	reportUC := usecase.NewReportUsecase(txRepo)
	txUC := usecase.NewTransactionUsecase(txRepo, nil, nil, nil, nil)
	// A real AIUsecase: /ai/chat streams from a goroutine, so a nil usecase
	// panics rather than returning an error. No embedder or gateway is wired, so
	// a permitted request fails downstream — which is fine, these tests only
	// assert whether the entitlement gate let it through.
	aiUC := usecase.NewAIUsecase(
		postgres.NewChatSessionRepo(pool),
		postgres.NewChatMessageRepo(pool),
		postgres.NewGeminiKeyRepo(pool),
		txRepo,
		postgres.NewIncomeCategoryRepo(pool),
		postgres.NewExpenseCategoryRepo(pool),
		postgres.NewSavingsCategoryRepo(pool),
		postgres.NewUserRepo(pool),
		make([]byte, 32),
		nil, nil,
	)

	entResolver := entitlement.NewResolver(postgres.NewEntitlementSource(billingRepo), nil)
	entMw := middleware.NewEntitlementMiddleware(entResolver)
	usage := &middleware.UsageProviders{
		CountTransactions: txRepo.CountActiveByUser,
		CurrentTokens:     middleware.TokenPeriodUsage(billingRepo),
		CountVisionScans: func(ctx context.Context, userID uuid.UUID) (int64, error) {
			return billingRepo.CountVisionScansInPeriod(ctx, userID, time.Now().UTC().Unix())
		},
	}

	// Only the usecases the exercised routes need are supplied; SetupRouter
	// tolerates nil for the rest (they are never reached in these tests).
	engine := deliveryHTTP.SetupRouter(
		deliveryHTTP.RouterConfig{
			JWTSecret:   entTestSecret,
			Entitlement: entMw,
			Usage:       usage,
		},
		txUC, nil, nil, nil, nil, nil, aiUC, nil, reportUC, nil, nil, nil, nil, nil, nil,
	)
	// The router is built in ReleaseMode; these tests assert on response bodies.
	gin.SetMode(gin.TestMode)

	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)

	return &entHarness{t: t, pool: pool, srv: srv}
}

// seedPlanCatalogue provisions the free/pro/business plans with the feature and
// limit rows the gates read. Idempotent, so it is safe across repeated runs and
// against a database that already has the seed data.
func seedPlanCatalogue(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	plans := []struct {
		id       uuid.UUID
		code     string
		name     string
		fallback string
		order    int
		price    int64
	}{
		{freePlanID, "free", "Free", "manual", 1, 0},
		{proPlanID, "pro", "Pro", "midtrans", 2, 39000},
		{businessPlanID, "business", "Business", "midtrans", 3, 99000},
	}
	for _, p := range plans {
		if _, err := pool.Exec(ctx, `
			INSERT INTO plans (id, code, name, price_monthly_idr, billing_period_days, is_active, is_public, fallback_gateway, sort_order)
			VALUES ($1,$2,$3,$4,30,true,true,$5,$6)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, price_monthly_idr = EXCLUDED.price_monthly_idr`,
			p.id, p.code, p.name, p.price, p.fallback, p.order); err != nil {
			t.Fatalf("seed plan %s: %v", p.code, err)
		}
	}

	// feature_key -> enabled, per plan.
	features := map[uuid.UUID]map[string]bool{
		freePlanID: {
			"ai_chat": true, "ocr_receipt": false, "voice_input": false,
			"rag_search": false, "reports_export": false, "advanced_budgeting": false,
			"api_access": false, "bank_sync": false,
		},
		proPlanID: {
			"ai_chat": true, "ocr_receipt": true, "voice_input": true,
			"rag_search": true, "reports_export": true, "advanced_budgeting": true,
			"api_access": false, "bank_sync": false,
		},
		businessPlanID: {
			"ai_chat": true, "ocr_receipt": true, "voice_input": true,
			"rag_search": true, "reports_export": true, "advanced_budgeting": true,
			"api_access": true, "bank_sync": false,
		},
	}
	for planID, kv := range features {
		for key, enabled := range kv {
			if _, err := pool.Exec(ctx, `
				INSERT INTO plan_entitlements (plan_id, feature_key, is_enabled)
				VALUES ($1,$2,$3)
				ON CONFLICT (plan_id, feature_key) DO UPDATE SET is_enabled = EXCLUDED.is_enabled`,
				planID, key, enabled); err != nil {
				t.Fatalf("seed entitlement %s/%s: %v", planID, key, err)
			}
		}
	}

	// limit_key -> value (-1 = unlimited).
	limits := map[uuid.UUID]map[string]int64{
		freePlanID: {
			"ai_tokens_per_cycle": 5000, "ai_requests_per_cycle": 50,
			"max_transactions": 100, "max_categories": 5, "max_wallets": 1,
			"ocr_scans_per_cycle": 0,
		},
		proPlanID: {
			"ai_tokens_per_cycle": 150000, "ai_requests_per_cycle": 2000,
			"max_transactions": 5000, "max_categories": 30, "max_wallets": 5,
			"ocr_scans_per_cycle": 100,
		},
		businessPlanID: {
			"ai_tokens_per_cycle": 1000000, "ai_requests_per_cycle": -1,
			"max_transactions": -1, "max_categories": -1, "max_wallets": -1,
			"ocr_scans_per_cycle": -1,
		},
	}
	for planID, kv := range limits {
		for key, value := range kv {
			if _, err := pool.Exec(ctx, `
				INSERT INTO plan_limits (plan_id, limit_key, limit_value)
				VALUES ($1,$2,$3)
				ON CONFLICT (plan_id, limit_key) DO UPDATE SET limit_value = EXCLUDED.limit_value`,
				planID, key, value); err != nil {
				t.Fatalf("seed limit %s/%s: %v", planID, key, err)
			}
		}
	}
}

type entEnvelope struct {
	Success    bool            `json:"success"`
	Error      string          `json:"error"`
	UpgradeURL string          `json:"upgrade_url"`
	Data       json.RawMessage `json:"data"`
}

func (h *entHarness) do(method, path, token string, body any) (int, entEnvelope) {
	h.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, h.srv.URL+"/api/v1"+path, reader)
	if err != nil {
		h.t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := h.srv.Client().Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		h.t.Fatalf("read: %v", err)
	}
	var env entEnvelope
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &env)
	}
	return resp.StatusCode, env
}

// seedEntUser creates a user, optionally with a subscription on planID, and
// returns the user id plus a valid access token.
func (h *entHarness) seedEntUser(planID *uuid.UUID) (uuid.UUID, string) {
	h.t.Helper()
	ctx := h.t.Context()

	email := fmt.Sprintf("ent-%s@fintrack.test", uuid.NewString())
	var id uuid.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO users (email, full_name, role) VALUES ($1,'Ent User','user') RETURNING id`,
		email).Scan(&id); err != nil {
		h.t.Fatalf("seed user: %v", err)
	}
	if planID != nil {
		if _, err := h.pool.Exec(ctx, `
			INSERT INTO subscriptions (user_id, plan_id, status, current_period_start, current_period_end)
			VALUES ($1,$2,'active', now(), now() + interval '30 days')`,
			id, *planID); err != nil {
			h.t.Fatalf("seed subscription: %v", err)
		}
	}
	h.t.Cleanup(func() {
		bg := context.Background()
		_, _ = h.pool.Exec(bg, `DELETE FROM subscriptions WHERE user_id=$1`, id)
		_, _ = h.pool.Exec(bg, `DELETE FROM users WHERE id=$1`, id)
	})

	tok, err := auth.IssueAccessToken(id, "user", email, []byte(entTestSecret), 30*time.Minute)
	if err != nil {
		h.t.Fatalf("issue token: %v", err)
	}
	return id, tok
}

// seedTransactions inserts n live transactions for the user, plus optionally
// some soft-deleted ones. A transaction of type 'expense' requires a category
// (check_expense_category). Categories are global (created_by, not user_id) with
// a unique name, so one shared row is upserted rather than created per user.
func (h *entHarness) seedTransactions(userID uuid.UUID, n int, softDeleted int) {
	h.t.Helper()
	ctx := h.t.Context()

	var catID uuid.UUID
	if err := h.pool.QueryRow(ctx, `
		INSERT INTO expense_categories (name, created_by) VALUES ('Ent Test Category', $1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, userID).Scan(&catID); err != nil {
		h.t.Fatalf("seed category: %v", err)
	}

	for i := 0; i < n; i++ {
		if _, err := h.pool.Exec(ctx, `
			INSERT INTO transactions (user_id, type, amount, expense_category_id, description, transaction_date)
			VALUES ($1,'expense',1000,$2,'ent test',now())`, userID, catID); err != nil {
			h.t.Fatalf("seed tx: %v", err)
		}
	}
	for i := 0; i < softDeleted; i++ {
		if _, err := h.pool.Exec(ctx, `
			INSERT INTO transactions (user_id, type, amount, expense_category_id, description, transaction_date, is_deleted, deleted_at)
			VALUES ($1,'expense',1000,$2,'ent test deleted',now(),true,now())`, userID, catID); err != nil {
			h.t.Fatalf("seed deleted tx: %v", err)
		}
	}
}

// ─── feature gates ───────────────────────────────────────────

// TestEntitlement_ReportsFeatureGate is the primary wiring guard: a Free user
// must be refused by the real router, a Pro user allowed.
func TestEntitlement_ReportsFeatureGate(t *testing.T) {
	h := newEntHarness(t)

	_, freeTok := h.seedEntUser(&freePlanID)
	if code, env := h.do(http.MethodGet, "/reports", freeTok, nil); code != http.StatusForbidden {
		t.Fatalf("free user on /reports: got %d (%s), want 403", code, env.Error)
	} else if env.Error != "feature_not_in_plan" {
		t.Fatalf("error = %q, want feature_not_in_plan", env.Error)
	}

	_, proTok := h.seedEntUser(&proPlanID)
	if code, env := h.do(http.MethodGet, "/reports", proTok, nil); code != http.StatusOK {
		t.Fatalf("pro user on /reports: got %d (%s), want 200", code, env.Error)
	}
}

// TestEntitlement_UserWithoutSubscriptionTreatedAsFree is the regression guard
// for the lockout bug: a user with no subscription row must resolve to Free
// (allowed ai_chat, blocked ocr_receipt) rather than deny-all.
func TestEntitlement_UserWithoutSubscriptionTreatedAsFree(t *testing.T) {
	h := newEntHarness(t)
	_, tok := h.seedEntUser(nil) // no subscription row at all

	// Free blocks OCR...
	if code, env := h.do(http.MethodPost, "/ai/receipt", tok, map[string]any{}); code != http.StatusForbidden {
		t.Fatalf("no-subscription user on /ai/receipt: got %d (%s), want 403", code, env.Error)
	}
	// ...but must NOT be denied for a feature every plan includes. The request
	// may still fail downstream (no AI provider configured), but it must not be
	// refused by the entitlement gate.
	if code, env := h.do(http.MethodPost, "/ai/chat", tok, map[string]any{"message": "hi"}); code == http.StatusForbidden {
		t.Fatalf("no-subscription user on /ai/chat was denied by the gate: %s", env.Error)
	}
}

// TestEntitlement_VoiceGateBlocksFree checks a second feature key on a
// different route, so a single mis-wired route cannot pass the suite.
func TestEntitlement_VoiceGateBlocksFree(t *testing.T) {
	h := newEntHarness(t)

	_, freeTok := h.seedEntUser(&freePlanID)
	if code, env := h.do(http.MethodPost, "/ai/voice", freeTok, nil); code != http.StatusForbidden {
		t.Fatalf("free user on /ai/voice: got %d (%s), want 403", code, env.Error)
	}
}

// ─── limit gates ─────────────────────────────────────────────

// TestEntitlement_TransactionLimit verifies the numeric gate, including that
// soft-deleted rows do not count toward usage.
func TestEntitlement_TransactionLimit(t *testing.T) {
	h := newEntHarness(t)

	// Free limit is 100. 99 live + 1 soft-deleted = 99 counted -> allowed.
	allowedID, allowedTok := h.seedEntUser(&freePlanID)
	h.seedTransactions(allowedID, 99, 1)
	if code, env := h.do(http.MethodPost, "/transactions", allowedTok, map[string]any{
		"type": "expense", "amount": 1000, "description": "ok",
	}); code == http.StatusPaymentRequired {
		t.Fatalf("99 live (+1 deleted) must be under the Free limit, got 402: %s", env.Error)
	}

	// At exactly 100 live -> 402 limit_exceeded.
	fullID, fullTok := h.seedEntUser(&freePlanID)
	h.seedTransactions(fullID, 100, 0)
	code, env := h.do(http.MethodPost, "/transactions", fullTok, map[string]any{
		"type": "expense", "amount": 1000, "description": "over",
	})
	if code != http.StatusPaymentRequired {
		t.Fatalf("100 live must hit the Free limit, got %d (%s)", code, env.Error)
	}
	if env.Error != "limit_exceeded" {
		t.Fatalf("error = %q, want limit_exceeded", env.Error)
	}

	// Business is unlimited (-1): even a user far over the Free number passes.
	overID, overTok := h.seedEntUser(&businessPlanID)
	h.seedTransactions(overID, 101, 0)
	if code, env := h.do(http.MethodPost, "/transactions", overTok, map[string]any{
		"type": "expense", "amount": 1000, "description": "unlimited",
	}); code == http.StatusPaymentRequired {
		t.Fatalf("business (unlimited) must not be limited, got 402: %s", env.Error)
	}
}

// TestEntitlement_TokenQuotaWithoutPeriod verifies the "no period = zero usage"
// contract end to end: a Free user with no token period must not be 402'd on
// /ai/chat. Denying here would be the same lockout bug in a different guise.
func TestEntitlement_TokenQuotaWithoutPeriod(t *testing.T) {
	h := newEntHarness(t)
	_, tok := h.seedEntUser(&freePlanID)

	code, env := h.do(http.MethodPost, "/ai/chat", tok, map[string]any{"message": "hi"})
	if code == http.StatusPaymentRequired {
		t.Fatalf("no token period must mean zero usage, not 402: %s", env.Error)
	}
}

// ─── payload contract ────────────────────────────────────────

// TestEntitlement_UpgradeURLInPayload verifies the 403/402 bodies carry the
// link the frontend mirror points at.
func TestEntitlement_UpgradeURLInPayload(t *testing.T) {
	h := newEntHarness(t)

	_, freeTok := h.seedEntUser(&freePlanID)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"feature", http.MethodGet, "/reports", nil},
		{"limit", http.MethodPost, "/transactions", map[string]any{"type": "expense", "amount": 1000}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "limit" {
				// Push the user to the Free ceiling first.
				id, tok := h.seedEntUser(&freePlanID)
				h.seedTransactions(id, 100, 0)
				code, env := h.do(tc.method, tc.path, tok, tc.body)
				if code != http.StatusPaymentRequired {
					t.Fatalf("expected 402, got %d", code)
				}
				assertUpgradeURL(t, env)
				return
			}
			code, env := h.do(tc.method, tc.path, freeTok, tc.body)
			if code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d", code)
			}
			assertUpgradeURL(t, env)
		})
	}
}

func assertUpgradeURL(t *testing.T, env entEnvelope) {
	t.Helper()
	if env.UpgradeURL != middleware.UpgradeURL {
		t.Fatalf("upgrade_url = %q, want %q", env.UpgradeURL, middleware.UpgradeURL)
	}
}

// TestEntitlement_ExpiredSubscriptionFallsBackToFree verifies a lapsed paid
// plan no longer grants its features. Without this, an expired Pro subscription
// would keep Pro features forever — an access-escalation bug that only becomes
// consequential once enforcement is on.
func TestEntitlement_ExpiredSubscriptionFallsBackToFree(t *testing.T) {
	h := newEntHarness(t)
	ctx := h.t.Context()

	email := fmt.Sprintf("ent-expired-%s@fintrack.test", uuid.NewString())
	var id uuid.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO users (email, full_name, role) VALUES ($1,'Ent Expired','user') RETURNING id`,
		email).Scan(&id); err != nil {
		h.t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = h.pool.Exec(bg, `DELETE FROM subscriptions WHERE user_id=$1`, id)
		_, _ = h.pool.Exec(bg, `DELETE FROM users WHERE id=$1`, id)
	})
	// A Pro subscription that has expired.
	if _, err := h.pool.Exec(ctx, `
		INSERT INTO subscriptions (user_id, plan_id, status, current_period_start, current_period_end)
		VALUES ($1,$2,'expired', now() - interval '60 days', now() - interval '30 days')`,
		id, proPlanID); err != nil {
		h.t.Fatalf("seed expired subscription: %v", err)
	}

	tok, err := auth.IssueAccessToken(id, "user", email, []byte(entTestSecret), 30*time.Minute)
	if err != nil {
		h.t.Fatalf("issue token: %v", err)
	}

	// reports_export is Pro-only: an expired Pro must not keep it.
	if code, env := h.do(http.MethodGet, "/reports", tok, nil); code != http.StatusForbidden {
		t.Fatalf("expired Pro must fall back to Free, got %d (%s)", code, env.Error)
	}
}

// TestEntitlement_SuperAdminBypasses verifies a super_admin is never gated.
func TestEntitlement_SuperAdminBypasses(t *testing.T) {
	h := newEntHarness(t)
	ctx := h.t.Context()

	email := fmt.Sprintf("ent-admin-%s@fintrack.test", uuid.NewString())
	var id uuid.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO users (email, full_name, role) VALUES ($1,'Ent Admin','super_admin') RETURNING id`,
		email).Scan(&id); err != nil {
		h.t.Fatalf("seed admin: %v", err)
	}
	t.Cleanup(func() { _, _ = h.pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, id) })

	tok, err := auth.IssueAccessToken(id, "super_admin", email, []byte(entTestSecret), 30*time.Minute)
	if err != nil {
		h.t.Fatalf("issue token: %v", err)
	}
	// No subscription at all, and a feature Free does not include.
	if code, env := h.do(http.MethodGet, "/reports", tok, nil); code == http.StatusForbidden {
		t.Fatalf("super_admin must bypass entitlement gates, got 403: %s", env.Error)
	}
}

// ─── helpers ─────────────────────────────────────────────────
