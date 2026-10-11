package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Registry holds the application metrics. A dedicated registry (rather than the
// global default) keeps the exposed surface explicit and testable.
type Registry struct {
	reg *prometheus.Registry

	// Four golden signals.
	HTTPRequests *prometheus.CounterVec
	HTTPDuration *prometheus.HistogramVec
	HTTPInFlight prometheus.Gauge
	HTTPErrors   *prometheus.CounterVec

	// Business counters (populated by later phases: P03 token/failover, P05 payment).
	TokensUsed      *prometheus.CounterVec
	GatewayFailover prometheus.Counter
	Payments        *prometheus.CounterVec
}

// New builds a metrics registry with all collectors registered.
func New() *Registry {
	reg := prometheus.NewRegistry()
	f := promauto.With(reg)

	m := &Registry{reg: reg}

	m.HTTPRequests = f.NewCounterVec(prometheus.CounterOpts{
		Name: "fintrack_http_requests_total",
		Help: "Total number of HTTP requests processed.",
	}, []string{"method", "route", "status"})

	m.HTTPDuration = f.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fintrack_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"method", "route"})

	m.HTTPInFlight = f.NewGauge(prometheus.GaugeOpts{
		Name: "fintrack_http_in_flight_requests",
		Help: "Number of HTTP requests currently being processed (saturation).",
	})

	m.HTTPErrors = f.NewCounterVec(prometheus.CounterOpts{
		Name: "fintrack_http_errors_total",
		Help: "Total number of HTTP requests that returned a 4xx/5xx status.",
	}, []string{"method", "route", "status"})

	m.TokensUsed = f.NewCounterVec(prometheus.CounterOpts{
		Name: "fintrack_ai_tokens_used_total",
		Help: "Total AI tokens consumed (by modality). Populated from P03.",
	}, []string{"modality"})

	m.GatewayFailover = f.NewCounter(prometheus.CounterOpts{
		Name: "fintrack_ai_gateway_failover_total",
		Help: "Total AI gateway endpoint failovers. Populated from P03.",
	})

	m.Payments = f.NewCounterVec(prometheus.CounterOpts{
		Name: "fintrack_payments_total",
		Help: "Total payment events by gateway and status. Populated from P05.",
	}, []string{"gateway", "status"})

	return m
}

// Registry exposes the underlying prometheus registry for the /metrics handler.
func (m *Registry) Registry() *prometheus.Registry { return m.reg }

// Middleware records the four golden signals for each request. Route labels use
// the matched route pattern (c.FullPath()) to keep cardinality bounded.
func (m *Registry) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		m.HTTPInFlight.Inc()

		c.Next()

		m.HTTPInFlight.Dec()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := c.Writer.Status()
		statusStr := strconv.Itoa(status)

		m.HTTPRequests.WithLabelValues(c.Request.Method, route, statusStr).Inc()
		m.HTTPDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
		if status >= 400 {
			m.HTTPErrors.WithLabelValues(c.Request.Method, route, statusStr).Inc()
		}
	}
}
