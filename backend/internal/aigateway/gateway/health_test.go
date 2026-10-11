package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// healthRepo is a repository stub for health checks.
type healthRepo struct {
	endpoints []*domain.AIEndpoint

	mu      sync.Mutex
	updated map[uuid.UUID]string
	lastErr map[uuid.UUID]*string
}

func newHealthRepo(eps ...*domain.AIEndpoint) *healthRepo {
	return &healthRepo{endpoints: eps, updated: map[uuid.UUID]string{}, lastErr: map[uuid.UUID]*string{}}
}

func (r *healthRepo) HasHealthyEndpoint(ctx context.Context, uc domain.UseCase) (bool, error) {
	routes, err := r.GetActiveRoutesByUseCase(ctx, uc)
	if err != nil {
		return false, err
	}
	for _, rt := range routes {
		if rt.Endpoint != nil && rt.Endpoint.Health == "healthy" {
			return true, nil
		}
	}
	return false, nil
}

func (r *healthRepo) HasActiveRoutes(ctx context.Context, uc domain.UseCase) (bool, error) {
	routes, err := r.GetActiveRoutesByUseCase(ctx, uc)
	if err != nil {
		return false, err
	}
	return len(routes) > 0, nil
}

func (r *healthRepo) GetActiveRoutesByUseCase(context.Context, domain.UseCase) ([]*domain.AIRoute, error) {
	return nil, nil
}
func (r *healthRepo) GetByID(context.Context, uuid.UUID) (*domain.AIEndpoint, error) { return nil, nil }
func (r *healthRepo) ListEndpoints(context.Context) ([]*domain.AIEndpoint, error) {
	return r.endpoints, nil
}
func (r *healthRepo) UpdateEndpointHealth(_ context.Context, id uuid.UUID, health string, _ time.Time, lastErr *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updated[id] = health
	r.lastErr[id] = lastErr
	return nil
}
func (r *healthRepo) InvalidateCache() {}

func (r *healthRepo) healthOf(id uuid.UUID) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.updated[id]
	return v, ok
}

// TestHealthChecker_MarksUnhealthy verifies a 5xx endpoint is marked unhealthy.
func TestHealthChecker_MarksUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "bad", BaseURL: srv.URL, Status: "active"}
	repo := newHealthRepo(ep)

	hc := NewHealthChecker(repo, HealthConfig{Interval: time.Hour, Timeout: 2 * time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	hc.Start(ctx)
	// Wait for the immediate first probe.
	time.Sleep(150 * time.Millisecond)
	hc.Stop()
	cancel()

	health, ok := repo.healthOf(ep.ID)
	if !ok {
		t.Fatal("expected health update")
	}
	if health != "unhealthy" {
		t.Fatalf("expected unhealthy, got %s", health)
	}
}

// TestHealthChecker_MarksHealthy verifies a reachable endpoint is healthy.
func TestHealthChecker_MarksHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "good", BaseURL: srv.URL, Status: "active"}
	repo := newHealthRepo(ep)

	hc := NewHealthChecker(repo, HealthConfig{Interval: time.Hour, Timeout: 2 * time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	hc.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	hc.Stop()
	cancel()

	health, ok := repo.healthOf(ep.ID)
	if !ok || health != "healthy" {
		t.Fatalf("expected healthy, got %s ok=%v", health, ok)
	}
}

// TestHealthChecker_SkipsInactive verifies inactive endpoints are not probed.
func TestHealthChecker_SkipsInactive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "off", BaseURL: srv.URL, Status: "inactive"}
	repo := newHealthRepo(ep)

	hc := NewHealthChecker(repo, HealthConfig{Interval: time.Hour, Timeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	hc.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	hc.Stop()
	cancel()

	if _, ok := repo.healthOf(ep.ID); ok {
		t.Fatal("inactive endpoint must not be probed")
	}
}

// TestHealthChecker_StopIdempotent verifies Stop can be called safely.
func TestHealthChecker_StopIdempotent(t *testing.T) {
	repo := newHealthRepo()
	hc := NewHealthChecker(repo, HealthConfig{Interval: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	hc.Start(ctx)
	hc.Stop()
	hc.Stop() // must not panic
	cancel()
}
