package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/aigateway/breaker"
	"github.com/fintrack/backend/internal/domain"
)

// sseServer streams two content chunks + a usage chunk + [DONE].
func sseServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		lines := []string{
			`data: {"choices":[{"delta":{"content":"Halo"}}]}`,
			`data: {"choices":[{"delta":{"content":" dunia"}}]}`,
			`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":7,"completion_tokens":2,"total_tokens":9}}`,
			`data: [DONE]`,
		}
		for _, l := range lines {
			fmt.Fprintf(w, "%s\n\n", l)
			fl.Flush()
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func singleRoute(srv *httptest.Server) *fakeRepo {
	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "primary", BaseURL: srv.URL, Provider: "9router", Status: "active"}
	return &fakeRepo{routes: []*domain.AIRoute{{
		ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID,
		Model: "gpt-4o-mini", Priority: 1, IsActive: true, TimeoutMS: 2000, Endpoint: ep,
	}}}
}

// TestIntegration_ChatStream_Normal verifies the full streaming path.
func TestIntegration_ChatStream_Normal(t *testing.T) {
	srv := sseServer(t)
	svc := New(singleRoute(srv), Config{DefaultTimeout: 2 * time.Second})

	ch, err := svc.ChatStream(context.Background(), &domain.AIChatRequest{
		Messages: []domain.AIChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	var text string
	var usage domain.TokenUsage
	for chunk := range ch {
		if chunk.Done {
			usage = chunk.Usage
			break
		}
		text += chunk.Content
	}
	if text != "Halo dunia" {
		t.Fatalf("expected 'Halo dunia', got %q", text)
	}
	if usage.TotalTokens != 9 {
		t.Fatalf("expected 9 tokens, got %d", usage.TotalTokens)
	}
}

// TestIntegration_TransparentFailover verifies 503 -> secondary in < 1s.
func TestIntegration_TransparentFailover(t *testing.T) {
	primary, _ := endpointServer(t, http.StatusServiceUnavailable)
	secondary := sseServer(t)

	ep1 := &domain.AIEndpoint{ID: uuid.New(), Name: "p", BaseURL: primary.URL, Provider: "9router", Status: "active"}
	ep2 := &domain.AIEndpoint{ID: uuid.New(), Name: "s", BaseURL: secondary.URL, Provider: "9router", Status: "active"}
	repo := &fakeRepo{routes: []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep1.ID, Model: "m", Priority: 1, IsActive: true, TimeoutMS: 2000, Endpoint: ep1},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep2.ID, Model: "m", Priority: 2, IsActive: true, TimeoutMS: 2000, Endpoint: ep2},
	}}
	svc := New(repo, Config{DefaultTimeout: 2 * time.Second})

	start := time.Now()
	ch, err := svc.ChatStream(context.Background(), &domain.AIChatRequest{})
	if err != nil {
		t.Fatalf("expected failover success: %v", err)
	}
	var text string
	for chunk := range ch {
		if chunk.Done {
			break
		}
		text += chunk.Content
	}
	if text != "Halo dunia" {
		t.Fatalf("expected secondary content, got %q", text)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("failover took %v, expected < 1s", elapsed)
	}
}

// TestIntegration_DeadModelQuarantinedAndFailsOver verifies that a model which
// keeps returning 503 (dead model) stops being hit and traffic fails over.
//
// NOTE (smart failover): an HTTP 503 is a MODEL-level failure, not an
// endpoint-level one — the gateway is clearly up and answering, it just cannot
// serve this model. So the ROUTE breaker quarantines the (endpoint, model) pair
// after routeBreakerFailureThreshold (2) failures, while the ENDPOINT breaker
// stays closed so healthy sibling models on the same endpoint keep serving.
// This replaces the older behaviour where 5 failures tripped the endpoint.
func TestIntegration_DeadModelQuarantinedAndFailsOver(t *testing.T) {
	var primaryHits int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&primaryHits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer primary.Close()
	secondary := sseServer(t)

	ep1 := &domain.AIEndpoint{ID: uuid.New(), Name: "p", BaseURL: primary.URL, Provider: "9router", Status: "active"}
	ep2 := &domain.AIEndpoint{ID: uuid.New(), Name: "s", BaseURL: secondary.URL, Provider: "9router", Status: "active"}
	repo := &fakeRepo{routes: []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep1.ID, Model: "m", Priority: 1, IsActive: true, TimeoutMS: 2000, Endpoint: ep1},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep2.ID, Model: "m", Priority: 2, IsActive: true, TimeoutMS: 2000, Endpoint: ep2},
	}}
	svc := New(repo, Config{
		DefaultTimeout: 2 * time.Second,
		Breaker:        breaker.Config{FailureThreshold: 5, Cooldown: time.Hour, HalfOpenMaxProbes: 1},
	})

	// Each call: primary fails, secondary succeeds. After the route threshold the
	// primary route is quarantined and no longer hit.
	for i := range 4 {
		ch, err := svc.ChatStream(context.Background(), &domain.AIChatRequest{})
		if err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
		for range ch {
		}
	}
	// The dead model was attempted exactly routeBreakerFailureThreshold times.
	if got := atomic.LoadInt32(&primaryHits); got != int32(routeBreakerFailureThreshold) {
		t.Fatalf("expected %d primary hits before quarantine, got %d", routeBreakerFailureThreshold, got)
	}

	// Further calls must not touch the dead model again.
	ch, err := svc.ChatStream(context.Background(), &domain.AIChatRequest{})
	if err != nil {
		t.Fatalf("call after quarantine: %v", err)
	}
	for range ch {
	}
	if got := atomic.LoadInt32(&primaryHits); got != int32(routeBreakerFailureThreshold) {
		t.Fatalf("quarantined route must not be hit; hits=%d", got)
	}

	// The endpoint must NOT be quarantined: the endpoint is up, only the model is bad.
	if svc.Breaker().State(ep1.ID.String()) != string(breaker.StateClosed) {
		t.Fatalf("endpoint breaker must stay closed for a model-level 503, got %s",
			svc.Breaker().State(ep1.ID.String()))
	}
	// The route itself is quarantined.
	if svc.RouteState("chat", ep1.ID.String(), "m", "") != "open" {
		t.Fatal("the dead model's route must be quarantined")
	}
}

// TestIntegration_TimeoutEnforced verifies a slow upstream triggers failover.
func TestIntegration_TimeoutEnforced(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slow.Close()
	secondary := sseServer(t)

	ep1 := &domain.AIEndpoint{ID: uuid.New(), Name: "slow", BaseURL: slow.URL, Provider: "9router", Status: "active"}
	ep2 := &domain.AIEndpoint{ID: uuid.New(), Name: "fast", BaseURL: secondary.URL, Provider: "9router", Status: "active"}
	repo := &fakeRepo{routes: []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep1.ID, Model: "m", Priority: 1, IsActive: true, TimeoutMS: 100, Endpoint: ep1},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep2.ID, Model: "m", Priority: 2, IsActive: true, TimeoutMS: 2000, Endpoint: ep2},
	}}
	svc := New(repo, Config{DefaultTimeout: 2 * time.Second})

	ch, err := svc.ChatStream(context.Background(), &domain.AIChatRequest{})
	if err != nil {
		t.Fatalf("expected failover after timeout: %v", err)
	}
	var text string
	for chunk := range ch {
		if chunk.Done {
			break
		}
		text += chunk.Content
	}
	if text != "Halo dunia" {
		t.Fatalf("expected secondary content after timeout, got %q", text)
	}
}

// TestIntegration_HalfOpenRecovery verifies cooldown -> half-open -> closed.
func TestIntegration_HalfOpenRecovery(t *testing.T) {
	b := breaker.New(breaker.Config{FailureThreshold: 5, Cooldown: 20 * time.Millisecond, HalfOpenMaxProbes: 1})
	id := "endpoint-x"

	for range 5 {
		b.RecordFailure(id)
	}
	if b.State(id) != string(breaker.StateOpen) {
		t.Fatal("expected open")
	}
	time.Sleep(30 * time.Millisecond)
	if !b.Allow(id) {
		t.Fatal("expected probe allowed after cooldown")
	}
	if b.State(id) != string(breaker.StateHalfOpen) {
		t.Fatalf("expected half_open, got %s", b.State(id))
	}
	b.RecordSuccess(id)
	if b.State(id) != string(breaker.StateClosed) {
		t.Fatalf("expected closed after probe success, got %s", b.State(id))
	}
}
