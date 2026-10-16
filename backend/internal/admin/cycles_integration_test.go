package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/admin"
	"github.com/fintrack/backend/internal/audit"
	"github.com/fintrack/backend/internal/quota"
)

type noopAudit struct{}

func (n noopAudit) Log(context.Context, audit.Event) {}

func adminTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestAdminCycles_GetAndPutValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Build handler via test harness or direct
	pool := adminTestPool(t)
	h := admin.NewHandler(admin.NewService(pool, noopAudit{}))

	rg := r.Group("/admin")
	admin.RegisterRoutes(rg, h)

	planID := uuid.New()
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO plans (id, code, name, price_monthly_idr, price_yearly_idr)
		VALUES ($1, $2, 'Admin Cycles Test Plan', 39000, 390000)`,
		planID, "cyc-"+planID.String()[:8]); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(t.Context(), `DELETE FROM plan_limit_cycles WHERE plan_id = $1`, planID)
		_, _ = pool.Exec(t.Context(), `DELETE FROM plans WHERE id = $1`, planID)
	})

	// 1. Initial GET -> 200, empty data
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/plans/"+planID.String()+"/cycles", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 2. PUT with missing weekly_limit -> 400
	badCycles := map[string]any{
		"cycles": []map[string]any{
			{"limit_key": "ai_tokens_per_cycle", "cycle_type": "weekly_monthly"},
		},
	}
	b, _ := json.Marshal(badCycles)
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/plans/"+planID.String()+"/cycles", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 on bad cycle, got %d", w.Code)
	}

	// 3. PUT valid cycle -> 200
	wLimit := int64(45000)
	degMode := quota.DegradedModelDowngrade
	model := "gpt-4o-mini"
	goodCycles := map[string]any{
		"cycles": []quota.PlanLimitCycle{
			{
				LimitKey:      "ai_tokens_per_cycle",
				CycleType:     quota.CycleWeeklyMonthly,
				WeeklyLimit:   &wLimit,
				DegradedMode:  &degMode,
				FallbackModel: &model,
			},
		},
	}
	b, _ = json.Marshal(goodCycles)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/admin/plans/"+planID.String()+"/cycles", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 on good cycle, got %d: %s", w.Code, w.Body.String())
	}

	// 4. GET again -> 1 item
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/plans/"+planID.String()+"/cycles", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var res struct {
		Data []quota.PlanLimitCycle `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if len(res.Data) != 1 || res.Data[0].WeeklyLimit == nil || *res.Data[0].WeeklyLimit != 45000 {
		t.Fatalf("expected 1 cycle with weekly_limit 45000, got %+v", res.Data)
	}
}
