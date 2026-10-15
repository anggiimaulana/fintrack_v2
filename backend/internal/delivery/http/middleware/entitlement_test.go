package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/entitlement"
)

type entSource struct {
	features map[string]bool
	limits   map[string]int64
}

func (e entSource) GetActivePlanForUser(context.Context, uuid.UUID) (*billing.Plan, billing.SubscriptionStatus, error) {
	return &billing.Plan{ID: uuid.New(), Code: "pro"}, billing.StatusActive, nil
}
func (e entSource) GetEntitlements(context.Context, uuid.UUID) (map[string]bool, error) {
	return e.features, nil
}
func (e entSource) GetLimits(context.Context, uuid.UUID) (map[string]int64, error) {
	return e.limits, nil
}

func setupEntRouter(role string, src entSource) (*gin.Engine, *EntitlementMiddleware) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Set("role", role)
		c.Next()
	})
	mw := NewEntitlementMiddleware(entitlement.NewResolver(src, nil))
	return r, mw
}

// TestRequireFeature_AllowedPasses verifies enabled features pass.
func TestRequireFeature_AllowedPasses(t *testing.T) {
	r, mw := setupEntRouter("user", entSource{features: map[string]bool{"ai_chat": true}})
	r.GET("/x", mw.RequireFeature("ai_chat"), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestRequireFeature_Denied403 verifies disabled features are blocked.
func TestRequireFeature_Denied403(t *testing.T) {
	r, mw := setupEntRouter("user", entSource{features: map[string]bool{"ai_chat": false}})
	r.GET("/x", mw.RequireFeature("ai_chat"), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	if !contains(w.Body.String(), "feature_not_in_plan") {
		t.Fatalf("expected feature_not_in_plan, got %s", w.Body.String())
	}
}

// TestRequireFeature_SuperAdminBypasses verifies super_admin bypass.
func TestRequireFeature_SuperAdminBypasses(t *testing.T) {
	r, mw := setupEntRouter("super_admin", entSource{features: map[string]bool{"ai_chat": false}})
	r.GET("/x", mw.RequireFeature("ai_chat"), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != 200 {
		t.Fatalf("super_admin must bypass, got %d", w.Code)
	}
}

// TestWithinLimit_UnlimitedPasses verifies -1 always passes.
func TestWithinLimit_UnlimitedPasses(t *testing.T) {
	r, mw := setupEntRouter("user", entSource{limits: map[string]int64{"max_transactions": -1}})
	usage := func(context.Context, uuid.UUID, string) (int64, error) { return 1_000_000, nil }
	r.GET("/x", mw.WithinLimit("max_transactions", usage), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != 200 {
		t.Fatalf("unlimited must pass, got %d", w.Code)
	}
}

// TestWithinLimit_Exceeded402 verifies the 402 response.
func TestWithinLimit_Exceeded402(t *testing.T) {
	r, mw := setupEntRouter("user", entSource{limits: map[string]int64{"max_transactions": 100}})
	usage := func(context.Context, uuid.UUID, string) (int64, error) { return 100, nil }
	r.GET("/x", mw.WithinLimit("max_transactions", usage), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != 402 {
		t.Fatalf("expected 402, got %d", w.Code)
	}
	if !contains(w.Body.String(), "limit_exceeded") {
		t.Fatalf("expected limit_exceeded, got %s", w.Body.String())
	}
}

// TestWithinLimit_AbsentKeyDenies verifies fail-closed for unknown limit keys.
func TestWithinLimit_AbsentKeyDenies(t *testing.T) {
	r, mw := setupEntRouter("user", entSource{limits: map[string]int64{}})
	r.GET("/x", mw.WithinLimit("nope", nil), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != 402 {
		t.Fatalf("absent limit key must deny (402), got %d", w.Code)
	}
}

// TestRequireFeature_IncludesUpgradeURL verifies the upsell payload carries the
// subscription link the frontend points users at.
func TestRequireFeature_IncludesUpgradeURL(t *testing.T) {
	r, mw := setupEntRouter("user", entSource{features: map[string]bool{"ai_chat": false}})
	r.GET("/x", mw.RequireFeature("ai_chat"), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if !contains(w.Body.String(), UpgradeURL) {
		t.Fatalf("expected upgrade_url %q in %s", UpgradeURL, w.Body.String())
	}
}

// TestWithinLimit_IncludesUpgradeURL covers both 402 payloads.
func TestWithinLimit_IncludesUpgradeURL(t *testing.T) {
	for name, limits := range map[string]map[string]int64{
		"exceeded": {"max_transactions": 100},
		"absent":   {},
	} {
		t.Run(name, func(t *testing.T) {
			r, mw := setupEntRouter("user", entSource{limits: limits})
			usage := func(context.Context, uuid.UUID, string) (int64, error) { return 100, nil }
			r.GET("/x", mw.WithinLimit("max_transactions", usage), func(c *gin.Context) { c.String(200, "ok") })

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
			if !contains(w.Body.String(), UpgradeURL) {
				t.Fatalf("expected upgrade_url %q in %s", UpgradeURL, w.Body.String())
			}
		})
	}
}

// TestWithinLimit_UsageErrorIs500 verifies a usage-query failure denies the
// request (500) rather than silently allowing it.
func TestWithinLimit_UsageErrorIs500(t *testing.T) {
	r, mw := setupEntRouter("user", entSource{limits: map[string]int64{"max_transactions": 100}})
	usage := func(context.Context, uuid.UUID, string) (int64, error) {
		return 0, errors.New("db down")
	}
	r.GET("/x", mw.WithinLimit("max_transactions", usage), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != 500 {
		t.Fatalf("usage error must be 500 (never allow), got %d", w.Code)
	}
}

// TestEnforcementDisabled_ValveBypasses verifies ENTITLEMENT_ENFORCEMENT=off
// turns both gates into no-ops (incident rollback valve).
func TestEnforcementDisabled_ValveBypasses(t *testing.T) {
	t.Setenv("ENTITLEMENT_ENFORCEMENT", "off")

	r, mw := setupEntRouter("user", entSource{
		features: map[string]bool{"ai_chat": false},
		limits:   map[string]int64{"max_transactions": 1},
	})
	usage := func(context.Context, uuid.UUID, string) (int64, error) { return 9999, nil }
	r.GET("/feat", mw.RequireFeature("ai_chat"), func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/lim", mw.WithinLimit("max_transactions", usage), func(c *gin.Context) { c.String(200, "ok") })

	for _, path := range []string{"/feat", "/lim"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: valve off must bypass, got %d", path, w.Code)
		}
	}
}

// TestUsageFor_MapsLimitKeys verifies the provider wiring for each limit key.
func TestUsageFor_MapsLimitKeys(t *testing.T) {
	p := &UsageProviders{
		CountTransactions: func(context.Context, uuid.UUID) (int64, error) { return 7, nil },
		CurrentTokens:     func(context.Context, uuid.UUID) (int64, error) { return 42, nil },
		CountVisionScans:  func(context.Context, uuid.UUID) (int64, error) { return 3, nil },
	}
	uid := uuid.New()
	for key, want := range map[string]int64{
		"max_transactions":    7,
		"ai_tokens_per_cycle": 42,
		"ocr_scans_per_cycle": 3,
	} {
		fn := p.UsageFor(context.Background(), key)
		if fn == nil {
			t.Fatalf("%s: expected a resolver", key)
		}
		got, err := fn(context.Background(), uid, key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if got != want {
			t.Fatalf("%s: got %d, want %d", key, got, want)
		}
	}
	if p.UsageFor(context.Background(), "unknown_key") != nil {
		t.Fatal("unknown limit key must have no resolver")
	}
}

// TestTokenPeriodUsage covers the "no period = zero usage" contract: a missing
// period must NOT deny the request, only a real error propagates.
func TestTokenPeriodUsage(t *testing.T) {
	uid := uuid.New()

	// No period -> 0, nil.
	none := TokenPeriodUsage(stubPeriodReader{})
	got, err := none(context.Background(), uid)
	if err != nil || got != 0 {
		t.Fatalf("no period: got (%d, %v), want (0, nil)", got, err)
	}

	// Period present -> used + reserved.
	present := TokenPeriodUsage(stubPeriodReader{period: &billing.TokenPeriod{UsedTokens: 120, ReservedTokens: 30}})
	if got, err = present(context.Background(), uid); err != nil || got != 150 {
		t.Fatalf("period: got (%d, %v), want (150, nil)", got, err)
	}

	// Error propagates.
	broken := TokenPeriodUsage(stubPeriodReader{err: errors.New("boom")})
	if _, err = broken(context.Background(), uid); err == nil {
		t.Fatal("query error must propagate")
	}
}

type stubPeriodReader struct {
	period *billing.TokenPeriod
	err    error
}

func (s stubPeriodReader) GetCurrentPeriod(context.Context, uuid.UUID, int64) (*billing.TokenPeriod, error) {
	return s.period, s.err
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
