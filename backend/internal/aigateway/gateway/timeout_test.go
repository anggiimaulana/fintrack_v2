package gateway

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// newRoute builds a minimal chat route with a given per-route timeout.
func newRoute(timeoutMS int) *domain.AIRoute {
	return &domain.AIRoute{
		ID:        uuid.New(),
		UseCase:   domain.UseCaseChat,
		Model:     "m",
		IsActive:  true,
		TimeoutMS: timeoutMS,
		Endpoint:  &domain.AIEndpoint{ID: uuid.New(), Name: "ep"},
	}
}

// TestTimeoutFor_Precedence locks the resolution order: an explicit Super Admin
// per-use-case override beats the route's timeout_ms, which beats the default.
func TestTimeoutFor_Precedence(t *testing.T) {
	def := 90 * time.Second
	svc := New(&fakeRepo{}, Config{DefaultTimeout: def})

	// 1. No override: the route's timeout_ms wins.
	rt := newRoute(5000)
	if got := svc.timeoutFor(rt); got != 5*time.Second {
		t.Fatalf("expected route timeout 5s, got %v", got)
	}

	// 2. No override and no route timeout: the default applies.
	rtNoTimeout := newRoute(0)
	if got := svc.timeoutFor(rtNoTimeout); got != def {
		t.Fatalf("expected default %v, got %v", def, got)
	}

	// 3. Override present: it beats the route's timeout_ms.
	svc.SetUseCaseTimeouts(map[domain.UseCase]time.Duration{
		domain.UseCaseChat: 120 * time.Second,
	})
	if got := svc.timeoutFor(rt); got != 120*time.Second {
		t.Fatalf("expected override 120s, got %v", got)
	}
	if got := svc.timeoutFor(rtNoTimeout); got != 120*time.Second {
		t.Fatalf("expected override 120s for route without timeout, got %v", got)
	}

	// 4. A different use case is unaffected by the chat override.
	visionRoute := newRoute(7000)
	visionRoute.UseCase = domain.UseCaseVision
	if got := svc.timeoutFor(visionRoute); got != 7*time.Second {
		t.Fatalf("expected vision route timeout 7s, got %v", got)
	}
}

// TestSetUseCaseTimeouts_ClearAndZeroIgnored verifies nil/empty clears overrides
// and non-positive durations are dropped (they mean "no override").
func TestSetUseCaseTimeouts_ClearAndZeroIgnored(t *testing.T) {
	svc := New(&fakeRepo{}, Config{DefaultTimeout: 30 * time.Second})
	rt := newRoute(5000)

	svc.SetUseCaseTimeouts(map[domain.UseCase]time.Duration{
		domain.UseCaseChat:   0, // ignored
		domain.UseCaseVision: 10 * time.Second,
	})
	if got := svc.timeoutFor(rt); got != 5*time.Second {
		t.Fatalf("zero override must be ignored, got %v", got)
	}

	// Clearing restores route/default resolution.
	svc.SetUseCaseTimeouts(nil)
	if got := svc.timeoutFor(rt); got != 5*time.Second {
		t.Fatalf("expected route timeout after clear, got %v", got)
	}
	if d := svc.UseCaseTimeout(domain.UseCaseVision); d != 0 {
		t.Fatalf("expected no vision override after clear, got %v", d)
	}
}
