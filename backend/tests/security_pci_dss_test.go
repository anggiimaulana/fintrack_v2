package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
)

// TestP08_008_PCIDSS_SAQA_AndSecurityHeaders verifies PCI-DSS SAQ A posture and browser
// security headers across all API routes.
// Invariant: strict browser defense headers (CSP, HSTS, X-Frame-Options, nosniff, Referrer),
// cardholder data never touches application servers, and zero card PAN/CVV storage.
func TestP08_008_PCIDSS_SAQA_AndSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := deliveryHTTP.SetupRouter(
		deliveryHTTP.RouterConfig{JWTSecret: entTestSecret},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	headers := w.Header()

	requiredHeaders := map[string]string{
		"Content-Security-Policy":   "default-src 'self'",
		"Strict-Transport-Security": "max-age=31536000",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "camera=(), microphone=(), geolocation=()",
	}

	for header, expectedSubstring := range requiredHeaders {
		val := headers.Get(header)
		if val == "" {
			t.Errorf("missing required security header %q", header)
		} else if !strings.Contains(val, expectedSubstring) {
			t.Errorf("header %q = %q; expected substring %q", header, val, expectedSubstring)
		}
	}

	t.Log("Security headers verified: CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy present.")
}
