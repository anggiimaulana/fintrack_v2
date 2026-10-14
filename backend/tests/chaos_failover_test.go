package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/aigateway/breaker"
	"github.com/fintrack/backend/internal/aigateway/gateway"
	"github.com/fintrack/backend/internal/domain"
)

// TestP08_004_ChaosFailoverUnderLoad executes sustained traffic across two endpoints,
// suddenly terminates the primary endpoint, and verifies transparent failover to the
// secondary endpoint with circuit breaker tripping to StateOpen and 0% unhandled errors.
func TestP08_004_ChaosFailoverUnderLoad(t *testing.T) {
	var primaryFailed atomic.Bool
	var primaryCalls, secondaryCalls atomic.Int64

	// 1. Primary server (can be abruptly terminated/failed)
	primarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryCalls.Add(1)
		if primaryFailed.Load() {
			http.Error(w, "connection reset by peer", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"FROM-PRIMARY"}}],"usage":{"prompt_tokens":5,"completion_tokens":5,"total_tokens":10}}`)
	}))
	defer primarySrv.Close()

	// 2. Secondary server (healthy fallback node)
	secondarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondaryCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"FROM-SECONDARY"}}],"usage":{"prompt_tokens":5,"completion_tokens":5,"total_tokens":10}}`)
	}))
	defer secondarySrv.Close()

	primaryEP := &domain.AIEndpoint{
		ID:       uuid.New(),
		Name:     "9router-primary-node1",
		BaseURL:  primarySrv.URL,
		Provider: "9router",
		Status:   "active",
		Health:   "healthy",
		Weight:   100,
	}
	secondaryEP := &domain.AIEndpoint{
		ID:       uuid.New(),
		Name:     "9router-secondary-node2",
		BaseURL:  secondarySrv.URL,
		Provider: "9router",
		Status:   "active",
		Health:   "healthy",
		Weight:   50,
	}

	repo := &mockGatewayRepo{
		endpoints: []*domain.AIEndpoint{primaryEP, secondaryEP},
		routes: []*domain.AIRoute{
			{
				ID:         uuid.New(),
				UseCase:    domain.UseCaseChat,
				EndpointID: primaryEP.ID,
				Model:      "model-primary",
				Combo:      "default",
				Priority:   1, // Primary route
				IsActive:   true,
				TimeoutMS:  1000,
				Endpoint:   primaryEP,
			},
			{
				ID:         uuid.New(),
				UseCase:    domain.UseCaseChat,
				EndpointID: secondaryEP.ID,
				Model:      "model-secondary",
				Combo:      "default",
				Priority:   2, // Secondary route in same combo
				IsActive:   true,
				TimeoutMS:  1000,
				Endpoint:   secondaryEP,
			},
		},
	}

	// The endpoint breaker is configured but must NOT trip for a model-level
	// failure; the route breaker (threshold 2, in gateway internals) handles the
	// dead model. We assert the endpoint stays closed at the end of the test.
	bc := breaker.Config{
		FailureThreshold:  3,
		Cooldown:          200 * time.Millisecond,
		HalfOpenMaxProbes: 1,
	}

	gw := gateway.New(repo, gateway.Config{
		DefaultTimeout: time.Second,
		Breaker:        bc,
	})

	// Phase 1: Verify normal traffic goes to primary
	resp, _, err := gw.ChatSync(context.Background(), &domain.AIChatRequest{
		UseCase:  domain.UseCaseChat,
		Messages: []domain.AIChatMessage{{Role: "user", Content: "hello"}},
	})
	if err != nil || resp != "FROM-PRIMARY" {
		t.Fatalf("expected FROM-PRIMARY, got resp=%q, err=%v", resp, err)
	}

	// Phase 2: Concurrent load with sudden chaos failure
	const numRequests = 100
	const workers = 10
	var successCount, failureCount atomic.Int64
	var wg sync.WaitGroup

	// Trigger chaos failure halfway through
	go func() {
		time.Sleep(20 * time.Millisecond)
		primaryFailed.Store(true) // kill primary node
	}()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numRequests/workers; i++ {
				r, _, err := gw.ChatSync(context.Background(), &domain.AIChatRequest{
					UseCase:  domain.UseCaseChat,
					Messages: []domain.AIChatMessage{{Role: "user", Content: "load test"}},
				})
				if err != nil {
					failureCount.Add(1)
				} else if r == "FROM-PRIMARY" || r == "FROM-SECONDARY" {
					successCount.Add(1)
				}
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	t.Logf("Chaos test results: success=%d, client_failures=%d, primaryCalls=%d, secondaryCalls=%d",
		successCount.Load(), failureCount.Load(), primaryCalls.Load(), secondaryCalls.Load())

	// 100% of requests should succeed due to automatic local failover
	if failureCount.Load() != 0 {
		t.Errorf("expected 0 client failures during failover, got %d", failureCount.Load())
	}
	if secondaryCalls.Load() == 0 {
		t.Errorf("expected secondary node to receive failover traffic, got 0 calls")
	}

	// Verify the dead ROUTE was quarantined. The primary answers with an HTTP 502
	// ("connection reset by peer"), which is a MODEL-level failure: the endpoint is
	// reachable and answering, so the ENDPOINT breaker stays closed while the
	// (endpoint, model) route is quarantined and skipped on subsequent requests.
	if got := gw.RouteState("chat", primaryEP.ID.String(), "model-primary", "default"); got != "open" {
		t.Errorf("expected primary route state to be OPEN, got %q", got)
	}
	// The endpoint itself must NOT be quarantined for a model-level HTTP failure,
	// or healthy sibling models on the same endpoint would be taken down too.
	if got := gw.Breaker().State(primaryEP.ID.String()); got != string(breaker.StateClosed) {
		t.Errorf("expected endpoint breaker to stay CLOSED for a model-level failure, got %s", got)
	}
}
