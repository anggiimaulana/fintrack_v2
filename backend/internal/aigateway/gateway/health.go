package gateway

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/pkg/logger"
)

// HealthChecker periodically probes active endpoints and persists their health.
type HealthChecker struct {
	repo     domain.AIEndpointRepository
	interval time.Duration
	timeout  time.Duration
	client   *http.Client

	stop chan struct{}
	done chan struct{}
}

// HealthConfig configures the checker.
type HealthConfig struct {
	Interval time.Duration // default 60s
	Timeout  time.Duration // default 5s
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
		repo:     repo,
		interval: cfg.Interval,
		timeout:  cfg.Timeout,
		client:   &http.Client{Timeout: cfg.Timeout},
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
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

// probe performs a lightweight HTTP request against the endpoint.
func (c *HealthChecker) probe(ctx context.Context, ep *domain.AIEndpoint) (string, *string) {
	url := strings.TrimRight(ep.BaseURL, "/") + "/models"
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
	// 2xx/3xx/4xx means the endpoint is reachable and responding.
	return "healthy", nil
}

// compile-time assertion.
var _ domain.HealthChecker = (*HealthChecker)(nil)
