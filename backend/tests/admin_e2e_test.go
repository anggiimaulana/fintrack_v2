// Package tests holds end-to-end HTTP tests for the Super Admin dashboard
// (P06). The suite boots the real admin HTTP surface — production middleware
// (auth + RequireSuperAdmin) over the real handlers, service, repositories and
// PostgreSQL — and is gated on TEST_DATABASE_URL so the default
// `go test ./...` stays green without infrastructure:
//
//	TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/fintrack_test \
//	  go test -v -run TestAdminE2E ./tests/...
//
// Only the full delivery router is stubbed out: SetupRouter would drag in 14
// usecases (transactions, SPK, embeddings, …) that the admin surface never
// touches, so the engine is assembled from the same middleware + route group
// the router mounts.
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
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/admin"
	"github.com/fintrack/backend/internal/audit"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/featureflag"
	"github.com/fintrack/backend/internal/infrastructure/auth"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/internal/repository/postgres"
)

const testJWTSecret = "test-secret-at-least-32-characters-long"

// ─── harness ──────────────────────────────────────────────────

type harness struct {
	t    *testing.T
	pool *pgxpool.Pool
	srv  *httptest.Server
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Super Admin E2E tests")
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

	auditSvc := audit.NewService(audit.NewPostgresRepository(pool), audit.Config{
		Buffer: 256, BatchSize: 4, FlushInterval: 10 * time.Millisecond,
	})
	t.Cleanup(auditSvc.Close)

	txManager := postgres.NewTxManager(pool)
	paymentRepo := postgres.NewPaymentRepo(pool)
	// nil activator: subscription activation is covered by the payment package;
	// this suite asserts the admin review transition + race guard.
	manual := payment.NewManualService(paymentRepo, nil, txManager, nil)
	killSwitch := admin.NewKillSwitchStore(pool, nil) // nil pub/sub = single instance
	flags := featureflag.NewService(flagRepo(pool), nil)

	svc := admin.NewService(pool, auditSvc,
		admin.WithManualService(manual),
		admin.WithKillSwitch(killSwitch),
		admin.WithFeatureFlags(flags),
		admin.WithEncryptionKey(bytes.Repeat([]byte{0x42}, 32)),
	)

	engine := gin.New()
	api := engine.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(testJWTSecret))
	sa := api.Group("/admin")
	sa.Use(middleware.RequireSuperAdmin(admin.NewRoleResolver(pool, nil)))
	admin.RegisterRoutes(sa, admin.NewHandler(svc))

	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)

	return &harness{t: t, pool: pool, srv: srv}
}

func flagRepo(pool *pgxpool.Pool) featureflag.Repository {
	return featureflag.NewPostgresRepository(pool)
}

type envelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

// do issues an admin request. An empty token sends no Authorization header.
func (h *harness) do(method, path, token string, body any) (int, envelope) {
	h.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, h.srv.URL+"/api/v1"+path, reader)
	if err != nil {
		h.t.Fatalf("build request: %v", err)
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
		h.t.Fatalf("read body: %v", err)
	}
	var env envelope
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil {
			h.t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
		}
	}
	return resp.StatusCode, env
}

// decode unmarshals an envelope's data payload.
func decode[T any](t *testing.T, env envelope) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(env.Data, &out); err != nil {
		t.Fatalf("decode data %q: %v", env.Data, err)
	}
	return out
}

// seedUser inserts a user with a unique email and returns its id.
func (h *harness) seedUser(role string) (uuid.UUID, string) {
	h.t.Helper()
	email := fmt.Sprintf("e2e-%s@fintrack.test", uuid.NewString())
	var id uuid.UUID
	if err := h.pool.QueryRow(h.t.Context(), `
		INSERT INTO users (email, full_name, role) VALUES ($1, 'E2E User', $2) RETURNING id`,
		email, role).Scan(&id); err != nil {
		h.t.Fatalf("seed user: %v", err)
	}
	return id, email
}

func (h *harness) token(id uuid.UUID, role, email string) string {
	h.t.Helper()
	tok, err := auth.IssueAccessToken(id, role, email, []byte(testJWTSecret), 30*time.Minute)
	if err != nil {
		h.t.Fatalf("issue token: %v", err)
	}
	return tok
}

// superAdmin seeds a promoted user and mints its token.
func (h *harness) superAdmin() (uuid.UUID, string) {
	h.t.Helper()
	id, email := h.seedUser("super_admin")
	return id, h.token(id, "super_admin", email)
}

// ─── 1. authorization ────────────────────────────────────────

func TestAdminE2E_Authorization(t *testing.T) {
	h := newHarness(t)

	if code, _ := h.do(http.MethodGet, "/admin/plans", "", nil); code != http.StatusUnauthorized {
		t.Fatalf("anonymous: got %d, want 401", code)
	}

	userID, email := h.seedUser("user")
	if code, _ := h.do(http.MethodGet, "/admin/plans", h.token(userID, "user", email), nil); code != http.StatusForbidden {
		t.Fatalf("regular user: got %d, want 403", code)
	}

	// Forged privilege: the JWT claims super_admin but the stored role is
	// `user`. The live role re-check (K-14) must fail closed.
	if code, _ := h.do(http.MethodGet, "/admin/plans", h.token(userID, "super_admin", email), nil); code != http.StatusForbidden {
		t.Fatalf("forged super_admin claim: got %d, want 403", code)
	}

	// Demotion while a still-valid super_admin token is in flight.
	adminID, adminEmail := h.superAdmin()
	adminTok := h.token(adminID, "super_admin", adminEmail)
	if code, _ := h.do(http.MethodGet, "/admin/plans", adminTok, nil); code != http.StatusOK {
		t.Fatalf("super admin: got %d, want 200", code)
	}
	if _, err := h.pool.Exec(t.Context(), `UPDATE users SET role = 'user' WHERE id = $1`, adminID); err != nil {
		t.Fatalf("demote: %v", err)
	}
	if code, _ := h.do(http.MethodGet, "/admin/plans", adminTok, nil); code != http.StatusForbidden {
		t.Fatalf("after demotion: got %d, want 403", code)
	}
}

// ─── 2. plan creation + audit trail ──────────────────────────

type planRow struct {
	ID                string           `json:"id"`
	Code              string           `json:"code"`
	FallbackGateway   string           `json:"fallback_gateway"`
	BillingPeriodDays int              `json:"billing_period_days"`
	Entitlements      map[string]bool  `json:"entitlements"`
	Limits            map[string]int64 `json:"limits"`
}

type auditRow struct {
	Action   string         `json:"action"`
	Entity   string         `json:"entity"`
	EntityID *string        `json:"entity_id"`
	ActorID  *string        `json:"actor_id"`
	After    map[string]any `json:"after"`
}

func TestAdminE2E_PlanCreateAndAuditTrail(t *testing.T) {
	h := newHarness(t)
	adminID, tok := h.superAdmin()

	code := "e2e-" + uuid.NewString()[:8]
	body := map[string]any{
		"code": code, "name": "E2E Pro",
		"price_monthly_idr": 49000, "price_yearly_idr": 490000,
		"billing_period_days": 45, "is_active": true, "is_public": true,
		"fallback_gateway": "manual",
		"entitlements":     map[string]bool{"ai_chat": true, "export_pdf": false},
		"limits":           map[string]int64{"max_transactions": 5000},
	}
	t.Cleanup(func() {
		_, _ = h.pool.Exec(context.Background(), `DELETE FROM plans WHERE code = $1`, code)
	})

	status, env := h.do(http.MethodPost, "/admin/plans", tok, body)
	if status != http.StatusCreated {
		t.Fatalf("create plan: got %d (%s)", status, env.Message)
	}
	planID := decode[struct {
		ID string `json:"id"`
	}](t, env).ID

	// Duplicate code is rejected.
	if status, _ := h.do(http.MethodPost, "/admin/plans", tok, body); status != http.StatusConflict {
		t.Fatalf("duplicate plan code: got %d, want 409", status)
	}

	// The plan (with its entitlement/limit matrix) is listed.
	status, env = h.do(http.MethodGet, "/admin/plans", tok, nil)
	if status != http.StatusOK {
		t.Fatalf("list plans: got %d", status)
	}
	plan, ok := findPlan(decode[[]planRow](t, env), planID)
	if !ok {
		t.Fatalf("plan %s missing from listing", planID)
	}
	if plan.Code != code || plan.FallbackGateway != "manual" || plan.BillingPeriodDays != 45 {
		t.Fatalf("plan fields not persisted: %+v", plan)
	}
	if !plan.Entitlements["ai_chat"] || plan.Entitlements["export_pdf"] {
		t.Fatalf("entitlements not persisted: %v", plan.Entitlements)
	}
	if plan.Limits["max_transactions"] != 5000 {
		t.Fatalf("limits not persisted: %v", plan.Limits)
	}

	// Audit trail: the async writer flushes within its 10ms interval; poll.
	logs := h.waitForAuditLog(tok, "entity=plan&entity_id="+planID)
	if logs[0].Action != "create" || logs[0].Entity != "plan" {
		t.Fatalf("unexpected audit row: %+v", logs[0])
	}
	if logs[0].ActorID == nil || *logs[0].ActorID != adminID.String() {
		t.Fatalf("audit actor = %v, want %s", logs[0].ActorID, adminID)
	}
	if got := logs[0].After["code"]; got != code {
		t.Fatalf("audit after.code = %v, want %s", got, code)
	}
}

func findPlan(plans []planRow, id string) (planRow, bool) {
	for _, p := range plans {
		if p.ID == id {
			return p, true
		}
	}
	return planRow{}, false
}

func (h *harness) waitForAuditLog(tok, query string) []auditRow {
	h.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status, env := h.do(http.MethodGet, "/admin/audit-logs?per_page=10&"+query, tok, nil)
		if status != http.StatusOK {
			h.t.Fatalf("audit-logs: got %d (%s)", status, env.Message)
		}
		if rows := decode[[]auditRow](h.t, env); len(rows) > 0 {
			return rows
		}
		time.Sleep(25 * time.Millisecond)
	}
	h.t.Fatalf("audit log for %q not persisted in time", query)
	return nil
}

// ─── 3. payment review race ──────────────────────────────────

// TestAdminE2E_PaymentReviewRace fires concurrent approvals at the same payment
// and asserts exactly one wins. A single round can pass by luck (the pre-fix
// code admitted several 200s only when transactions actually overlapped), so
// the burst is repeated over independent payments: any unguarded window shows
// up as a double-approval in at least one round.
func TestAdminE2E_PaymentReviewRace(t *testing.T) {
	h := newHarness(t)
	_, tok := h.superAdmin()
	userID, _ := h.seedUser("user")

	const (
		rounds  = 3
		workers = 8
	)
	for round := range rounds {
		paymentID, invoiceID := h.seedPendingPayment(userID)

		codes := make([]int, workers)
		var wg sync.WaitGroup
		for i := range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				codes[i], _ = h.do(http.MethodPost, "/admin/payments/"+paymentID.String()+"/approve", tok, nil)
			}()
		}
		wg.Wait()

		ok, conflict := 0, 0
		for _, c := range codes {
			switch c {
			case http.StatusOK:
				ok++
			case http.StatusConflict:
				conflict++
			default:
				t.Fatalf("round %d: unexpected status %d in %v", round, c, codes)
			}
		}
		if ok != 1 || conflict != workers-1 {
			t.Fatalf("round %d: race not mitigated: %v (want exactly 1x200, %dx409)", round, codes, workers-1)
		}

		var status string
		if err := h.pool.QueryRow(t.Context(), `SELECT status FROM payments WHERE id = $1`, paymentID).Scan(&status); err != nil {
			t.Fatalf("round %d: read payment: %v", round, err)
		}
		if status != "settled" {
			t.Fatalf("round %d: payment status = %s, want settled", round, status)
		}

		var events int
		if err := h.pool.QueryRow(t.Context(),
			`SELECT count(*) FROM payment_events WHERE payment_id = $1 AND event = 'approve'`, paymentID).Scan(&events); err != nil {
			t.Fatalf("round %d: count payment events: %v", round, err)
		}
		if events != 1 {
			t.Fatalf("round %d: approve events = %d, want 1", round, events)
		}

		h.cleanupPayment(paymentID, invoiceID)
	}
}

// seedPendingPayment creates an issued invoice plus a manual payment awaiting
// review, registered for cleanup.
func (h *harness) seedPendingPayment(userID uuid.UUID) (paymentID, invoiceID uuid.UUID) {
	h.t.Helper()

	if err := h.pool.QueryRow(h.t.Context(), `
		INSERT INTO invoices (invoice_number, user_id, subtotal_idr, total_idr, status, notes)
		VALUES ($1, $2, 49000, 49000, 'issued', 'e2e-plan') RETURNING id`,
		"INV/E2E/"+uuid.NewString()[:8], userID).Scan(&invoiceID); err != nil {
		h.t.Fatalf("seed invoice: %v", err)
	}
	if err := h.pool.QueryRow(h.t.Context(), `
		INSERT INTO payments (invoice_id, user_id, gateway, amount_idr, status, idempotency_key)
		VALUES ($1, $2, 'manual', 49000, 'pending_review', $3) RETURNING id`,
		invoiceID, userID, "e2e-"+uuid.NewString()).Scan(&paymentID); err != nil {
		h.t.Fatalf("seed payment: %v", err)
	}
	h.t.Cleanup(func() { h.cleanupPayment(paymentID, invoiceID) })
	return paymentID, invoiceID
}

func (h *harness) cleanupPayment(paymentID, invoiceID uuid.UUID) {
	ctx := context.Background()
	_, _ = h.pool.Exec(ctx, `DELETE FROM payments WHERE id = $1`, paymentID)
	_, _ = h.pool.Exec(ctx, `DELETE FROM invoices WHERE id = $1`, invoiceID)
}

// ─── 4. idempotent token adjustment ──────────────────────────

type adjustResult struct {
	LedgerID string `json:"ledger_id"`
	Quota    int64  `json:"quota_tokens"`
	Replayed bool   `json:"replayed"`
}

func TestAdminE2E_TokenAdjustIdempotent(t *testing.T) {
	h := newHarness(t)
	_, tok := h.superAdmin()

	userID, _ := h.seedUser("user")
	if _, err := h.pool.Exec(t.Context(), `
		INSERT INTO token_periods (user_id, period_start, period_end, quota_tokens, used_tokens)
		VALUES ($1, now() - interval '1 day', now() + interval '29 days', 1000, 100)`, userID); err != nil {
		t.Fatalf("seed token period: %v", err)
	}

	key := uuid.NewString()
	body := map[string]any{"user_id": userID.String(), "amount": 250, "reason": "e2e top-up", "idempotency_key": key}

	status, env := h.do(http.MethodPost, "/admin/tokens/adjust", tok, body)
	if status != http.StatusOK {
		t.Fatalf("adjust: got %d (%s)", status, env.Message)
	}
	first := decode[adjustResult](t, env)
	if first.Replayed || first.Quota != 1250 {
		t.Fatalf("first adjust = %+v, want replayed=false quota=1250", first)
	}

	// Replay of the same key must not mutate anything.
	status, env = h.do(http.MethodPost, "/admin/tokens/adjust", tok, body)
	if status != http.StatusOK {
		t.Fatalf("replay: got %d (%s)", status, env.Message)
	}
	second := decode[adjustResult](t, env)
	if !second.Replayed || second.Quota != 1250 || second.LedgerID != first.LedgerID {
		t.Fatalf("replay = %+v, want replayed=true quota=1250 same ledger", second)
	}

	var periodQuota, rows int64
	if err := h.pool.QueryRow(t.Context(),
		`SELECT quota_tokens FROM token_periods WHERE user_id = $1`, userID).Scan(&periodQuota); err != nil {
		t.Fatalf("read token period: %v", err)
	}
	if periodQuota != 1250 {
		t.Fatalf("period quota = %d, want 1250 (replay must not double-apply)", periodQuota)
	}
	if err := h.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM token_ledger WHERE idempotency_key = $1`, key).Scan(&rows); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if rows != 1 {
		t.Fatalf("ledger rows for key = %d, want 1", rows)
	}
}

// ─── 5. AI gateway kill switch ───────────────────────────────

func TestAdminE2E_KillSwitchActivation(t *testing.T) {
	h := newHarness(t)
	_, tok := h.superAdmin()

	t.Cleanup(func() {
		// Leave the persisted matrix in the disabled state.
		_, _ = h.pool.Exec(context.Background(), `
			UPDATE settings SET value = '{}'::jsonb WHERE key = 'ai_kill_switch'`)
	})

	status, env := h.do(http.MethodPost, "/admin/ai-gateway/kill-switch", tok,
		map[string]any{"target": "chat", "enabled": true})
	if status != http.StatusOK {
		t.Fatalf("enable chat: got %d (%s)", status, env.Message)
	}
	matrix := decode[map[string]bool](t, env)
	if !matrix["chat"] || matrix["global"] {
		t.Fatalf("matrix after enable = %v", matrix)
	}

	// Persisted: a fresh read of the settings row sees the same state.
	var raw []byte
	if err := h.pool.QueryRow(t.Context(), `SELECT value FROM settings WHERE key = 'ai_kill_switch'`).Scan(&raw); err != nil {
		t.Fatalf("read persisted switch: %v", err)
	}
	var persisted map[string]bool
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("decode persisted switch: %v", err)
	}
	if !persisted["chat"] {
		t.Fatalf("persisted switch = %s, want chat=true", raw)
	}

	// GET reflects the stored matrix, and unknown targets are refused.
	if status, env = h.do(http.MethodGet, "/admin/ai-gateway/kill-switch", tok, nil); status != http.StatusOK {
		t.Fatalf("get kill switch: got %d", status)
	}
	if m := decode[map[string]bool](t, env); !m["chat"] {
		t.Fatalf("GET matrix = %v, want chat=true", m)
	}
	if status, _ := h.do(http.MethodPost, "/admin/ai-gateway/kill-switch", tok,
		map[string]any{"target": "bogus", "enabled": true}); status != http.StatusBadRequest {
		t.Fatalf("invalid target: got %d, want 400", status)
	}

	if status, _ = h.do(http.MethodPost, "/admin/ai-gateway/kill-switch", tok,
		map[string]any{"target": "chat", "enabled": false}); status != http.StatusOK {
		t.Fatalf("disable chat: got %d", status)
	}
}

// ─── 6. settings + feature flags ─────────────────────────────

func TestAdminE2E_SettingsAndFeatureFlags(t *testing.T) {
	h := newHarness(t)
	adminID, tok := h.superAdmin()

	var original []byte
	if err := h.pool.QueryRow(t.Context(), `SELECT value FROM settings WHERE key = 'tax_rate_percent'`).Scan(&original); err != nil {
		t.Fatalf("read tax rate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.pool.Exec(context.Background(),
			`UPDATE settings SET value = $2 WHERE key = $1`, "tax_rate_percent", original)
	})

	status, env := h.do(http.MethodGet, "/admin/settings", tok, nil)
	if status != http.StatusOK {
		t.Fatalf("list settings: got %d", status)
	}
	if !hasSettingKey(decode[[]struct {
		Key string `json:"key"`
	}](t, env), "grace_period_hours") {
		t.Fatal("grace_period_hours missing from settings listing")
	}

	if status, env = h.do(http.MethodPut, "/admin/settings", tok, map[string]any{
		"settings": map[string]any{"tax_rate_percent": 11},
	}); status != http.StatusOK {
		t.Fatalf("update settings: got %d (%s)", status, env.Message)
	}
	var updatedBy *string
	var newValue int
	if err := h.pool.QueryRow(t.Context(),
		`SELECT updated_by, (value #>> '{}')::int FROM settings WHERE key = 'tax_rate_percent'`).
		Scan(&updatedBy, &newValue); err != nil {
		t.Fatalf("read updated setting: %v", err)
	}
	if newValue != 11 {
		t.Fatalf("tax_rate_percent = %d, want 11", newValue)
	}
	if updatedBy == nil || *updatedBy != adminID.String() {
		t.Fatalf("updated_by = %v, want %s", updatedBy, adminID)
	}

	// Non-allowlisted keys are refused (the kill switch has its own endpoint).
	if status, env = h.do(http.MethodPut, "/admin/settings", tok, map[string]any{
		"settings": map[string]any{"ai_kill_switch": "{}"},
	}); status != http.StatusBadRequest || env.Error != "setting_not_writable" {
		t.Fatalf("non-writable key: got %d/%s, want 400/setting_not_writable", status, env.Error)
	}

	// Feature flags: upsert, range validation, listing.
	flagKey := "e2e-flag-" + uuid.NewString()[:8]
	t.Cleanup(func() {
		_, _ = h.pool.Exec(context.Background(), `DELETE FROM feature_flags WHERE key = $1`, flagKey)
	})
	if status, env = h.do(http.MethodPut, "/admin/feature-flags", tok, map[string]any{
		"key": flagKey, "name": "E2E Flag", "enabled": true, "rollout": 50,
	}); status != http.StatusOK {
		t.Fatalf("upsert flag: got %d (%s)", status, env.Message)
	}
	flag := decode[struct {
		Key       string `json:"key"`
		IsEnabled bool   `json:"is_enabled"`
		Rollout   int    `json:"rollout"`
	}](t, env)
	if !flag.IsEnabled || flag.Rollout != 50 || flag.Key != flagKey {
		t.Fatalf("flag = %+v", flag)
	}
	if status, _ = h.do(http.MethodPut, "/admin/feature-flags", tok, map[string]any{
		"key": flagKey, "enabled": true, "rollout": 150,
	}); status != http.StatusBadRequest {
		t.Fatalf("out-of-range rollout: got %d, want 400", status)
	}
	if status, env = h.do(http.MethodGet, "/admin/feature-flags", tok, nil); status != http.StatusOK {
		t.Fatalf("list flags: got %d", status)
	}
	if !hasFlagKey(decode[[]struct {
		Key string `json:"key"`
	}](t, env), flagKey) {
		t.Fatalf("flag %s missing from listing", flagKey)
	}
}

func hasSettingKey(rows []struct {
	Key string `json:"key"`
}, want string) bool {
	for _, r := range rows {
		if r.Key == want {
			return true
		}
	}
	return false
}

func hasFlagKey(rows []struct {
	Key string `json:"key"`
}, want string) bool {
	for _, r := range rows {
		if r.Key == want {
			return true
		}
	}
	return false
}

// TestAdminE2E_PricingRuleCostBasis verifies the pricing-rule CRUD round-trips the
// new cost-basis fields (price_unit + provider_cost_per_1k_idr) and that the
// active rule exposes them, so the Super Admin UI can enter "Rp3.000 / 10.000
// token" and see the margin.
func TestAdminE2E_PricingRuleCostBasis(t *testing.T) {
	h := newHarness(t)
	_, tok := h.superAdmin()

	ruleName := fmt.Sprintf("e2e-basis-%s", uuid.NewString()[:8])
	status, env := h.do(http.MethodPost, "/admin/pricing-rules", tok, map[string]any{
		"rule_name":                ruleName,
		"base_per_1k_idr":          3000,
		"price_unit":               "per_10k",
		"provider_cost_per_1k_idr": 200,
		"markup_multiplier":        1.0,
		"markup_percent":           0,
		"modality_multipliers":     map[string]float64{"chat": 1.0},
	})
	if status != http.StatusCreated {
		t.Fatalf("create pricing rule: got %d (%s)", status, env.Message)
	}
	created := decode[struct {
		ID string `json:"id"`
	}](t, env)
	t.Cleanup(func() {
		_, _ = h.pool.Exec(context.Background(), `DELETE FROM pricing_rules WHERE rule_name = $1`, ruleName)
	})

	// The listing must carry the new fields back.
	status, env = h.do(http.MethodGet, "/admin/pricing-rules", tok, nil)
	if status != http.StatusOK {
		t.Fatalf("list pricing rules: got %d", status)
	}
	rules := decode[[]struct {
		ID                   string `json:"id"`
		RuleName             string `json:"rule_name"`
		PriceUnit            string `json:"price_unit"`
		ProviderCostPer1kIDR int64  `json:"provider_cost_per_1k_idr"`
	}](t, env)
	var found *struct {
		ID                   string `json:"id"`
		RuleName             string `json:"rule_name"`
		PriceUnit            string `json:"price_unit"`
		ProviderCostPer1kIDR int64  `json:"provider_cost_per_1k_idr"`
	}
	for i := range rules {
		if rules[i].ID == created.ID {
			found = &rules[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("created rule %s not in listing", created.ID)
	}
	if found.PriceUnit != "per_10k" {
		t.Fatalf("price_unit = %q, want per_10k", found.PriceUnit)
	}
	if found.ProviderCostPer1kIDR != 200 {
		t.Fatalf("provider_cost_per_1k_idr = %d, want 200", found.ProviderCostPer1kIDR)
	}

	// Update must persist a new unit + cost.
	status, env = h.do(http.MethodPut, "/admin/pricing-rules/"+created.ID, tok, map[string]any{
		"price_unit":               "per_1k",
		"provider_cost_per_1k_idr": 250,
	})
	if status != http.StatusOK {
		t.Fatalf("update pricing rule: got %d (%s)", status, env.Message)
	}
	var unit string
	var cost int64
	if err := h.pool.QueryRow(t.Context(),
		`SELECT price_unit, provider_cost_per_1k_idr FROM pricing_rules WHERE id = $1`, created.ID).
		Scan(&unit, &cost); err != nil {
		t.Fatalf("read pricing rule: %v", err)
	}
	if unit != "per_1k" || cost != 250 {
		t.Fatalf("after update: unit=%q cost=%d, want per_1k/250", unit, cost)
	}

	// An invalid unit must be rejected.
	if status, _ = h.do(http.MethodPut, "/admin/pricing-rules/"+created.ID, tok, map[string]any{
		"price_unit": "per_100",
	}); status != http.StatusBadRequest {
		t.Fatalf("invalid price_unit: got %d, want 400", status)
	}
}

// TestAdminE2E_EstimateSettingsWritable verifies the AI reserve estimates (P3)
// are exposed in the settings listing and can be updated from the Super Admin
// portal, so an operator can tune the token hold without SQL.
func TestAdminE2E_EstimateSettingsWritable(t *testing.T) {
	h := newHarness(t)
	adminID, tok := h.superAdmin()

	// The setting may or may not be seeded; capture whatever exists so the test
	// restores it, and upsert a known row otherwise (the PUT below upserts too).
	var original []byte
	haveOriginal := true
	if err := h.pool.QueryRow(t.Context(), `SELECT value FROM settings WHERE key = 'ai_estimate_chat'`).Scan(&original); err != nil {
		haveOriginal = false
	}
	t.Cleanup(func() {
		if haveOriginal {
			_, _ = h.pool.Exec(context.Background(),
				`UPDATE settings SET value = $2 WHERE key = $1`, "ai_estimate_chat", original)
		} else {
			_, _ = h.pool.Exec(context.Background(),
				`DELETE FROM settings WHERE key = $1`, "ai_estimate_chat")
		}
	})

	// The PUT upserts the row; then the listing must surface it, proving the
	// Super Admin UI can both write and read back the estimate.
	status, env := h.do(http.MethodPut, "/admin/settings", tok, map[string]any{
		"settings": map[string]any{"ai_estimate_chat": 1234},
	})
	if status != http.StatusOK {
		t.Fatalf("update ai_estimate_chat: got %d (%s)", status, env.Message)
	}

	status, env = h.do(http.MethodGet, "/admin/settings", tok, nil)
	if status != http.StatusOK {
		t.Fatalf("list settings: got %d", status)
	}
	if !hasSettingKey(decode[[]struct {
		Key string `json:"key"`
	}](t, env), "ai_estimate_chat") {
		t.Fatal("ai_estimate_chat missing from settings listing")
	}
	var updatedBy *string
	var newValue int
	if err := h.pool.QueryRow(t.Context(),
		`SELECT updated_by, (value #>> '{}')::int FROM settings WHERE key = 'ai_estimate_chat'`).
		Scan(&updatedBy, &newValue); err != nil {
		t.Fatalf("read updated setting: %v", err)
	}
	if newValue != 1234 {
		t.Fatalf("ai_estimate_chat = %d, want 1234", newValue)
	}
	if updatedBy == nil || *updatedBy != adminID.String() {
		t.Fatalf("updated_by = %v, want %s", updatedBy, adminID)
	}

	// Out-of-range values are rejected by the writable-spec (min 1).
	if status, _ = h.do(http.MethodPut, "/admin/settings", tok, map[string]any{
		"settings": map[string]any{"ai_estimate_chat": 0},
	}); status != http.StatusBadRequest {
		t.Fatalf("ai_estimate_chat=0: got %d, want 400", status)
	}
}

// TestAdminE2E_BreakerAndDunningSettingsWritable verifies the AI gateway breaker
// tuning and dunning reminder offsets are exposed in the settings listing and can
// be updated from the Super Admin portal (no redeploy needed).
func TestAdminE2E_BreakerAndDunningSettingsWritable(t *testing.T) {
	h := newHarness(t)
	_, tok := h.superAdmin()

	// Capture + restore any pre-existing values.
	keys := []string{
		"dunning_hours_urgent", "dunning_hours_soon",
		"breaker_failure_threshold", "breaker_cooldown_seconds", "breaker_half_open_probes",
		"route_breaker_threshold", "route_breaker_cooldown_secs",
	}
	type saved struct {
		raw    []byte
		exists bool
	}
	orig := map[string]saved{}
	for _, k := range keys {
		var raw []byte
		err := h.pool.QueryRow(t.Context(), `SELECT value FROM settings WHERE key = $1`, k).Scan(&raw)
		orig[k] = saved{raw: raw, exists: err == nil}
	}
	t.Cleanup(func() {
		for _, k := range keys {
			if orig[k].exists {
				_, _ = h.pool.Exec(context.Background(),
					`UPDATE settings SET value = $2 WHERE key = $1`, k, orig[k].raw)
			} else {
				_, _ = h.pool.Exec(context.Background(), `DELETE FROM settings WHERE key = $1`, k)
			}
		}
	})

	// Upsert via the admin API.
	status, env := h.do(http.MethodPut, "/admin/settings", tok, map[string]any{
		"settings": map[string]any{
			"dunning_hours_urgent":        12,
			"dunning_hours_soon":          48,
			"breaker_failure_threshold":   7,
			"breaker_cooldown_seconds":    45,
			"breaker_half_open_probes":    2,
			"route_breaker_threshold":     3,
			"route_breaker_cooldown_secs": 90,
		},
	})
	if status != http.StatusOK {
		t.Fatalf("update settings: got %d (%s)", status, env.Message)
	}

	// They must appear in the listing.
	status, env = h.do(http.MethodGet, "/admin/settings", tok, nil)
	if status != http.StatusOK {
		t.Fatalf("list settings: got %d", status)
	}
	rows := decode[[]struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}](t, env)
	got := map[string]bool{}
	for _, r := range rows {
		got[r.Key] = true
	}
	for _, k := range keys {
		if !got[k] {
			t.Errorf("setting %s missing from listing", k)
		}
	}

	// Values must round-trip as integers.
	var v int
	if err := h.pool.QueryRow(t.Context(),
		`SELECT (value #>> '{}')::int FROM settings WHERE key = 'breaker_failure_threshold'`).Scan(&v); err != nil {
		t.Fatalf("read breaker_failure_threshold: %v", err)
	}
	if v != 7 {
		t.Fatalf("breaker_failure_threshold = %d, want 7", v)
	}

	// Out-of-range values are rejected (min 1).
	if status, _ = h.do(http.MethodPut, "/admin/settings", tok, map[string]any{
		"settings": map[string]any{"breaker_failure_threshold": 0},
	}); status != http.StatusBadRequest {
		t.Fatalf("breaker_failure_threshold=0: got %d, want 400", status)
	}
}

// TestAdminE2E_MeteringSettingsWritable verifies the AI token metering heuristics
// (local token estimation when provider omits usage) are exposed in the settings
// listing and can be retuned by a Super Admin at runtime without a redeploy.
func TestAdminE2E_MeteringSettingsWritable(t *testing.T) {
	h := newHarness(t)
	_, tok := h.superAdmin()

	keys := []string{
		"metering_chars_per_token",
		"metering_safety_buffer_pct",
		"metering_image_tokens",
		"metering_audio_tokens_per_sec",
	}
	type saved struct {
		raw    []byte
		exists bool
	}
	orig := map[string]saved{}
	for _, k := range keys {
		var raw []byte
		err := h.pool.QueryRow(t.Context(), `SELECT value FROM settings WHERE key = $1`, k).Scan(&raw)
		orig[k] = saved{raw: raw, exists: err == nil}
	}
	t.Cleanup(func() {
		for _, k := range keys {
			if orig[k].exists {
				_, _ = h.pool.Exec(context.Background(),
					`UPDATE settings SET value = $2 WHERE key = $1`, k, orig[k].raw)
			} else {
				_, _ = h.pool.Exec(context.Background(), `DELETE FROM settings WHERE key = $1`, k)
			}
		}
	})

	// Update via admin API
	status, env := h.do(http.MethodPut, "/admin/settings", tok, map[string]any{
		"settings": map[string]any{
			"metering_chars_per_token":      6,
			"metering_safety_buffer_pct":    20,
			"metering_image_tokens":         120,
			"metering_audio_tokens_per_sec": 35,
		},
	})
	if status != http.StatusOK {
		t.Fatalf("update settings: got %d (%s)", status, env.Message)
	}

	// Must appear in GET /admin/settings
	status, env = h.do(http.MethodGet, "/admin/settings", tok, nil)
	if status != http.StatusOK {
		t.Fatalf("list settings: got %d", status)
	}
	rows := decode[[]struct {
		Key string `json:"key"`
	}](t, env)
	got := map[string]bool{}
	for _, r := range rows {
		got[r.Key] = true
	}
	for _, k := range keys {
		if !got[k] {
			t.Errorf("setting %s missing from listing", k)
		}
	}

	// Verify value stored in database
	var cpt int
	if err := h.pool.QueryRow(t.Context(),
		`SELECT (value #>> '{}')::int FROM settings WHERE key = 'metering_chars_per_token'`).Scan(&cpt); err != nil {
		t.Fatalf("read metering_chars_per_token: %v", err)
	}
	if cpt != 6 {
		t.Fatalf("metering_chars_per_token = %d, want 6", cpt)
	}

	// Out-of-range value rejected (min 1)
	if status, _ = h.do(http.MethodPut, "/admin/settings", tok, map[string]any{
		"settings": map[string]any{"metering_chars_per_token": 0},
	}); status != http.StatusBadRequest {
		t.Fatalf("metering_chars_per_token=0: got %d, want 400", status)
	}
}
