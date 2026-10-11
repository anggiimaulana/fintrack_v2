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

// TestIntegration_BreakerRejectsSixth verifies the 6th request is fast-failed
// without hitting the primary endpoint (5 consecutive failures trip it).
func TestIntegration_BreakerRejectsSixth(t *testing.T) {
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

	// 5 calls: primary fails each time, secondary succeeds.
	for i := range 5 {
		ch, err := svc.ChatStream(context.Background(), &domain.AIChatRequest{})
		if err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
		for range ch {
		}
	}
	if got := atomic.LoadInt32(&primaryHits); got != 5 {
		t.Fatalf("expected 5 primary hits, got %d", got)
	}

	// 6th call: breaker open -> primary skipped.
	ch, err := svc.ChatStream(context.Background(), &domain.AIChatRequest{})
	if err != nil {
		t.Fatalf("6th call: %v", err)
	}
	for range ch {
	}
	if got := atomic.LoadInt32(&primaryHits); got != 5 {
		t.Fatalf("6th call must not hit primary; hits=%d", got)
	}
	if svc.Breaker().State(ep1.ID.String()) != string(breaker.StateOpen) {
		t.Fatalf("expected breaker open, got %s", svc.Breaker().State(ep1.ID.String()))
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
