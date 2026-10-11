// Package breaker implements a per-endpoint circuit breaker used by the AI
// gateway to fail fast and fail over. States follow the classic model:
//
//	closed    -> requests flow; N consecutive failures trip the breaker open
//	open      -> requests are rejected immediately until the cooldown elapses
//	half_open -> exactly one probe is allowed; success closes, failure reopens
package breaker

import (
	"sync"
	"time"
)

// State is the breaker state for a single endpoint.
type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half_open"
)

// Config tunes a breaker set.
type Config struct {
	// FailureThreshold is the number of consecutive failures that trips the
	// breaker open (default 5).
	FailureThreshold int
	// Cooldown is how long the breaker stays open before allowing a probe
	// (default 30s).
	Cooldown time.Duration
	// HalfOpenMaxProbes is the number of concurrent probes allowed in
	// half-open (default 1).
	HalfOpenMaxProbes int
	// OnStateChange, when set, is invoked after a state transition. It is called
	// WITHOUT the breaker lock held, so the callback may safely call back into
	// the breaker (State, Failures, ...) without deadlocking.
	OnStateChange func(endpointID string, from, to State)
}

// DefaultConfig returns the documented defaults.
func DefaultConfig() Config {
	return Config{FailureThreshold: 5, Cooldown: 30 * time.Second, HalfOpenMaxProbes: 1}
}

type endpointState struct {
	state          State
	failures       int
	openedAt       time.Time
	probesInFlight int
}

// Breaker manages independent breaker state per endpoint key.
type Breaker struct {
	cfg Config
	now func() time.Time

	mu     sync.Mutex
	states map[string]*endpointState
}

// New builds a Breaker with the given config.
func New(cfg Config) *Breaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = 30 * time.Second
	}
	if cfg.HalfOpenMaxProbes <= 0 {
		cfg.HalfOpenMaxProbes = 1
	}
	return &Breaker{
		cfg:    cfg,
		now:    time.Now,
		states: make(map[string]*endpointState),
	}
}

// WithClock overrides the time source (used in tests).
func (b *Breaker) WithClock(now func() time.Time) *Breaker {
	b.now = now
	return b
}

func (b *Breaker) get(id string) *endpointState {
	st, ok := b.states[id]
	if !ok {
		st = &endpointState{state: StateClosed}
		b.states[id] = st
	}
	return st
}

// Allow reports whether a request may proceed for the endpoint. In half-open it
// admits at most HalfOpenMaxProbes concurrent probes.
func (b *Breaker) Allow(id string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	st := b.get(id)
	switch st.state {
	case StateClosed:
		return true

	case StateOpen:
		if b.now().Sub(st.openedAt) >= b.cfg.Cooldown {
			// Cooldown elapsed: move to half-open and admit a probe.
			st.state = StateHalfOpen
			st.probesInFlight = 1
			return true
		}
		return false

	case StateHalfOpen:
		if st.probesInFlight < b.cfg.HalfOpenMaxProbes {
			st.probesInFlight++
			return true
		}
		return false
	}
	return false
}

// RecordSuccess records a successful call, closing the breaker.
func (b *Breaker) RecordSuccess(id string) {
	b.mu.Lock()
	st := b.get(id)
	from := st.state
	st.state = StateClosed
	st.failures = 0
	st.probesInFlight = 0
	b.mu.Unlock()

	// Notify outside the lock: the callback may read breaker state.
	if from != StateClosed {
		b.notify(id, from, StateClosed)
	}
}

// RecordFailure records a failed call and transitions the state.
func (b *Breaker) RecordFailure(id string) {
	b.mu.Lock()
	st := b.get(id)
	from := st.state
	switch st.state {
	case StateClosed:
		st.failures++
		if st.failures >= b.cfg.FailureThreshold {
			st.state = StateOpen
			st.openedAt = b.now()
		}
	case StateHalfOpen:
		// Probe failed: reopen with a fresh cooldown.
		st.state = StateOpen
		st.openedAt = b.now()
		st.probesInFlight = 0
		st.failures = b.cfg.FailureThreshold
	case StateOpen:
		// Already open; keep the window but refresh the timestamp is not
		// desired (would extend cooldown indefinitely). Leave as-is.
	}
	to := st.state
	b.mu.Unlock()

	if from != to {
		b.notify(id, from, to)
	}
}

// notify invokes the configured callback outside the breaker lock.
func (b *Breaker) notify(endpointID string, from, to State) {
	if b.cfg.OnStateChange == nil {
		return
	}
	func() {
		// A panicking callback must not take down the caller's request path.
		defer func() { _ = recover() }()
		b.cfg.OnStateChange(endpointID, from, to)
	}()
}

// State returns the current state name for the endpoint.
func (b *Breaker) State(id string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.get(id).state)
}

// Reset forcibly closes the breaker for an endpoint, clearing its failure
// window. Used by the admin circuit-breaker reset action.
func (b *Breaker) Reset(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	st := b.get(id)
	st.state = StateClosed
	st.failures = 0
	st.openedAt = time.Time{}
	st.probesInFlight = 0
}

// Failures returns the current consecutive failure count (for diagnostics).
func (b *Breaker) Failures(id string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.get(id).failures
}

// Config exposes the effective configuration.
func (b *Breaker) Config() Config { return b.cfg }
