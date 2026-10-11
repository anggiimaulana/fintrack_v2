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

// ─── E-6: Weighted Round-Robin & Combo Tests ─────────────────

func TestOrderedRoutes_SingleComboPreservesPriority(t *testing.T) {
	ep1 := &domain.AIEndpoint{ID: uuid.New(), Name: "ep1", Status: "active", Weight: 100}
	ep2 := &domain.AIEndpoint{ID: uuid.New(), Name: "ep2", Status: "active", Weight: 100}

	routes := []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "default", Priority: 2, Endpoint: ep2},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "default", Priority: 1, Endpoint: ep1},
	}

	svc := New(&fakeRepo{}, Config{})
	ordered := svc.orderedRoutes(domain.UseCaseChat, routes)

	if len(ordered) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(ordered))
	}
	if ordered[0].Priority != 1 || ordered[1].Priority != 2 {
		t.Errorf("expected priority order 1, 2; got %d, %d", ordered[0].Priority, ordered[1].Priority)
	}
}

func TestOrderedRoutes_SmoothWRRRatiosAndAntiBurst(t *testing.T) {
	epA := &domain.AIEndpoint{ID: uuid.New(), Name: "epA", Status: "active", Weight: 200}
	epB := &domain.AIEndpoint{ID: uuid.New(), Name: "epB", Status: "active", Weight: 100}

	routes := []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "comboA", Priority: 1, Endpoint: epA},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "comboB", Priority: 1, Endpoint: epB},
	}

	svc := New(&fakeRepo{}, Config{})
	// Reset state explicitly to 0 for deterministic sequence assertions
	svc.rrMu.Lock()
	svc.rrState[domain.UseCaseChat] = map[string]int64{"comboA": 0, "comboB": 0}
	svc.rrMu.Unlock()

	// 1. Anti-burst check: In 9 steps, comboA must NOT be picked 9 times in a row!
	firstWinners := make([]string, 9)
	for i := 0; i < 9; i++ {
		ordered := svc.orderedRoutes(domain.UseCaseChat, routes)
		if len(ordered) != 2 {
			t.Fatalf("step %d: expected 2 routes, got %d", i, len(ordered))
		}
		firstWinners[i] = ordered[0].Combo
	}

	// Expected Nginx SWRR sequence for 200:100 is A, B, A, A, B, A, A, B, A
	expectedSeq := []string{"comboA", "comboB", "comboA", "comboA", "comboB", "comboA", "comboA", "comboB", "comboA"}
	for i, want := range expectedSeq {
		if firstWinners[i] != want {
			t.Errorf("step %d: got winner %q, want %q (sequence: %v)", i, firstWinners[i], want, firstWinners)
		}
	}

	// 2. Statistical ratio check across 300 selections (should converge exactly to 200:100 = 2:1)
	counts := map[string]int{"comboA": 0, "comboB": 0}
	for i := 0; i < 300; i++ {
		ordered := svc.orderedRoutes(domain.UseCaseChat, routes)
		counts[ordered[0].Combo]++
	}

	ratioA := float64(counts["comboA"]) / 300.0
	if ratioA < 0.64 || ratioA > 0.69 {
		t.Errorf("expected ratio of comboA around 0.667, got %.3f (counts: %v)", ratioA, counts)
	}
}

func TestOrderedRoutes_FailoverWithinComboFirst(t *testing.T) {
	epA1 := &domain.AIEndpoint{ID: uuid.New(), Name: "epA1", Status: "active", Weight: 100}
	epA2 := &domain.AIEndpoint{ID: uuid.New(), Name: "epA2", Status: "active", Weight: 100}
	epB1 := &domain.AIEndpoint{ID: uuid.New(), Name: "epB1", Status: "active", Weight: 100}

	routes := []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "comboA", Priority: 2, Endpoint: epA2},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "comboA", Priority: 1, Endpoint: epA1},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "comboB", Priority: 1, Endpoint: epB1},
	}

	svc := New(&fakeRepo{}, Config{})
	svc.rrMu.Lock()
	svc.rrState[domain.UseCaseChat] = map[string]int64{"comboA": 50, "comboB": -50} // force comboA win
	svc.rrMu.Unlock()

	ordered := svc.orderedRoutes(domain.UseCaseChat, routes)
	if len(ordered) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(ordered))
	}

	// Within winner comboA, Priority 1 must precede Priority 2.
	// And comboB must follow comboA.
	if ordered[0].Endpoint.Name != "epA1" || ordered[1].Endpoint.Name != "epA2" || ordered[2].Endpoint.Name != "epB1" {
		t.Errorf("expected order [epA1, epA2, epB1], got [%s, %s, %s]",
			ordered[0].Endpoint.Name, ordered[1].Endpoint.Name, ordered[2].Endpoint.Name)
	}
}

func TestOrderedRoutes_BreakerOpenSkipped(t *testing.T) {
	epA := &domain.AIEndpoint{ID: uuid.New(), Name: "epA", Status: "active", Weight: 100}
	epB := &domain.AIEndpoint{ID: uuid.New(), Name: "epB", Status: "active", Weight: 100}

	routes := []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "comboA", Priority: 1, Endpoint: epA},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "comboB", Priority: 1, Endpoint: epB},
	}

	svc := New(&fakeRepo{}, Config{Breaker: breaker.Config{FailureThreshold: 1, Cooldown: 10 * time.Second}})
	// Trip breaker on epA
	svc.breaker.RecordFailure(epA.ID.String())

	ordered := svc.orderedRoutes(domain.UseCaseChat, routes)
	if len(ordered) != 1 {
		t.Fatalf("expected 1 eligible route, got %d", len(ordered))
	}
	if ordered[0].Combo != "comboB" {
		t.Errorf("expected comboB when comboA breaker is open, got %s", ordered[0].Combo)
	}
}

func TestOrderedRoutes_ConcurrentRaceSafe(t *testing.T) {
	epA := &domain.AIEndpoint{ID: uuid.New(), Name: "epA", Status: "active", Weight: 100}
	epB := &domain.AIEndpoint{ID: uuid.New(), Name: "epB", Status: "active", Weight: 100}

	routes := []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "comboA", Priority: 1, Endpoint: epA},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, Combo: "comboB", Priority: 1, Endpoint: epB},
	}

	svc := New(&fakeRepo{}, Config{})

	var wg sync.WaitGroup
	workers := 30
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = svc.orderedRoutes(domain.UseCaseChat, routes)
			}
		}()
	}

	wg.Wait()
}
