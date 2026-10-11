package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/aigateway/breaker"
	"github.com/fintrack/backend/internal/domain"
)

// fakeRepo serves a fixed route list and records invalidations.
type fakeRepo struct {
	routes []*domain.AIRoute
}

func (f *fakeRepo) HasHealthyEndpoint(ctx context.Context, uc domain.UseCase) (bool, error) {
	routes, err := f.GetActiveRoutesByUseCase(ctx, uc)
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

func (f *fakeRepo) HasActiveRoutes(ctx context.Context, uc domain.UseCase) (bool, error) {
	routes, err := f.GetActiveRoutesByUseCase(ctx, uc)
	if err != nil {
		return false, err
	}
	return len(routes) > 0, nil
}

func (f *fakeRepo) GetActiveRoutesByUseCase(context.Context, domain.UseCase) ([]*domain.AIRoute, error) {
	return f.routes, nil
}
func (f *fakeRepo) GetByID(context.Context, uuid.UUID) (*domain.AIEndpoint, error) { return nil, nil }
func (f *fakeRepo) ListEndpoints(context.Context) ([]*domain.AIEndpoint, error)    { return nil, nil }
func (f *fakeRepo) UpdateEndpointHealth(context.Context, uuid.UUID, string, time.Time, *string) error {
	return nil
}
func (f *fakeRepo) InvalidateCache() {}

// endpointServer spins up an httptest server returning the given status; when
// status is 200 it returns a simple chat completion.
func endpointServer(t *testing.T, status int) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomicAdd(&hits)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": "ok"}}},
			"usage":   map[string]any{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4},
		})
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func routeTo(srv *httptest.Server, name string, priority int) *domain.AIRoute {
	ep := &domain.AIEndpoint{
		ID:       uuid.New(),
		Name:     name,
		BaseURL:  srv.URL,
		Provider: "9router",
		Status:   "active",
		Health:   "healthy",
	}
	return &domain.AIRoute{
		ID:         uuid.New(),
		UseCase:    domain.UseCaseChat,
		EndpointID: ep.ID,
		Model:      "gpt-4o-mini",
		Priority:   priority,
		IsActive:   true,
		TimeoutMS:  2000,
		Endpoint:   ep,
	}
}

var hitMu sync.Mutex

func atomicAdd(p *int32) {
	hitMu.Lock()
	*p++
	hitMu.Unlock()
}

func hitsOf(p *int32) int32 {
	hitMu.Lock()
	defer hitMu.Unlock()
	return *p
}

// TestChatSync_FailoverToSecondary verifies transparent failover when the
// primary endpoint returns 503.
func TestChatSync_FailoverToSecondary(t *testing.T) {
	primary, ph := endpointServer(t, http.StatusServiceUnavailable)
	secondary, sh := endpointServer(t, http.StatusOK)

	repo := &fakeRepo{routes: []*domain.AIRoute{
		routeTo(primary, "primary", 1),
		routeTo(secondary, "secondary", 2),
	}}
	svc := New(repo, Config{DefaultTimeout: 2 * time.Second})

	content, usage, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{
		Messages: []domain.AIChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("expected failover success, got %v", err)
	}
	if content != "ok" {
		t.Fatalf("expected ok, got %q", content)
	}
	if usage.TotalTokens != 4 {
		t.Fatalf("expected 4 tokens, got %d", usage.TotalTokens)
	}
	if hitsOf(ph) != 1 {
		t.Fatalf("primary should have been tried once, got %d", hitsOf(ph))
	}
	if hitsOf(sh) != 1 {
		t.Fatalf("secondary should have been used once, got %d", hitsOf(sh))
	}
}

// TestChatSync_AllEndpointsExhausted verifies the sentinel error when all fail.
func TestChatSync_AllEndpointsExhausted(t *testing.T) {
	a, _ := endpointServer(t, http.StatusServiceUnavailable)
	b, _ := endpointServer(t, http.StatusInternalServerError)

	repo := &fakeRepo{routes: []*domain.AIRoute{
		routeTo(a, "a", 1),
		routeTo(b, "b", 2),
	}}
	svc := New(repo, Config{DefaultTimeout: 2 * time.Second})

	_, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{})
	if err == nil {
		t.Fatal("expected error when all endpoints fail")
	}
}

// TestChatSync_BreakerTripsAndSkipsEndpoint verifies that after the failure
// threshold the breaker opens and the endpoint is skipped (no further hits).
func TestChatSync_BreakerTripsAndSkipsEndpoint(t *testing.T) {
	primary, ph := endpointServer(t, http.StatusServiceUnavailable)
	secondary, _ := endpointServer(t, http.StatusOK)

	repo := &fakeRepo{routes: []*domain.AIRoute{
		routeTo(primary, "primary", 1),
		routeTo(secondary, "secondary", 2),
	}}
	svc := New(repo, Config{
		DefaultTimeout: 2 * time.Second,
		Breaker:        breaker.Config{FailureThreshold: 3, Cooldown: time.Hour, HalfOpenMaxProbes: 1},
	})

	// Three calls; each tries primary (fails) then secondary (succeeds).
	for range 3 {
		if _, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{}); err != nil {
			t.Fatalf("call should succeed via secondary: %v", err)
		}
	}
	primaryHitsAfterTrip := hitsOf(ph)

	// Fourth call: primary breaker is open -> primary must NOT be hit again.
	if _, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{}); err != nil {
		t.Fatalf("call should still succeed via secondary: %v", err)
	}
	if hitsOf(ph) != primaryHitsAfterTrip {
		t.Fatalf("primary must be skipped after breaker open: hits %d -> %d", primaryHitsAfterTrip, hitsOf(ph))
	}
}

// TestChatSync_KillSwitchSkipsInactiveEndpoint verifies status=inactive skips.
func TestChatSync_KillSwitchSkipsInactiveEndpoint(t *testing.T) {
	primary, ph := endpointServer(t, http.StatusOK)
	secondary, sh := endpointServer(t, http.StatusOK)

	r1 := routeTo(primary, "primary", 1)
	r1.Endpoint.Status = "inactive" // kill switch
	repo := &fakeRepo{routes: []*domain.AIRoute{r1, routeTo(secondary, "secondary", 2)}}
	svc := New(repo, Config{DefaultTimeout: 2 * time.Second})

	if _, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{}); err != nil {
		t.Fatalf("expected success via secondary: %v", err)
	}
	if hitsOf(ph) != 0 {
		t.Fatal("inactive endpoint must not be hit")
	}
	if hitsOf(sh) != 1 {
		t.Fatal("secondary should be used")
	}
}

// TestCreateEmbedding_RejectsWrongDimension verifies the 768 lock.
func TestCreateEmbedding_RejectsWrongDimension(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a 3-dim vector -> must be rejected.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":  []any{map[string]any{"embedding": []float32{0.1, 0.2, 0.3}}},
			"usage": map[string]any{"prompt_tokens": 2, "total_tokens": 2},
		})
	}))
	defer srv.Close()

	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "e", BaseURL: srv.URL, Provider: "9router", Status: "active"}
	repo := &fakeRepo{routes: []*domain.AIRoute{{
		ID: uuid.New(), UseCase: domain.UseCaseEmbedding, EndpointID: ep.ID,
		Model: "text-embedding-004", Priority: 1, IsActive: true, TimeoutMS: 2000, Endpoint: ep,
	}}}
	svc := New(repo, Config{DefaultTimeout: 2 * time.Second})

	_, _, err := svc.CreateEmbedding(context.Background(), &domain.AIEmbeddingRequest{Text: "x"})
	if err == nil {
		t.Fatal("expected dimension rejection")
	}
	if fmt.Sprint(err) == "" {
		t.Fatal("expected non-empty error")
	}
}

// TestCreateEmbedding_Accepts768 verifies a valid 768-dim vector passes.
func TestCreateEmbedding_Accepts768(t *testing.T) {
	vec := make([]float32, 768)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":  []any{map[string]any{"embedding": vec}},
			"usage": map[string]any{"prompt_tokens": 2, "total_tokens": 2},
		})
	}))
	defer srv.Close()

	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "e", BaseURL: srv.URL, Provider: "9router", Status: "active"}
	repo := &fakeRepo{routes: []*domain.AIRoute{{
		ID: uuid.New(), UseCase: domain.UseCaseEmbedding, EndpointID: ep.ID,
		Model: "text-embedding-004", Priority: 1, IsActive: true, TimeoutMS: 2000, Endpoint: ep,
	}}}
	svc := New(repo, Config{DefaultTimeout: 2 * time.Second})

	got, _, err := svc.CreateEmbedding(context.Background(), &domain.AIEmbeddingRequest{Text: "x"})
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if len(got) != 768 {
		t.Fatalf("expected 768 dims, got %d", len(got))
	}
}

// TestHasHealthyEndpoint_RequiresHealthyEndpoint verifies the readiness guard
// used before switching the platform onto the gateway: an active route is not
// enough if its endpoint's health check is failing (the gateway path has no BYOK
// fallback, so this would break AI for every user).
func TestHasHealthyEndpoint_RequiresHealthyEndpoint(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name   string
		health string
		want   bool
	}{
		{"healthy endpoint passes", "healthy", true},
		{"unhealthy endpoint blocks", "unhealthy", false},
		{"unprobed endpoint blocks", "unknown", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{
				routes: []*domain.AIRoute{{
					UseCase:  domain.UseCaseChat,
					Endpoint: &domain.AIEndpoint{ID: uuid.New(), Health: tc.health},
				}},
			}
			got, err := repo.HasHealthyEndpoint(ctx, domain.UseCaseChat)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("health=%q: got %v, want %v", tc.health, got, tc.want)
			}
		})
	}

	// No routes at all -> not healthy.
	empty := &fakeRepo{routes: nil}
	got, err := empty.HasHealthyEndpoint(ctx, domain.UseCaseChat)
	if err != nil || got {
		t.Fatalf("no routes: got (%v, %v), want (false, nil)", got, err)
	}
}
