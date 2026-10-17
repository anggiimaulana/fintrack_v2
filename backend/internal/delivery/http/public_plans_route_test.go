package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/payment"
)

// stubPlanRepo implements billing.PlanRepository for the route test. It mimics
// the real repository contract: return only active plans, ordered by
// sort_order. The embedded interface satisfies the remaining methods and
// panics loudly if an unexpected one is reached.
type stubPlanRepo struct {
	billing.PlanRepository
	plans []*billing.Plan
}

func (s *stubPlanRepo) ListActivePlans(context.Context) ([]*billing.Plan, error) {
	out := make([]*billing.Plan, 0, len(s.plans))
	for _, p := range s.plans {
		if p.IsActive {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].SortOrder < out[j].SortOrder })
	return out, nil
}

// stubEntitlements is the PlanEntitlementSource backing the catalog.
type stubEntitlements struct{}

func (stubEntitlements) GetEntitlements(context.Context, uuid.UUID) (map[string]bool, error) {
	return map[string]bool{"ai_chat": true}, nil
}

func (stubEntitlements) GetLimits(context.Context, uuid.UUID) (map[string]int64, error) {
	return map[string]int64{"max_transactions": 100, "max_categories": 30}, nil
}

// publicPlansEngine boots the real router with only the payment handler wired
// (every other use case nil), which is enough to test the public catalog route
// and the protected-route regression without a database.
func publicPlansEngine(t *testing.T, paymentH *handler.PaymentHandler) *gin.Engine {
	t.Helper()
	return SetupRouter(
		RouterConfig{JWTSecret: "test-secret"},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		paymentH, nil, nil,
	)
}

// seededPaymentHandler builds a payment handler over a fake catalog whose plans
// are deliberately unordered and include non-public and inactive entries.
func seededPaymentHandler() *handler.PaymentHandler {
	repo := &stubPlanRepo{plans: []*billing.Plan{
		// Unordered on purpose; the service is expected to surface sort_order.
		{ID: uuid.New(), Code: "pro", Name: "Pro", PriceMonthlyIDR: 39000, IsActive: true, IsPublic: true, SortOrder: 2},
		{ID: uuid.New(), Code: "legacy", Name: "Legacy", IsActive: true, IsPublic: false, SortOrder: 3},
		{ID: uuid.New(), Code: "archived", Name: "Archived", IsActive: false, IsPublic: true, SortOrder: 4},
		{ID: uuid.New(), Code: "free", Name: "Free", IsActive: true, IsPublic: true, SortOrder: 1},
	}}
	catalog := payment.NewCatalogService(repo, stubEntitlements{}, []payment.LimitMeta{
		{Key: "max_transactions", Label: "Transaksi", Unit: "transaksi", Enforced: true},
		{Key: "max_categories", Label: "Kategori", Unit: "kategori", Enforced: false},
	}, nil)
	return handler.NewPaymentHandler(nil).WithCatalog(catalog)
}

type publicPlansBody struct {
	Success bool `json:"success"`
	Data    []struct {
		Code      string                       `json:"code"`
		IsPublic  bool                         `json:"is_public"`
		LimitMeta map[string]payment.LimitMeta `json:"limit_meta"`
	} `json:"data"`
}

func decodePlans(t *testing.T, w *httptest.ResponseRecorder) publicPlansBody {
	t.Helper()
	var body publicPlansBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return body
}

// TestPublicPlans_AnonymousCanRead locks the new public catalog route: an
// unauthenticated request must succeed, return only active+public plans in
// sort_order, and carry the `enforced` flag the pricing UI relies on.
func TestPublicPlans_AnonymousCanRead(t *testing.T) {
	engine := publicPlansEngine(t, seededPaymentHandler())

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/public/plans", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("GET /public/plans: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	body := decodePlans(t, w)
	if !body.Success {
		t.Fatal("success = false, want true")
	}
	if len(body.Data) != 2 {
		t.Fatalf("data length = %d, want 2 (only active+public plans)", len(body.Data))
	}
	// Ordering follows sort_order (free=1 before pro=2), even though the repo
	// returned them shuffled.
	if body.Data[0].Code != "free" || body.Data[1].Code != "pro" {
		t.Fatalf("plan order = [%s, %s], want [free, pro]", body.Data[0].Code, body.Data[1].Code)
	}
	for _, p := range body.Data {
		if !p.IsPublic {
			t.Fatalf("plan %q returned with is_public=false", p.Code)
		}
	}
	// The enforced flag must survive so the UI can hide unenforced limits.
	if meta, ok := body.Data[0].LimitMeta["max_categories"]; !ok || meta.Enforced {
		t.Fatalf("max_categories limit_meta = %+v, want present with enforced=false", meta)
	}
}

// TestPublicPlans_NilCatalogReturnsEmpty covers a misconfigured backend: the
// handler must still answer 200 with an empty array rather than 500.
func TestPublicPlans_NilCatalogReturnsEmpty(t *testing.T) {
	engine := publicPlansEngine(t, handler.NewPaymentHandler(nil))

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/public/plans", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("GET /public/plans (nil catalog): got %d, want 200", w.Code)
	}
	body := decodePlans(t, w)
	if !body.Success {
		t.Fatal("success = false, want true")
	}
	if len(body.Data) != 0 {
		t.Fatalf("data length = %d, want 0", len(body.Data))
	}
}

// TestProtectedPlans_AnonymousIsUnauthorized guards against a regression where
// the authenticated catalog route accidentally becomes public.
func TestProtectedPlans_AnonymousIsUnauthorized(t *testing.T) {
	engine := publicPlansEngine(t, seededPaymentHandler())

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/plans", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /plans without JWT: got %d, want 401 (body: %s)", w.Code, w.Body.String())
	}
}
