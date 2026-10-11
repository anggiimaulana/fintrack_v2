// Package http — health probes for orchestrators.
//
// Two distinct questions, two endpoints:
//
//	/health/live  — is the process alive? No I/O, must never fail because a
//	                dependency is down (otherwise the orchestrator would restart
//	                a perfectly healthy process).
//	/health/ready — can this instance serve traffic? Checks the dependencies it
//	                genuinely needs.
//
// The old /health is kept as an alias of /live for compatibility with existing
// documentation and ad-hoc probes.
package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
)

// HealthHandler answers liveness and readiness probes.
type HealthHandler struct {
	pool  *pgxpool.Pool
	redis *infraRedis.Client
}

// NewHealthHandler builds the handler. pool and redis may be nil in tests, in
// which case the corresponding check reports "down"/"degraded".
func NewHealthHandler(pool *pgxpool.Pool, redis *infraRedis.Client) *HealthHandler {
	return &HealthHandler{pool: pool, redis: redis}
}

// dependencyResult is the diagnostic payload for one dependency.
type dependencyResult struct {
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Detail    string `json:"detail,omitempty"`
}

// Live handles GET /health/live (and /health). It performs no I/O: a live
// process is alive even when its dependencies are not.
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "fintrack-api",
	})
}

// Ready handles GET /health/ready.
//
// PostgreSQL is a hard dependency: without it the instance cannot serve any
// request, so a failure returns 503 and the orchestrator takes it out of
// rotation. Redis is optional in this codebase (REDIS_URL may be empty and the
// cache degrades gracefully), so a Redis failure is reported as "degraded" but
// still returns 200 — marking the pod unready for a cache outage would remove
// capacity the service does not actually need.
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	db := h.checkDatabase(ctx)
	redis := h.checkRedis(ctx)

	ready := db.Status == "ok"
	status := "ok"
	code := http.StatusOK
	if !ready {
		status = "unavailable"
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, gin.H{
		"status":  status,
		"service": "fintrack-api",
		"checks": gin.H{
			"postgres": db,
			"redis":    redis,
		},
	})
}

// checkDatabase pings PostgreSQL and verifies the pgvector extension is present,
// since the RAG/embedding path depends on it.
func (h *HealthHandler) checkDatabase(ctx context.Context) dependencyResult {
	if h.pool == nil {
		return dependencyResult{Status: "down", Detail: "not configured"}
	}
	start := time.Now()
	if err := h.pool.Ping(ctx); err != nil {
		return dependencyResult{Status: "down", LatencyMS: elapsedMS(start), Detail: err.Error()}
	}

	var hasVector bool
	if err := h.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector')`).Scan(&hasVector); err != nil {
		return dependencyResult{Status: "down", LatencyMS: elapsedMS(start), Detail: err.Error()}
	}
	res := dependencyResult{Status: "ok", LatencyMS: elapsedMS(start)}
	if !hasVector {
		// Embeddings would fail; treat as down so it surfaces loudly.
		res.Status = "down"
		res.Detail = "pgvector extension missing"
	}
	return res
}

// checkRedis pings Redis. Unconfigured or unreachable Redis is "degraded", not
// "down": the service still serves requests without it.
func (h *HealthHandler) checkRedis(ctx context.Context) dependencyResult {
	if h.redis == nil || !h.redis.Enabled() {
		return dependencyResult{Status: "degraded", Detail: "not configured"}
	}
	start := time.Now()
	if err := h.redis.Ping(ctx); err != nil {
		return dependencyResult{Status: "degraded", LatencyMS: elapsedMS(start), Detail: err.Error()}
	}
	return dependencyResult{Status: "ok", LatencyMS: elapsedMS(start)}
}

func elapsedMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
