package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestBYOK_MutationsReturn410 verifies all mutation endpoints return 410 Gone.
func TestBYOK_MutationsReturn410(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &GeminiKeyHandler{}

	cases := []struct {
		name string
		fn   func(*gin.Context)
	}{
		{"create", h.Create},
		{"update", h.Update},
		{"delete", h.Delete},
		{"test", h.Test},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/gemini-keys", nil)
		tc.fn(c)

		if w.Code != http.StatusGone {
			t.Errorf("%s: expected 410, got %d", tc.name, w.Code)
		}
		if !strings.Contains(w.Body.String(), "BYOK_DEPRECATED") {
			t.Errorf("%s: expected BYOK_DEPRECATED, got %s", tc.name, w.Body.String())
		}
	}
}

// TestBYOK_ListReturnsEmpty200 verifies GET returns 200 with an empty array.
func TestBYOK_ListReturnsEmpty200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &GeminiKeyHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/gemini-keys", nil)
	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Fatalf("expected empty data array, got %s", w.Body.String())
	}
}
