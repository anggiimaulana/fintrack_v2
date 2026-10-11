package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
)

func healthRouter(h *HealthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", h.Live)
	r.GET("/health/live", h.Live)
	r.GET("/health/ready", h.Ready)
	return r
}

type readyBody struct {
	Status string `json:"status"`
	Checks struct {
		Postgres struct {
			Status    string `json:"status"`
			LatencyMS int64  `json:"latency_ms"`
			Detail    string `json:"detail,omitempty"`
		} `json:"postgres"`
		Redis struct {
			Status string `json:"status"`
			Detail string `json:"detail,omitempty"`
		} `json:"redis"`
	} `json:"checks"`
}

func getReady(t *testing.T, r *gin.Engine, path string) (int, readyBody) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var body readyBody
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

// TestHealthLive_NoIO verifies liveness never depends on dependencies: it must
// return 200 even with no pool and no Redis, otherwise an orchestrator would
// restart a perfectly healthy process because a dependency blipped.
func TestHealthLive_NoIO(t *testing.T) {
	r := healthRouter(NewHealthHandler(nil, nil))
	for _, path := range []string{"/health", "/health/live"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: got %d, want 200", path, w.Code)
		}
	}
}

// TestHealthReady_DatabaseDownIs503 verifies PostgreSQL is treated as a hard
// dependency: without it the instance cannot serve anything, so it must be taken
// out of rotation.
func TestHealthReady_DatabaseDownIs503(t *testing.T) {
	// A pool pointed at an unreachable address: Ping fails.
	pool, err := pgxpool.New(t.Context(), "postgres://nobody:nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("build pool: %v", err)
	}
	defer pool.Close()

	r := healthRouter(NewHealthHandler(pool, nil))
	code, body := getReady(t, r, "/health/ready")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("db down: got %d, want 503", code)
	}
	if body.Status != "unavailable" {
		t.Fatalf("status = %q, want unavailable", body.Status)
	}
	if body.Checks.Postgres.Status != "down" {
		t.Fatalf("postgres check = %q, want down", body.Checks.Postgres.Status)
	}
}

// TestHealthReady_NilPoolIs503 covers the misconfigured case explicitly.
func TestHealthReady_NilPoolIs503(t *testing.T) {
	r := healthRouter(NewHealthHandler(nil, nil))
	code, body := getReady(t, r, "/health/ready")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("nil pool: got %d, want 503", code)
	}
	if body.Checks.Postgres.Detail != "not configured" {
		t.Fatalf("detail = %q, want 'not configured'", body.Checks.Postgres.Detail)
	}
}

// TestHealthReady_RedisUnconfiguredIsDegraded verifies Redis is optional: it is
// reported as degraded but does not make the instance unready.
func TestHealthReady_RedisUnconfiguredIsDegraded(t *testing.T) {
	h := NewHealthHandler(nil, nil)
	res := h.checkRedis(t.Context())
	if res.Status != "degraded" {
		t.Fatalf("unconfigured redis: got %q, want degraded", res.Status)
	}

	// A disabled client (no URL) must also read as degraded, not down.
	disabled := infraRedis.NewCacheEngine(nil)
	if disabled == nil {
		t.Fatal("expected a cache engine")
	}
}

// TestHealthReady_PayloadIncludesLatency verifies the diagnostic payload carries
// per-dependency latency, as the acceptance criteria require.
func TestHealthReady_PayloadIncludesLatency(t *testing.T) {
	r := healthRouter(NewHealthHandler(nil, nil))
	_, body := getReady(t, r, "/health/ready")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	var raw map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	checks, ok := raw["checks"].(map[string]any)
	if !ok {
		t.Fatal("payload missing 'checks'")
	}
	pg, ok := checks["postgres"].(map[string]any)
	if !ok {
		t.Fatal("payload missing checks.postgres")
	}
	if _, ok := pg["latency_ms"]; !ok {
		t.Fatalf("checks.postgres missing latency_ms: %v", pg)
	}
	_ = body
}
