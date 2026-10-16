package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/quota"
)

type fakeQuotaCycleRepo struct {
	cycles       map[string][]quota.PlanLimitCycle
	counters     map[uuid.UUID][]quota.UsageCounter
	weeklyAnchor *time.Time
}

func (f *fakeQuotaCycleRepo) GetPlanCycles(_ context.Context, planID uuid.UUID) ([]quota.PlanLimitCycle, error) {
	return f.cycles[planID.String()], nil
}

func (f *fakeQuotaCycleRepo) UpsertPlanCycles(_ context.Context, planID uuid.UUID, cycles []quota.PlanLimitCycle) error {
	f.cycles[planID.String()] = cycles
	return nil
}

func (f *fakeQuotaCycleRepo) GetWarningThreshold(_ context.Context, _ uuid.UUID) (int, error) {
	return 80, nil
}

func (f *fakeQuotaCycleRepo) GetCurrentCounters(_ context.Context, userID uuid.UUID, _ time.Time) ([]quota.UsageCounter, error) {
	return f.counters[userID], nil
}
func (f *fakeQuotaCycleRepo) GetOrCreateCounter(_ context.Context, userID uuid.UUID, limitKey, cycleType string, start, end time.Time) (*quota.UsageCounter, error) {
	return &quota.UsageCounter{
		ID:         uuid.New(),
		UserID:     userID,
		LimitKey:   limitKey,
		CycleType:  cycleType,
		CycleStart: start,
		CycleEnd:   end,
	}, nil
}

func (f *fakeQuotaCycleRepo) GetSubscriptionWeeklyAnchor(_ context.Context, _ uuid.UUID) (*time.Time, error) {
	return f.weeklyAnchor, nil
}

func (f *fakeQuotaCycleRepo) GetUserPlanID(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func setupQuotaRouter(repo *fakeQuotaCycleRepo, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	h := handler.NewQuotaHandler(repo, nil, nil)
	r.GET("/api/v1/admin/plans/:id/cycles", h.GetPlanCycles)
	r.PUT("/api/v1/admin/plans/:id/cycles", h.UpsertPlanCycles)
	r.GET("/api/v1/me/quota", h.GetUserQuotaSnapshot)
	return r
}

func TestQuotaHandler_GetAndUpsertPlanCycles(t *testing.T) {
	planID := uuid.New()
	repo := &fakeQuotaCycleRepo{cycles: make(map[string][]quota.PlanLimitCycle)}
	r := setupQuotaRouter(repo, uuid.New())

	// 1. Initial GET -> empty
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/plans/"+planID.String()+"/cycles", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. PUT invalid weekly cycle (missing weekly_limit) -> 400
	payloadBad := map[string]any{
		"cycles": []map[string]any{
			{"limit_key": "ai_tokens_per_cycle", "cycle_type": "weekly_monthly"},
		},
	}
	body, _ := json.Marshal(payloadBad)
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/plans/"+planID.String()+"/cycles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 on invalid cycle, got %d", w.Code)
	}

	// 3. PUT valid weekly cycle
	wLimit := int64(50000)
	degMode := "model_downgrade"
	fallbackModel := "gpt-4o-mini"
	payloadGood := map[string]any{
		"cycles": []quota.PlanLimitCycle{
			{
				LimitKey:      "ai_tokens_per_cycle",
				CycleType:     "weekly_monthly",
				WeeklyLimit:   &wLimit,
				DegradedMode:  &degMode,
				FallbackModel: &fallbackModel,
			},
		},
	}
	body, _ = json.Marshal(payloadGood)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/plans/"+planID.String()+"/cycles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 on valid cycle, got %d: %s", w.Code, w.Body.String())
	}

	// 4. GET again -> returns updated cycle
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/plans/"+planID.String()+"/cycles", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestQuotaHandler_GetUserQuotaSnapshot(t *testing.T) {
	userID := uuid.New()
	anchor := time.Now().Add(-2 * 24 * time.Hour)
	repo := &fakeQuotaCycleRepo{
		cycles: make(map[string][]quota.PlanLimitCycle),
		counters: map[uuid.UUID][]quota.UsageCounter{
			userID: {
				{
					LimitKey:   "ai_tokens_per_cycle",
					CycleType:  "weekly",
					CycleStart: anchor,
					CycleEnd:   anchor.Add(7 * 24 * time.Hour),
					Used:       15000,
				},
			},
		},
		weeklyAnchor: &anchor,
	}
	r := setupQuotaRouter(repo, userID)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/me/quota", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
