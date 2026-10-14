package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// verifyRepo serves routes per use case.
type verifyRepo struct {
	byUseCase map[domain.UseCase][]*domain.AIRoute
}

func (r *verifyRepo) HasHealthyEndpoint(context.Context, domain.UseCase) (bool, error) {
	return true, nil
}
func (r *verifyRepo) HasActiveRoutes(ctx context.Context, uc domain.UseCase) (bool, error) {
	return len(r.byUseCase[uc]) > 0, nil
}
func (r *verifyRepo) GetActiveRoutesByUseCase(_ context.Context, uc domain.UseCase) ([]*domain.AIRoute, error) {
	return r.byUseCase[uc], nil
}
func (r *verifyRepo) GetByID(context.Context, uuid.UUID) (*domain.AIEndpoint, error) { return nil, nil }
func (r *verifyRepo) ListEndpoints(context.Context) ([]*domain.AIEndpoint, error)    { return nil, nil }
func (r *verifyRepo) UpdateEndpointHealth(context.Context, uuid.UUID, string, time.Time, *string) error {
	return nil
}
func (r *verifyRepo) InvalidateCache() {}

// verifyRouteOn builds a route pointing at srv for a use case + model.
func verifyRouteOn(srv *httptest.Server, uc domain.UseCase, model string) *domain.AIRoute {
	ep := &domain.AIEndpoint{
		ID:       uuid.New(),
		Name:     "9router-primary",
		BaseURL:  srv.URL,
		Provider: "9router",
		Status:   "active",
		Health:   "healthy",
	}
	return &domain.AIRoute{
		ID:         uuid.New(),
		UseCase:    uc,
		EndpointID: ep.ID,
		Model:      model,
		Combo:      "default",
		Priority:   1,
		IsActive:   true,
		Endpoint:   ep,
	}
}

// chatOKServer answers chat completions with 200 and embeddings with 768 dims.
func chatOKServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/embeddings":
			vec := make([]float32, 768)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":  []any{map[string]any{"embedding": vec}},
				"usage": map[string]any{"prompt_tokens": 1, "total_tokens": 1},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{"message": map[string]any{"content": "OK"}}},
				"usage":   map[string]any{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4},
			})
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestVerifyRoutes_CatchesDeadModel is the core regression guard: the endpoint
// is REACHABLE (answers 503, not a transport error) but the configured model is
// dead. Reachability probing reports "healthy"; only a real model call catches it.
func TestVerifyRoutes_CatchesDeadModel(t *testing.T) {
	srv, _ := endpointServer(t, http.StatusServiceUnavailable)
	repo := &verifyRepo{byUseCase: map[domain.UseCase][]*domain.AIRoute{
		domain.UseCaseChat: {verifyRouteOn(srv, domain.UseCaseChat, "gemini-2.5-flash")},
	}}
	svc := New(repo, Config{DefaultTimeout: 5 * time.Second})

	results := svc.VerifyRoutes(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	got := results[0]
	if got.OK {
		t.Fatalf("dead model must not be reported OK: %+v", got)
	}
	if got.Error == "" {
		t.Fatal("expected an error message for a dead model")
	}
	if got.Model != "gemini-2.5-flash" {
		t.Fatalf("expected model echoed, got %q", got.Model)
	}
}

// TestVerifyRoutes_HealthyModel verifies a working model is reported OK.
func TestVerifyRoutes_HealthyModel(t *testing.T) {
	srv := chatOKServer(t)
	repo := &verifyRepo{byUseCase: map[domain.UseCase][]*domain.AIRoute{
		domain.UseCaseChat: {verifyRouteOn(srv, domain.UseCaseChat, "gemini/gemini-3.5-flash-lite")},
	}}
	svc := New(repo, Config{DefaultTimeout: 5 * time.Second})

	results := svc.VerifyRoutes(context.Background())
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("healthy model should be OK: %+v", results)
	}
}

// TestVerifyRoutes_VoiceIsSkippedNeverFailed verifies voice routes are reported
// but never failed (they need real audio, which has no synthetic fixture).
func TestVerifyRoutes_VoiceIsSkippedNeverFailed(t *testing.T) {
	srv, hits := endpointServer(t, http.StatusServiceUnavailable)
	repo := &verifyRepo{byUseCase: map[domain.UseCase][]*domain.AIRoute{
		domain.UseCaseVoice: {verifyRouteOn(srv, domain.UseCaseVoice, "whisper-1")},
	}}
	svc := New(repo, Config{DefaultTimeout: 5 * time.Second})

	results := svc.VerifyRoutes(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Skipped || results[0].OK {
		t.Fatalf("voice must be skipped, not failed: %+v", results[0])
	}
	if *hits != 0 {
		t.Fatalf("voice probe must not call upstream, got %d hits", *hits)
	}
}

// TestVerifyRoutes_EmbeddingChecksDimensions verifies a wrong-dimension
// embedding is a failure (the schema is vector(768)).
func TestVerifyRoutes_EmbeddingChecksDimensions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":  []any{map[string]any{"embedding": make([]float32, 3072)}},
			"usage": map[string]any{"total_tokens": 1},
		})
	}))
	defer srv.Close()

	repo := &verifyRepo{byUseCase: map[domain.UseCase][]*domain.AIRoute{
		domain.UseCaseEmbedding: {verifyRouteOn(srv, domain.UseCaseEmbedding, "gemini-embedding-001")},
	}}
	svc := New(repo, Config{DefaultTimeout: 5 * time.Second})

	results := svc.VerifyRoutes(context.Background())
	if len(results) != 1 || results[0].OK {
		t.Fatalf("wrong-dimension embedding must fail: %+v", results)
	}
}

// TestVerifyRoutes_EmbeddingOK verifies a correct 768-dim embedding passes.
func TestVerifyRoutes_EmbeddingOK(t *testing.T) {
	srv := chatOKServer(t)
	repo := &verifyRepo{byUseCase: map[domain.UseCase][]*domain.AIRoute{
		domain.UseCaseEmbedding: {verifyRouteOn(srv, domain.UseCaseEmbedding, "gemini-embedding-001")},
	}}
	svc := New(repo, Config{DefaultTimeout: 5 * time.Second})

	results := svc.VerifyRoutes(context.Background())
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("768-dim embedding should pass: %+v", results)
	}
}

// TestVerifyRoutes_TimeoutIsReported verifies a hanging model is reported as a
// timeout rather than hanging the caller forever.
func TestVerifyRoutes_TimeoutIsReported(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block // never answers
	}))
	defer func() { close(block); srv.Close() }()

	rt := verifyRouteOn(srv, domain.UseCaseChat, "slow-model")
	rt.TimeoutMS = 300 // 300ms probe budget
	repo := &verifyRepo{byUseCase: map[domain.UseCase][]*domain.AIRoute{
		domain.UseCaseChat: {rt},
	}}
	svc := New(repo, Config{DefaultTimeout: 5 * time.Second})

	start := time.Now()
	results := svc.VerifyRoutes(context.Background())
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("probe must respect the route timeout, took %v", elapsed)
	}
	if len(results) != 1 || results[0].OK {
		t.Fatalf("hanging model must fail: %+v", results)
	}
	if results[0].Error == "" {
		t.Fatal("expected a timeout error message")
	}
}

// TestVerifyRoutes_IsReadOnly verifies verification never mutates anything: it
// must not touch health, routes, or the breaker — a false negative can never
// take AI down.
func TestVerifyRoutes_IsReadOnly(t *testing.T) {
	srv, _ := endpointServer(t, http.StatusServiceUnavailable)
	ep := &domain.AIEndpoint{
		ID: uuid.New(), Name: "9router-primary", BaseURL: srv.URL,
		Provider: "9router", Status: "active", Health: "healthy",
	}
	rt := &domain.AIRoute{
		ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID,
		Model: "dead", Priority: 1, IsActive: true, Endpoint: ep,
	}
	repo := &verifyRepo{byUseCase: map[domain.UseCase][]*domain.AIRoute{
		domain.UseCaseChat: {rt},
	}}
	svc := New(repo, Config{DefaultTimeout: 5 * time.Second})

	_ = svc.VerifyRoutes(context.Background())

	if ep.Health != "healthy" {
		t.Fatalf("verify must not change endpoint health, got %q", ep.Health)
	}
	if rt.Model != "dead" || !rt.IsActive {
		t.Fatalf("verify must not change the route: %+v", rt)
	}
	// Breaker must stay closed (verify does not record failures).
	if state := svc.breaker.State(ep.ID.String()); state != "closed" {
		t.Fatalf("verify must not trip the breaker, got %q", state)
	}
}

// TestVerifyRoutes_NoRoutesIsEmpty verifies an empty config yields no results.
func TestVerifyRoutes_NoRoutesIsEmpty(t *testing.T) {
	svc := New(&verifyRepo{byUseCase: map[domain.UseCase][]*domain.AIRoute{}}, Config{})
	if got := svc.VerifyRoutes(context.Background()); len(got) != 0 {
		t.Fatalf("expected no results, got %+v", got)
	}
}

// TestDecodeProbePNG sanity-checks the vision fixture is a real PNG.
func TestDecodeProbePNG(t *testing.T) {
	b := decodeProbePNG()
	if len(b) < 8 || b[0] != 0x89 || b[1] != 'P' || b[2] != 'N' || b[3] != 'G' {
		t.Fatalf("expected a PNG header, got % x", b[:min(8, len(b))])
	}
}
