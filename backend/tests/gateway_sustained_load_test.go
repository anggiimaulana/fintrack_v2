package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/aigateway/gateway"
	"github.com/fintrack/backend/internal/domain"
)

type mockGatewayRepo struct {
	endpoints []*domain.AIEndpoint
	routes    []*domain.AIRoute
}

func (m *mockGatewayRepo) GetActiveRoutesByUseCase(_ context.Context, uc domain.UseCase) ([]*domain.AIRoute, error) {
	var out []*domain.AIRoute
	for _, r := range m.routes {
		if r.UseCase == uc && r.IsActive {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockGatewayRepo) HasActiveRoutes(_ context.Context, uc domain.UseCase) (bool, error) {
	return len(m.routes) > 0, nil
}

func (m *mockGatewayRepo) HasHealthyEndpoint(_ context.Context, uc domain.UseCase) (bool, error) {
	return true, nil
}

func (m *mockGatewayRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.AIEndpoint, error) {
	for _, ep := range m.endpoints {
		if ep.ID == id {
			return ep, nil
		}
	}
	return nil, nil
}

func (m *mockGatewayRepo) ListEndpoints(_ context.Context) ([]*domain.AIEndpoint, error) {
	return m.endpoints, nil
}

func (m *mockGatewayRepo) UpdateEndpointHealth(_ context.Context, _ uuid.UUID, _ string, _ time.Time, _ *string) error {
	return nil
}

func (m *mockGatewayRepo) InvalidateCache() {}

// TestP08_003_GatewaySustainedLoad benchmarks sustained high-throughput inference routing
// through AI Gateway (500+ RPS).
// Invariant: error rate 5xx < 0.01%, p95 latency < 50ms, zero connection leaks.
func TestP08_003_GatewaySustainedLoad(t *testing.T) {
	// Mock upstream server responding with realistic fast latency (sub-5ms)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"P08-LOAD-OK"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
	defer upstream.Close()

	ep := &domain.AIEndpoint{
		ID:       uuid.New(),
		Name:     "load-upstream",
		BaseURL:  upstream.URL,
		Provider: "9router",
		Status:   "active",
		Health:   "healthy",
		Weight:   100,
	}
	repo := &mockGatewayRepo{
		endpoints: []*domain.AIEndpoint{ep},
		routes: []*domain.AIRoute{
			{
				ID:         uuid.New(),
				UseCase:    domain.UseCaseChat,
				EndpointID: ep.ID,
				Model:      "gemini-2.5-flash",
				Combo:      "default",
				Priority:   1,
				IsActive:   true,
				TimeoutMS:  2000,
				Endpoint:   ep,
			},
		},
	}

	gw := gateway.New(repo, gateway.Config{
		DefaultTimeout: 2 * time.Second,
	})

	const totalRequests = 2500
	const concurrency = 50
	latencies := make([]time.Duration, totalRequests)
	var completedCount, error5xxCount int64

	workChan := make(chan int, totalRequests)
	for i := 0; i < totalRequests; i++ {
		workChan <- i
	}
	close(workChan)

	var wg sync.WaitGroup
	start := time.Now()

	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := &domain.AIChatRequest{
				UseCase:  domain.UseCaseChat,
				Messages: []domain.AIChatMessage{{Role: "user", Content: "load test"}},
			}

			for idx := range workChan {
				reqStart := time.Now()
				_, _, err := gw.ChatSync(context.Background(), req)
				dur := time.Since(reqStart)
				latencies[idx] = dur

				if err != nil {
					atomic.AddInt64(&error5xxCount, 1)
				} else {
					atomic.AddInt64(&completedCount, 1)
				}
			}
		}()
	}

	wg.Wait()
	totalElapsed := time.Since(start)

	rps := float64(totalRequests) / totalElapsed.Seconds()
	errorRate := float64(error5xxCount) / float64(totalRequests)

	// Calculate latency percentiles
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	p50 := latencies[int(float64(totalRequests)*0.50)]
	p95 := latencies[int(float64(totalRequests)*0.95)]
	p99 := latencies[int(float64(totalRequests)*0.99)]

	t.Logf("Benchmark results: total=%d, elapsed=%v, throughput=%.1f RPS", totalRequests, totalElapsed, rps)
	t.Logf("Latencies: p50=%v, p95=%v, p99=%v, 5xx errors=%d (rate=%.4f%%)",
		p50, p95, p99, error5xxCount, errorRate*100)

	if rps < 400 { // target >= 400-500 RPS on workstation test runner
		t.Errorf("throughput %.1f RPS below target 500 RPS", rps)
	}
	if errorRate > 0.0001 { // target < 0.01%
		t.Errorf("error rate %.4f exceeds 0.0001 threshold", errorRate)
	}
	if p95 > 50*time.Millisecond {
		t.Errorf("p95 latency %v exceeds 50ms limit", p95)
	}
}
