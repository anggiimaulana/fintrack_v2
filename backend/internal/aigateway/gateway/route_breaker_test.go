package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/aigateway/breaker"
	"github.com/fintrack/backend/internal/aigateway/client"
	"github.com/fintrack/backend/internal/domain"
)

// modelHits is a concurrency-safe per-model hit counter.
type modelHits struct {
	mu sync.Mutex
	m  map[string]int32
}

func newModelHits() *modelHits { return &modelHits{m: map[string]int32{}} }

func (h *modelHits) add(model string) {
	h.mu.Lock()
	h.m[model]++
	h.mu.Unlock()
}

func (h *modelHits) get(model string) int32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.m[model]
}

// modelAwareServer answers per-model: models listed in dead return 503, every
// other model returns 200. This models the real 9router behaviour that motivated
// smart failover: one dead model on an otherwise healthy endpoint.
func modelAwareServer(t *testing.T, dead ...string) (*httptest.Server, *modelHits) {
	t.Helper()
	deadSet := map[string]bool{}
	for _, m := range dead {
		deadSet[m] = true
	}
	hits := newModelHits()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		hits.add(body.Model)
		if deadSet[body.Model] {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": "ok"}}},
			"usage":   map[string]any{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4},
		})
	}))
	t.Cleanup(srv.Close)
	return srv, hits
}

// ─── route breaker unit tests ─────────────────────────────────

func TestRouteHealth_BlocksAfterThreshold(t *testing.T) {
	rh := newRouteHealth(2, time.Minute)
	key := "chat|ep|model|default"

	if rh.blocked(key) {
		t.Fatal("fresh route must not be blocked")
	}
	rh.fail(key, "boom")
	if rh.blocked(key) {
		t.Fatal("one failure must not block (threshold=2)")
	}
	rh.fail(key, "boom")
	if !rh.blocked(key) {
		t.Fatal("route must be blocked after reaching the threshold")
	}
}

func TestRouteHealth_RecoversAfterCooldown(t *testing.T) {
	now := time.Unix(1000, 0)
	rh := newRouteHealth(1, 30*time.Second).withClock(func() time.Time { return now })
	key := "chat|ep|m|default"

	rh.fail(key, "boom")
	if !rh.blocked(key) {
		t.Fatal("route must be blocked after failure")
	}
	// Still cooling.
	now = now.Add(10 * time.Second)
	if !rh.blocked(key) {
		t.Fatal("route must stay blocked within the cooldown")
	}
	// Cooldown elapsed: allowed again.
	now = now.Add(25 * time.Second)
	if rh.blocked(key) {
		t.Fatal("route must be retried after the cooldown elapses")
	}
}

func TestRouteHealth_SuccessClearsWindow(t *testing.T) {
	rh := newRouteHealth(2, time.Minute)
	key := "chat|ep|m|default"
	rh.fail(key, "boom")
	rh.fail(key, "boom")
	if !rh.blocked(key) {
		t.Fatal("expected blocked")
	}
	rh.ok(key)
	if rh.blocked(key) {
		t.Fatal("success must clear the failure window")
	}
}

func TestRouteHealth_KeysAreDistinctPerModel(t *testing.T) {
	epID := uuid.New().String()
	k1 := routeKeyParts("chat", epID, "model-a", "default")
	k2 := routeKeyParts("chat", epID, "model-b", "default")
	if k1 == k2 {
		t.Fatalf("different models on the same endpoint must not share a key: %q", k1)
	}
	if routeKeyParts("chat", epID, "m", "a") == routeKeyParts("chat", epID, "m", "b") {
		t.Fatal("different combos must not share a key")
	}
	if routeKeyParts("chat", epID, "m", "") != routeKeyParts("chat", epID, "m", "default") {
		t.Fatal("empty combo must normalize to default")
	}
}

// ─── the core regression: a dead model must not kill its healthy sibling ───

// TestSmartFailover_DeadModelDoesNotQuarantineHealthySibling is the whole point.
// Two models share ONE endpoint; model A is dead (503), model B is healthy.
// After A fails, the gateway must keep serving via B AND must NOT trip the
// endpoint breaker (which would kill B too).
func TestSmartFailover_DeadModelDoesNotQuarantineHealthySibling(t *testing.T) {
	srv, hits := modelAwareServer(t, "dead-model")
	ep := &domain.AIEndpoint{
		ID: uuid.New(), Name: "shared", BaseURL: srv.URL,
		Provider: "9router", Status: "active", Health: "healthy",
	}
	routes := []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID, Model: "dead-model", Combo: "default", Priority: 1, IsActive: true, Endpoint: ep},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID, Model: "good-model", Combo: "default", Priority: 2, IsActive: true, Endpoint: ep},
	}
	svc := New(&fakeRepo{routes: routes}, Config{
		DefaultTimeout: 2 * time.Second,
		Breaker:        breaker.Config{FailureThreshold: 2, Cooldown: time.Hour, HalfOpenMaxProbes: 1},
	})

	// First two calls: dead-model fails (trip threshold = 2), good-model serves.
	for range 2 {
		if _, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{}); err != nil {
			t.Fatalf("call must succeed via healthy sibling: %v", err)
		}
	}

	// The endpoint must still be CLOSED: only the model was bad, not the endpoint.
	if state := svc.breaker.State(ep.ID.String()); state != "closed" {
		t.Fatalf("endpoint breaker must stay closed for a model-level failure, got %q", state)
	}

	deadHitsBefore := hits.get("dead-model")

	// Subsequent calls: dead-model is quarantined, so it must NOT be hit again.
	for range 3 {
		if _, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{}); err != nil {
			t.Fatalf("call must keep succeeding via sibling: %v", err)
		}
	}
	if got := hits.get("dead-model"); got != deadHitsBefore {
		t.Fatalf("quarantined dead model must not be retried: hits %d -> %d", deadHitsBefore, got)
	}
	if hits.get("good-model") == 0 {
		t.Fatal("healthy sibling must keep serving")
	}
}

// TestSmartFailover_RouteRecoversAfterCooldown proves automatic recovery: once
// the cooldown elapses the previously-dead model is retried, and a success
// un-quarantines it.
func TestSmartFailover_RouteRecoversAfterCooldown(t *testing.T) {
	// The upstream is dead for the first two real calls, then healthy.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": "ok"}}},
			"usage":   map[string]any{"total_tokens": 4},
		})
	}))
	t.Cleanup(srv.Close)

	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "ep", BaseURL: srv.URL, Provider: "9router", Status: "active", Health: "healthy"}
	rt := &domain.AIRoute{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID, Model: "m", Combo: "default", Priority: 1, IsActive: true, Endpoint: ep}
	svc := New(&fakeRepo{routes: []*domain.AIRoute{rt}}, Config{DefaultTimeout: 2 * time.Second})

	now := time.Unix(0, 0)
	svc.routeHealth.withClock(func() time.Time { return now })

	// Two real calls fail, tripping the route breaker (threshold = 2).
	for range 2 {
		if _, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{}); err == nil {
			t.Fatal("expected failure while the model is dead")
		}
	}
	if !svc.routeHealth.blocked(routeKey(rt)) {
		t.Fatal("route must be blocked after threshold failures")
	}

	// Before cooldown: the route is skipped entirely (no further upstream call).
	before := atomic.LoadInt32(&calls)
	if _, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{}); err == nil {
		t.Fatal("expected failure while the only route is quarantined")
	}
	if atomic.LoadInt32(&calls) != before {
		t.Fatal("quarantined route must not be attempted before cooldown")
	}

	// After cooldown: route is retried and now succeeds.
	now = now.Add(2 * time.Minute)
	if _, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{}); err != nil {
		t.Fatalf("route must recover after cooldown: %v", err)
	}
	if svc.routeHealth.blocked(routeKey(rt)) {
		t.Fatal("a successful retry must un-quarantine the route")
	}
}

// TestSmartFailover_TransportFailureTripsEndpointBreaker verifies the other
// direction: a genuine endpoint-down failure (connection refused) DOES trip the
// endpoint breaker, so all its routes are skipped.
func TestSmartFailover_TransportFailureTripsEndpointBreaker(t *testing.T) {
	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "down", BaseURL: "http://127.0.0.1:1/v1", Provider: "9router", Status: "active", Health: "healthy"}
	rt := &domain.AIRoute{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID, Model: "m", Priority: 1, IsActive: true, Endpoint: ep}
	svc := New(&fakeRepo{routes: []*domain.AIRoute{rt}}, Config{
		DefaultTimeout: 500 * time.Millisecond,
		Breaker:        breaker.Config{FailureThreshold: 1, Cooldown: time.Hour, HalfOpenMaxProbes: 1},
	})

	_, _, _ = svc.ChatSync(context.Background(), &domain.AIChatRequest{})
	if state := svc.breaker.State(ep.ID.String()); state != "open" {
		t.Fatalf("a transport failure must trip the endpoint breaker, got %q", state)
	}
}

// TestIsEndpointLevelFailure pins the classification that keeps the two breaker
// levels from interfering.
func TestIsEndpointLevelFailure(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil is not endpoint-level", nil, false},
		{"503 (dead model) is NOT endpoint-level", &client.UpstreamError{StatusCode: 503}, false},
		{"404 is NOT endpoint-level", &client.UpstreamError{StatusCode: 404}, false},
		{"400 is NOT endpoint-level", &client.UpstreamError{StatusCode: 400}, false},
		{"timeout is NOT endpoint-level", context.DeadlineExceeded, false},
		{"transport error IS endpoint-level", errors.New("dial tcp: connection refused"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isEndpointLevelFailure(tc.err); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestQuarantineBrokenRoutes verifies verification results trip the right routes.
func TestQuarantineBrokenRoutes(t *testing.T) {
	svc := New(&fakeRepo{}, Config{})
	epID := uuid.New().String()

	results := []RouteVerification{
		{UseCase: "chat", Model: "dead", EndpointID: epID, Combo: "default", OK: false, Error: "503"},
		{UseCase: "chat", Model: "good", EndpointID: epID, Combo: "default", OK: true},
		{UseCase: "voice", Model: "whisper-1", EndpointID: epID, Combo: "default", Skipped: true},
	}
	if n := svc.QuarantineBrokenRoutes(results); n != 1 {
		t.Fatalf("expected 1 quarantined route, got %d", n)
	}
	if svc.RouteState("chat", epID, "dead", "default") != "open" {
		t.Fatal("broken route must be quarantined")
	}
	if svc.RouteState("chat", epID, "good", "default") != "closed" {
		t.Fatal("healthy route must not be quarantined")
	}
	if svc.RouteState("voice", epID, "whisper-1", "default") != "closed" {
		t.Fatal("skipped route must not be quarantined")
	}
}

// TestQuarantineBrokenRoutes_MakesNextRequestSkipDeadModel proves the end-to-end
// value: after quarantine, a request never even attempts the dead model.
func TestQuarantineBrokenRoutes_MakesNextRequestSkipDeadModel(t *testing.T) {
	srv, hits := modelAwareServer(t, "dead")
	ep := &domain.AIEndpoint{ID: uuid.New(), Name: "ep", BaseURL: srv.URL, Provider: "9router", Status: "active", Health: "healthy"}
	routes := []*domain.AIRoute{
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID, Model: "dead", Combo: "default", Priority: 1, IsActive: true, Endpoint: ep},
		{ID: uuid.New(), UseCase: domain.UseCaseChat, EndpointID: ep.ID, Model: "good", Combo: "default", Priority: 2, IsActive: true, Endpoint: ep},
	}
	svc := New(&fakeRepo{routes: routes}, Config{DefaultTimeout: 2 * time.Second})

	svc.QuarantineBrokenRoutes([]RouteVerification{
		{UseCase: "chat", Model: "dead", EndpointID: ep.ID.String(), Combo: "default", OK: false},
	})

	if _, _, err := svc.ChatSync(context.Background(), &domain.AIChatRequest{}); err != nil {
		t.Fatalf("expected success via sibling: %v", err)
	}
	if got := hits.get("dead"); got != 0 {
		t.Fatalf("quarantined model must not be hit at all, got %d hits", got)
	}
	if hits.get("good") != 1 {
		t.Fatalf("healthy sibling must serve, got %d hits", hits.get("good"))
	}
}

// ─── quarantine/recovery event emission (admin alerts) ────────

// TestRouteStateChange_EmitsOnQuarantineAndRecovery verifies the observer fires
// exactly once on the closed->open transition and once when the route recovers,
// carrying the route identity so the admin alert is actionable.
func TestRouteStateChange_EmitsOnQuarantineAndRecovery(t *testing.T) {
	epID := uuid.New().String()
	var mu sync.Mutex
	var events []RouteStateChange

	svc := New(&fakeRepo{}, Config{
		OnRouteStateChange: func(ev RouteStateChange) {
			mu.Lock()
			events = append(events, ev)
			mu.Unlock()
		},
	})
	now := time.Unix(0, 0)
	svc.routeHealth.withClock(func() time.Time { return now })

	key := routeKeyParts("chat", epID, "dead-model", "default")

	// First failure: below threshold, no event.
	svc.routeHealth.fail(key, "503 dead")
	if len(events) != 0 {
		t.Fatalf("no event expected below threshold, got %d", len(events))
	}
	// Second failure: crosses threshold -> exactly one "opened" event.
	svc.routeHealth.fail(key, "503 dead")
	if len(events) != 1 {
		t.Fatalf("expected 1 quarantine event, got %d", len(events))
	}
	got := events[0]
	if !got.Opened || got.UseCase != "chat" || got.Model != "dead-model" || got.EndpointID != epID {
		t.Fatalf("unexpected quarantine event: %+v", got)
	}
	if got.Reason != "503 dead" {
		t.Fatalf("expected the failure reason, got %q", got.Reason)
	}

	// More failures while open must NOT re-emit.
	svc.routeHealth.fail(key, "503 dead")
	if len(events) != 1 {
		t.Fatalf("must not re-emit while already open, got %d", len(events))
	}

	// After cooldown, the route is retried (blocked() clears it) -> recovery event.
	now = now.Add(2 * time.Minute)
	_ = svc.routeHealth.blocked(key)
	if len(events) != 2 {
		t.Fatalf("expected a recovery event, got %d", len(events))
	}
	rec := events[1]
	if rec.Opened || rec.Model != "dead-model" {
		t.Fatalf("unexpected recovery event: %+v", rec)
	}
}

// TestRouteStateChange_SuccessAfterQuarantineEmitsRecovery verifies a successful
// retry clears the route AND emits exactly one recovery event.
func TestRouteStateChange_SuccessAfterQuarantineEmitsRecovery(t *testing.T) {
	epID := uuid.New().String()
	var mu sync.Mutex
	var events []RouteStateChange

	svc := New(&fakeRepo{}, Config{
		OnRouteStateChange: func(ev RouteStateChange) {
			mu.Lock()
			events = append(events, ev)
			mu.Unlock()
		},
	})
	key := routeKeyParts("chat", epID, "m", "default")
	svc.routeHealth.fail(key, "boom")
	svc.routeHealth.fail(key, "boom")

	svc.routeHealth.ok(key)
	mu.Lock()
	defer mu.Unlock()
	// Expected order: [open, recover].
	if len(events) != 2 || !events[0].Opened || events[1].Opened {
		t.Fatalf("expected [open, recover], got %+v", events)
	}
}

// TestRouteStateChange_NilObserverIsSafe verifies a nil observer never panics.
func TestRouteStateChange_NilObserverIsSafe(t *testing.T) {
	svc := New(&fakeRepo{}, Config{})
	key := routeKeyParts("chat", uuid.New().String(), "m", "default")
	svc.routeHealth.fail(key, "x")
	svc.routeHealth.fail(key, "x")
	svc.routeHealth.ok(key) // must not panic with no observer
}

// TestRouteHealth_SetThresholdsAtRuntime verifies the route breaker can be
// retuned without a restart and keeps its quarantined entries.
func TestRouteHealth_SetThresholdsAtRuntime(t *testing.T) {
	rh := newRouteHealth(2, time.Minute)
	key := "chat|ep|model|default"

	// Retune to threshold 3.
	rh.setThresholds(3, 2*time.Minute)
	th, cd := rh.thresholds()
	if th != 3 || cd != 2*time.Minute {
		t.Fatalf("thresholds = (%d, %v), want (3, 2m)", th, cd)
	}

	rh.fail(key, "boom")
	rh.fail(key, "boom")
	if rh.blocked(key) {
		t.Fatal("must not block before reaching the new threshold of 3")
	}
	rh.fail(key, "boom")
	if !rh.blocked(key) {
		t.Fatal("must block after reaching the retuned threshold of 3")
	}

	// A zero update must not clobber the current values.
	rh.setThresholds(0, 0)
	th, cd = rh.thresholds()
	if th != 3 || cd != 2*time.Minute {
		t.Fatalf("zero update clobbered thresholds: (%d, %v)", th, cd)
	}
}
