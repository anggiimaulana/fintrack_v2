package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TestMiddleware_RecordsGoldenSignals verifies that requests are counted, that
// latency is observed, and that errors are recorded.
func TestMiddleware_RecordsGoldenSignals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := New()

	r := gin.New()
	r.Use(m.Middleware())
	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.GET("/bad", func(c *gin.Context) { c.JSON(http.StatusInternalServerError, gin.H{"err": true}) })

	// Exercise a 200 and a 500.
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ok", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/bad", nil))

	// Scrape metrics.
	w := httptest.NewRecorder()
	promhttp.HandlerFor(m.Registry(), promhttp.HandlerOpts{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := w.Body.String()

	checks := []string{
		`fintrack_http_requests_total{method="GET",route="/ok",status="200"} 1`,
		`fintrack_http_requests_total{method="GET",route="/bad",status="500"} 1`,
		`fintrack_http_errors_total{method="GET",route="/bad",status="500"} 1`,
		`fintrack_http_request_duration_seconds_count{method="GET",route="/ok"} 1`,
		`fintrack_http_in_flight_requests 0`,
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q\n---\n%s", want, body)
		}
	}

	// Errors must NOT be recorded for successful requests.
	if strings.Contains(body, `fintrack_http_errors_total{method="GET",route="/ok"`) {
		t.Fatal("successful request must not be counted as an error")
	}
}
