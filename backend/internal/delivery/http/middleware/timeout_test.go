package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestSetAIWriteTimeout_StoreAndRead verifies the runtime knob round-trips and
// that a non-positive value means "unbounded" (0).
func TestSetAIWriteTimeout_StoreAndRead(t *testing.T) {
	t.Cleanup(func() { SetAIWriteTimeout(0) })

	SetAIWriteTimeout(120 * time.Second)
	if got := AIWriteTimeout(); got != 120*time.Second {
		t.Fatalf("expected 120s, got %v", got)
	}

	SetAIWriteTimeout(0)
	if got := AIWriteTimeout(); got != 0 {
		t.Fatalf("expected 0 (unbounded), got %v", got)
	}
}

// TestAIDeadline_DoesNotBreakHandler verifies the middleware is transparent: the
// handler still runs and writes its response, whether the budget is set or not.
// (httptest's writer does not support SetWriteDeadline, so this also exercises
// the graceful no-op path.)
func TestAIDeadline_DoesNotBreakHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { SetAIWriteTimeout(0) })

	for _, budget := range []time.Duration{0, 90 * time.Second} {
		SetAIWriteTimeout(budget)
		r := gin.New()
		r.GET("/x", AIDeadline(), func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Body.String() != "ok" {
			t.Fatalf("budget %v: expected 200/ok, got %d/%q", budget, w.Code, w.Body.String())
		}
	}
}

// TestNoWriteTimeout_DoesNotBreakHandler is the SSE counterpart: clearing the
// deadline must not disturb a normal response.
func TestNoWriteTimeout_DoesNotBreakHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", NoWriteTimeout(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Fatalf("expected 200/ok, got %d/%q", w.Code, w.Body.String())
	}
}
