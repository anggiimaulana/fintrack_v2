package middleware

import (
	"context"
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
