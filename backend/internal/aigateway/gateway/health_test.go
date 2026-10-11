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

// TestProbe_StatusSemantics pins what "healthy" means: reachable and not 5xx.
// A provider that answers 401/403/404 is up; only transport failure or 5xx is
// unhealthy.
func TestProbe_StatusSemantics(t *testing.T) {
	cases := []struct {
		status int
		want   string
	}{
		{200, "healthy"},
		{301, "healthy"},
		{401, "healthy"}, // unauthorized: server is up and routing
		{403, "healthy"},
		{404, "healthy"},
		{500, "unhealthy"},
		{503, "unhealthy"},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			c := NewHealthChecker(&healthRepo{}, HealthConfig{Timeout: 2 * time.Second})
			got, _ := c.probe(context.Background(), &domain.AIEndpoint{BaseURL: srv.URL})
			if got != tc.want {
				t.Fatalf("status %d: got %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

// TestProbe_HangingApiPathIsAvoided is the regression guard for the real bug
// found against the local 9router, where these were measured:
//
//	GET /             -> 307 (fast)
//	GET /health       -> 404 (fast)
//	GET /v1           -> hangs until timeout
//	GET /v1/models    -> hangs until timeout
//
// Probing the API base path therefore marked a working gateway "unhealthy"
// forever. The probe must target the ORIGIN with a neutral path.
func TestProbe_HangingApiPathIsAvoided(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1", "/v1/models":
			<-block // never answers, exactly like the local 9router
		default:
			w.WriteHeader(http.StatusNotFound) // any response proves reachability
		}
	}))
	defer func() { close(block); srv.Close() }()

	base := srv.URL + "/v1"

	// The default probe targets the origin (/health) and succeeds even though the
	// API base path hangs: a response under 500 is all reachability needs.
	def := NewHealthChecker(&healthRepo{}, HealthConfig{Timeout: 2 * time.Second})
	if got, _ := def.probe(context.Background(), &domain.AIEndpoint{BaseURL: base}); got != "healthy" {
		t.Fatalf("origin probe: got %q, want healthy", got)
	}

	// If an operator explicitly configures a hanging path, it is still reported
	// unhealthy — the probe does not pretend a silent endpoint is fine.
	explicit := NewHealthChecker(&healthRepo{}, HealthConfig{
		Timeout:    300 * time.Millisecond,
		HealthPath: "v1", // hangs on this test server
	})
	if got, _ := explicit.probe(context.Background(), &domain.AIEndpoint{BaseURL: base}); got != "unhealthy" {
		t.Fatalf("explicit hanging path: got %q, want unhealthy", got)
	}
}

// TestProbeURL_UsesOrigin pins the URL construction.
func TestProbeURL_UsesOrigin(t *testing.T) {
	cases := []struct{ base, path, want string }{
		{"http://localhost:20128/v1", "", "http://localhost:20128/health"},
		{"http://localhost:20128/v1", "readyz", "http://localhost:20128/readyz"},
		{"http://host:9000/api/v2/deep", "", "http://host:9000/health"},
	}
	for _, tc := range cases {
		if got := probeURL(tc.base, tc.path); got != tc.want {
			t.Fatalf("probeURL(%q,%q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}

// TestProbe_TransportFailureIsUnhealthy covers an unreachable endpoint.
func TestProbe_TransportFailureIsUnhealthy(t *testing.T) {
	c := NewHealthChecker(&healthRepo{}, HealthConfig{Timeout: 300 * time.Millisecond})
	// Port 1 is not listening.
	got, msg := c.probe(context.Background(), &domain.AIEndpoint{BaseURL: "http://127.0.0.1:1/v1"})
	if got != "unhealthy" {
		t.Fatalf("unreachable: got %q, want unhealthy", got)
	}
	if msg == nil {
		t.Fatal("expected an error message for an unreachable endpoint")
	}
}
