package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// benchServer streams a minimal SSE response for benchmarks/tests.
func benchServer(b testing.TB) *httptest.Server {
	b.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{}}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1,\"total_tokens\":2}}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	b.Cleanup(srv.Close)
	return srv
}

// BenchmarkChatStream measures end-to-end streaming throughput/latency through
// the gateway (transport + breaker + metering overhead).
func BenchmarkChatStream(b *testing.B) {
	srv := benchServer(b)
	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "b", BaseURL: srv.URL, Provider: "9router", Status: "active"}
	repo := &fakeRepo{routes: []*domain.AIRoute{{
		ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID,
		Model: "m", Priority: 1, IsActive: true, TimeoutMS: 2000, Endpoint: ep,
	}}}
	svc := New(repo, Config{DefaultTimeout: 2 * time.Second})
	req := &domain.AIChatRequest{Messages: []domain.AIChatMessage{{Role: "user", Content: "hi"}}}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		ch, err := svc.ChatStream(context.Background(), req)
		if err != nil {
			b.Fatalf("stream: %v", err)
		}
		for range ch {
		}
	}
}

// jsonServer returns a minimal non-streaming chat completion for benchmarks.
func jsonServer(b testing.TB) *httptest.Server {
	b.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	b.Cleanup(srv.Close)
	return srv
}

// BenchmarkChatSync measures non-streaming latency through the gateway.
func BenchmarkChatSync(b *testing.B) {
	srv := jsonServer(b)
	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "b", BaseURL: srv.URL, Provider: "9router", Status: "active"}
	repo := &fakeRepo{routes: []*domain.AIRoute{{
		ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID,
		Model: "m", Priority: 1, IsActive: true, TimeoutMS: 2000, Endpoint: ep,
	}}}
	svc := New(repo, Config{DefaultTimeout: 2 * time.Second})
	req := &domain.AIChatRequest{Messages: []domain.AIChatMessage{{Role: "user", Content: "hi"}}}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, _, err := svc.ChatSync(context.Background(), req); err != nil {
			b.Fatalf("sync: %v", err)
		}
	}
}

// TestConcurrentStreamsNoLeak runs many concurrent streams and asserts none
// hang (a proxy for goroutine/connection leaks in SSE handling).
func TestConcurrentStreamsNoLeak(t *testing.T) {
	srv := benchServer(t)
	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "b", BaseURL: srv.URL, Provider: "9router", Status: "active"}
	repo := &fakeRepo{routes: []*domain.AIRoute{{
		ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID,
		Model: "m", Priority: 1, IsActive: true, TimeoutMS: 5000, Endpoint: ep,
	}}}
	svc := New(repo, Config{DefaultTimeout: 5 * time.Second})

	const workers = 50
	done := make(chan struct{}, workers)
	for range workers {
		go func() {
			defer func() { done <- struct{}{} }()
			ch, err := svc.ChatStream(context.Background(), &domain.AIChatRequest{})
			if err != nil {
				return
			}
			for range ch {
			}
		}()
	}

	timeout := time.After(10 * time.Second)
	for range workers {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("concurrent streams did not complete — possible leak/hang")
		}
	}
}
