package breaker

import (
	"sync"
	"testing"
	"time"
)

// fakeClock is a controllable time source for deterministic cooldown tests.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func newTestBreaker() (*Breaker, *fakeClock) {
	clk := &fakeClock{t: time.Unix(0, 0)}
	b := New(Config{FailureThreshold: 5, Cooldown: 30 * time.Second, HalfOpenMaxProbes: 1}).WithClock(clk.now)
	return b, clk
}

// TestBreaker_ClosedAllows verifies a fresh breaker allows requests.
func TestBreaker_ClosedAllows(t *testing.T) {
	b, _ := newTestBreaker()
	if !b.Allow("e1") {
		t.Fatal("closed breaker must allow")
	}
	if b.State("e1") != string(StateClosed) {
		t.Fatalf("expected closed, got %s", b.State("e1"))
	}
}

// TestBreaker_TripsOpenAfterThreshold verifies 5 consecutive failures open it.
func TestBreaker_TripsOpenAfterThreshold(t *testing.T) {
	b, _ := newTestBreaker()
	for i := range 4 {
		b.RecordFailure("e1")
		if b.State("e1") != string(StateClosed) {
			t.Fatalf("must stay closed after %d failures", i+1)
		}
	}
	b.RecordFailure("e1") // 5th
	if b.State("e1") != string(StateOpen) {
		t.Fatalf("expected open after 5 failures, got %s", b.State("e1"))
	}
	if b.Allow("e1") {
		t.Fatal("open breaker must reject before cooldown")
	}
}

// TestBreaker_SuccessResetsFailures verifies a success resets the counter.
func TestBreaker_SuccessResetsFailures(t *testing.T) {
	b, _ := newTestBreaker()
	b.RecordFailure("e1")
	b.RecordFailure("e1")
	b.RecordSuccess("e1")
	if b.Failures("e1") != 0 {
		t.Fatalf("expected failures reset to 0, got %d", b.Failures("e1"))
	}
	// Four more failures must NOT open (counter restarted).
	for range 4 {
		b.RecordFailure("e1")
	}
	if b.State("e1") != string(StateClosed) {
		t.Fatalf("expected still closed, got %s", b.State("e1"))
	}
}

// TestBreaker_HalfOpenThenClose verifies cooldown -> half-open -> success closes.
func TestBreaker_HalfOpenThenClose(t *testing.T) {
	b, clk := newTestBreaker()
	for range 5 {
		b.RecordFailure("e1")
	}
	if b.State("e1") != string(StateOpen) {
		t.Fatal("expected open")
	}
	if b.Allow("e1") {
		t.Fatal("must reject before cooldown")
	}

	clk.advance(30 * time.Second)
	if !b.Allow("e1") {
		t.Fatal("must allow probe after cooldown")
	}
	if b.State("e1") != string(StateHalfOpen) {
		t.Fatalf("expected half_open, got %s", b.State("e1"))
	}
	// Second concurrent probe rejected while one in flight.
	if b.Allow("e1") {
		t.Fatal("half-open must allow only 1 probe")
	}

	b.RecordSuccess("e1")
	if b.State("e1") != string(StateClosed) {
		t.Fatalf("expected closed after successful probe, got %s", b.State("e1"))
	}
}

// TestBreaker_HalfOpenProbeFailsReopens verifies a failed probe reopens.
func TestBreaker_HalfOpenProbeFailsReopens(t *testing.T) {
	b, clk := newTestBreaker()
	for range 5 {
		b.RecordFailure("e1")
	}
	clk.advance(30 * time.Second)
	if !b.Allow("e1") {
		t.Fatal("probe should be allowed")
	}
	b.RecordFailure("e1") // probe fails
	if b.State("e1") != string(StateOpen) {
		t.Fatalf("expected open after failed probe, got %s", b.State("e1"))
	}
	// Fresh cooldown required.
	if b.Allow("e1") {
		t.Fatal("must reject immediately after reopen")
	}
	clk.advance(30 * time.Second)
	if !b.Allow("e1") {
		t.Fatal("should allow again after new cooldown")
	}
}

// TestBreaker_IndependentEndpoints verifies state isolation per endpoint.
func TestBreaker_IndependentEndpoints(t *testing.T) {
	b, _ := newTestBreaker()
	for range 5 {
		b.RecordFailure("e1")
	}
	if b.State("e1") != string(StateOpen) {
		t.Fatal("e1 should be open")
	}
	if !b.Allow("e2") {
		t.Fatal("e2 must be unaffected")
	}
}

// TestBreaker_ConcurrentAllowIsRaceFree exercises concurrency (run with -race).
func TestBreaker_ConcurrentAllowIsRaceFree(t *testing.T) {
	b, _ := newTestBreaker()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "e" + string(rune('0'+n%3))
			if b.Allow(id) {
				if n%2 == 0 {
					b.RecordSuccess(id)
				} else {
					b.RecordFailure(id)
				}
			}
		}(i)
	}
	wg.Wait()
}
