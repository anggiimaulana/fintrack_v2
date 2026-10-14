package gateway

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/fintrack/backend/internal/domain"
)

// Smart auto-failover: the endpoint breaker quarantines a whole ENDPOINT, which
// is right when the endpoint itself is down (connection refused, everything 5xx).
// It is wrong when only ONE MODEL is dead: the gateway (e.g. 9router) still
// answers for other models on the same endpoint. The measured case that
// motivated this: `gemini-2.5-flash` returned 503 while `gemini/gemini-3.5-flash`
// on the SAME endpoint returned 200. Endpoint-level tripping would quarantine a
// perfectly healthy endpoint and take down every sibling route with it.
//
// routeHealth tracks failures per route IDENTITY (use_case + endpoint + model +
// combo) and skips just that route while it is cooling down. Recovery is
// automatic: once the cooldown elapses the route is tried again, and a success
// clears its failure window.
//
// It deliberately avoids half-open "probe slots": a route can be allowed by the
// filter and then never attempted (a higher-priority sibling succeeded first),
// which would leak a slot and wedge the route forever. A simple cooldown window
// has no such state to leak.

// Route breaker defaults. They are more aggressive than the endpoint breaker: a
// model that fails twice in a row is almost certainly misconfigured or
// decommissioned, and skipping it is cheap because a sibling route is one hop away.
const (
	routeBreakerFailureThreshold = 2
	routeBreakerCooldown         = 60 * time.Second
)

type routeHealthEntry struct {
	failures  int
	openUntil time.Time
	// lastReason is the most recent failure message, surfaced in admin alerts.
	lastReason string
}

// routeHealth is a per-route failure tracker with a cooldown window.
type routeHealth struct {
	threshold int
	cooldown  time.Duration
	now       func() time.Time
	// onOpen is called once when a route transitions into quarantine.
	onOpen func(key string)
	// onRecover is called once when a quarantined route is cleared.
	onRecover func(key string)

	mu      sync.Mutex
	entries map[string]*routeHealthEntry
}

func newRouteHealth(threshold int, cooldown time.Duration) *routeHealth {
	if threshold <= 0 {
		threshold = routeBreakerFailureThreshold
	}
	if cooldown <= 0 {
		cooldown = routeBreakerCooldown
	}
	return &routeHealth{
		threshold: threshold,
		cooldown:  cooldown,
		now:       time.Now,
		entries:   make(map[string]*routeHealthEntry),
	}
}

// withClock overrides the time source (tests).
func (r *routeHealth) withClock(now func() time.Time) *routeHealth {
	r.now = now
	return r
}

// setThresholds replaces the tuning parameters at runtime (Super Admin settings
// change) without dropping quarantined routes. Only positive values apply.
func (r *routeHealth) setThresholds(threshold int, cooldown time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if threshold > 0 {
		r.threshold = threshold
	}
	if cooldown > 0 {
		r.cooldown = cooldown
	}
}

// thresholds returns the current tuning parameters.
func (r *routeHealth) thresholds() (int, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.threshold, r.cooldown
}

// withCallbacks installs transition callbacks (fired outside the lock).
func (r *routeHealth) withCallbacks(onOpen, onRecover func(key string)) *routeHealth {
	r.onOpen = onOpen
	r.onRecover = onRecover
	return r
}

// blocked reports whether the route should be skipped right now. When the
// cooldown has elapsed the entry is cleared, so the route is retried (and will
// re-open on failure if the model is still dead).
func (r *routeHealth) blocked(key string) bool {
	if key == "" {
		return false
	}
	r.mu.Lock()
	e := r.entries[key]
	if e == nil || e.openUntil.IsZero() {
		r.mu.Unlock()
		return false
	}
	if r.now().Before(e.openUntil) {
		r.mu.Unlock()
		return true
	}
	// Cooldown elapsed: allow a retry with a clean slate.
	delete(r.entries, key)
	r.mu.Unlock()

	// Notify outside the lock (the callback may call back into the tracker).
	r.fire(r.onRecover, key)
	return false
}

// fail records a failure, opening the route for the cooldown once the threshold
// is reached. Further failures while open extend nothing (the window stays).
// reason is the upstream error, kept for the admin alert.
func (r *routeHealth) fail(key, reason string) {
	if key == "" {
		return
	}
	r.mu.Lock()
	e := r.entries[key]
	if e == nil {
		e = &routeHealthEntry{}
		r.entries[key] = e
	}
	if reason != "" {
		e.lastReason = reason
	}
	if !e.openUntil.IsZero() && r.now().Before(e.openUntil) {
		r.mu.Unlock()
		return // already open and cooling; nothing to change
	}
	e.failures++
	opened := false
	if e.failures >= r.threshold {
		if e.openUntil.IsZero() {
			opened = true // transition closed -> open
		}
		e.openUntil = r.now().Add(r.cooldown)
	}
	r.mu.Unlock()

	if opened {
		r.fire(r.onOpen, key)
	}
}

// lastReason returns the last recorded failure reason for a route.
func (r *routeHealth) lastReason(key string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := r.entries[key]; e != nil {
		return e.lastReason
	}
	return ""
}

// ok clears the failure window for a route (a success proves it works).
func (r *routeHealth) ok(key string) {
	if key == "" {
		return
	}
	r.mu.Lock()
	e := r.entries[key]
	wasOpen := e != nil && !e.openUntil.IsZero() && r.now().Before(e.openUntil)
	delete(r.entries, key)
	r.mu.Unlock()

	if wasOpen {
		r.fire(r.onRecover, key)
	}
}

// fire invokes a transition callback outside the lock, recovering from panics
// so a misbehaving observer can never break the request path.
func (r *routeHealth) fire(cb func(string), key string) {
	if cb == nil {
		return
	}
	func() {
		defer func() { _ = recover() }()
		cb(key)
	}()
}

// state reports "open" while cooling, otherwise "closed" (diagnostics).
func (r *routeHealth) state(key string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.entries[key]
	if e != nil && !e.openUntil.IsZero() && r.now().Before(e.openUntil) {
		return "open"
	}
	return "closed"
}

// reset forcibly clears a route's window (admin action).
func (r *routeHealth) reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, key)
}

// routeKey is the stable identity of a route for the route breaker. It must not
// collide between distinct (endpoint, model) pairs on the same use case.
func routeKey(rt *domain.AIRoute) string {
	if rt == nil || rt.Endpoint == nil {
		return ""
	}
	model := rt.Model
	if model == "" {
		model = rt.Endpoint.Provider
	}
	return routeKeyParts(string(rt.UseCase), rt.Endpoint.ID.String(), model, rt.Combo)
}

// routeKeyParts builds the same key from raw fields (used when only a
// verification result is available, not a *domain.AIRoute).
func routeKeyParts(useCase, endpointID, model, combo string) string {
	if endpointID == "" {
		return ""
	}
	if model == "" {
		model = "?"
	}
	combo = strings.TrimSpace(combo)
	if combo == "" {
		combo = "default"
	}
	return useCase + "|" + endpointID + "|" + model + "|" + combo
}

// parseRouteKey splits a route key back into its parts. The endpoint id is a
// UUID (no "|"), so a 4-way split is unambiguous.
func parseRouteKey(key string) (useCase, endpointID, model, combo string, ok bool) {
	parts := strings.SplitN(key, "|", 4)
	if len(parts) != 4 {
		return "", "", "", "", false
	}
	return parts[0], parts[1], parts[2], parts[3], true
}

// withRouteCallbacks installs the quarantine/recovery observer, translating the
// opaque route key back into a human-meaningful RouteStateChange.
func (s *Service) withRouteCallbacks(cfg Config) {
	onOpen := func(key string) {
		if cfg.OnRouteStateChange == nil {
			return
		}
		useCase, epID, model, combo, ok := parseRouteKey(key)
		if !ok {
			return
		}
		cfg.OnRouteStateChange(RouteStateChange{
			UseCase:    useCase,
			Model:      model,
			EndpointID: epID,
			Endpoint:   s.endpointName(epID),
			Combo:      combo,
			Opened:     true,
			Reason:     s.lastReason(key),
		})
	}
	onRecover := func(key string) {
		if cfg.OnRouteStateChange == nil {
			return
		}
		useCase, epID, model, combo, ok := parseRouteKey(key)
		if !ok {
			return
		}
		cfg.OnRouteStateChange(RouteStateChange{
			UseCase:    useCase,
			Model:      model,
			EndpointID: epID,
			Endpoint:   s.endpointName(epID),
			Combo:      combo,
			Opened:     false,
		})
	}
	s.routeHealth.withCallbacks(onOpen, onRecover)
}

// routeAllowed reports whether the route may be attempted (route breaker).
func (s *Service) routeAllowed(rt *domain.AIRoute) bool {
	return !s.routeHealth.blocked(routeKey(rt))
}

// recordRouteFailure trips the route breaker for a failed route. reason is the
// upstream error, kept for the admin alert.
func (s *Service) recordRouteFailure(rt *domain.AIRoute, reason string) {
	s.routeHealth.fail(routeKey(rt), reason)
}

// recordRouteSuccess closes the route breaker for a healthy route.
func (s *Service) recordRouteSuccess(rt *domain.AIRoute) {
	s.routeHealth.ok(routeKey(rt))
}

// QuarantineBrokenRoutes trips the route breaker for every route that failed
// verification, so the next request fails over immediately instead of hitting a
// known-dead model. Recovery is automatic once the cooldown elapses.
//
// Only routes with a concrete failure are quarantined: skipped routes (voice,
// which cannot be probed) are ignored. Returns how many routes were quarantined.
func (s *Service) QuarantineBrokenRoutes(results []RouteVerification) int {
	n := 0
	for _, r := range results {
		if r.OK || r.Skipped || r.EndpointID == "" {
			continue
		}
		key := routeKeyParts(r.UseCase, r.EndpointID, r.Model, r.Combo)
		if key == "" {
			continue
		}
		// Trip it fully so the route is skipped for the whole cooldown, not just
		// after `threshold` more failures.
		s.routeHealth.fail(key, r.Error)
		s.routeHealth.fail(key, r.Error)
		n++
	}
	return n
}

// RouteState reports a route's breaker state ("open"|"closed") for diagnostics.
func (s *Service) RouteState(useCase, endpointID, model, combo string) string {
	return s.routeHealth.state(routeKeyParts(useCase, endpointID, model, combo))
}

// lastReason exposes the last recorded failure reason for a route key.
func (s *Service) lastReason(key string) string {
	return s.routeHealth.lastReason(key)
}

// endpointName resolves an endpoint id to its display name, best-effort. It uses
// the cached route pool (no extra DB query on the request path); an unknown id
// yields "".
func (s *Service) endpointName(endpointID string) string {
	for _, uc := range []domain.UseCase{
		domain.UseCaseChat, domain.UseCaseReasoning, domain.UseCaseVision,
		domain.UseCaseVoice, domain.UseCaseEmbedding,
	} {
		routes, err := s.repo.GetActiveRoutesByUseCase(context.Background(), uc)
		if err != nil {
			continue
		}
		for _, rt := range routes {
			if rt.Endpoint != nil && rt.Endpoint.ID.String() == endpointID {
				return rt.Endpoint.Name
			}
		}
	}
	return ""
}

// ResetRoute clears a route's breaker window (admin action).
func (s *Service) ResetRoute(useCase, endpointID, model, combo string) {
	s.routeHealth.reset(routeKeyParts(useCase, endpointID, model, combo))
}
