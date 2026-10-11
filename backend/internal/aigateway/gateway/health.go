package gateway

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/pkg/logger"
)

// HealthChecker periodically probes active endpoints and persists their health.
type HealthChecker struct {
	repo       domain.AIEndpointRepository
	interval   time.Duration
	timeout    time.Duration
	healthPath string
	client     *http.Client

	stop chan struct{}
	done chan struct{}
}

// HealthConfig configures the checker.
type HealthConfig struct {
	Interval time.Duration // default 60s
	Timeout  time.Duration // default 5s
	// HealthPath is the path probed on the endpoint's ORIGIN (scheme+host), not
	// on its API base path. Default "/health".
	//
	// Why the origin and a neutral path: probing the API base (…/v1) or a
	// vendor-specific path is unreliable. Measured against the local 9router:
	//   GET /        -> 307 (fast)
	//   GET /health  -> 404 (fast)
	//   GET /v1      -> hangs until timeout
	//   GET /v1/models -> hangs until timeout
	// Probing …/v1 or …/v1/models therefore reported a working gateway as
	// "unhealthy" forever. The probe measures REACHABILITY — any response under
	// 500 proves the server is up and routing; whether a model answers is what
	// real requests exercise.
	HealthPath string
}

// NewHealthChecker builds a health checker with documented defaults.
func NewHealthChecker(repo domain.AIEndpointRepository, cfg HealthConfig) *HealthChecker {
	if cfg.Interval <= 0 {
		cfg.Interval = 60 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &HealthChecker{
		repo:       repo,
		interval:   cfg.Interval,
		timeout:    cfg.Timeout,
		healthPath: cfg.HealthPath,
		client:     &http.Client{Timeout: cfg.Timeout},
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// Start launches the background probe loop.
func (c *HealthChecker) Start(ctx context.Context) {
	go func() {
		defer close(c.done)
		// Run one probe immediately, then on each tick.
		c.runOnce(ctx)
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stop:
				return
			case <-ticker.C:
				c.runOnce(ctx)
			}
		}
	}()
}

// Stop signals the loop to exit and waits for it to finish.
func (c *HealthChecker) Stop() {
	select {
	case <-c.stop:
		// already stopped
	default:
		close(c.stop)
	}
	<-c.done
}

// runOnce probes every endpoint and records the result.
func (c *HealthChecker) runOnce(ctx context.Context) {
	endpoints, err := c.repo.ListEndpoints(ctx)
	if err != nil {
		logger.Warn("health check: list endpoints failed", zap.Error(err))
		return
	}
	for _, ep := range endpoints {
		if ep.Status != "active" {
			continue
		}
		health, lastErr := c.probe(ctx, ep)
		if uerr := c.repo.UpdateEndpointHealth(ctx, ep.ID, health, time.Now(), lastErr); uerr != nil {
			logger.Warn("health check: update failed", zap.String("endpoint", ep.Name), zap.Error(uerr))
		}
	}
}

// probe checks whether the endpoint is REACHABLE.
//
// Any HTTP response with status < 500 counts as healthy, including 401/403/404:
// a server that answers "unauthorized" or "not found" is up and routing, which
// is what readiness needs to know. Only a transport failure (timeout, connection
// refused) or a 5xx marks it unhealthy.
//
// The probe deliberately does not assert that a model endpoint works — that is
// what real requests exercise, and a probe that hangs on a provider-specific path
// would report a working gateway as down.
func (c *HealthChecker) probe(ctx context.Context, ep *domain.AIEndpoint) (string, *string) {
	url := probeURL(ep.BaseURL, c.healthPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		msg := err.Error()
		return "unhealthy", &msg
	}
	resp, err := c.client.Do(req)
	if err != nil {
		msg := err.Error()
		return "unhealthy", &msg
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		msg := resp.Status
		return "unhealthy", &msg
	}
	return "healthy", nil
}

// probeURL builds the reachability URL: the endpoint's ORIGIN plus the probe
// path (default "/health"). Using the origin deliberately avoids the API base
// path, which some gateways answer only for real API calls (or not at all).
func probeURL(baseURL, healthPath string) string {
	if healthPath == "" {
		healthPath = "/health"
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		// Fall back to naive joining when the base URL is not absolute.
		return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(healthPath, "/")
	}
	return u.Scheme + "://" + u.Host + "/" + strings.TrimLeft(healthPath, "/")
}

// compile-time assertion.
var _ domain.HealthChecker = (*HealthChecker)(nil)
