package middleware

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestUsageFor_AllKeysHaveResolvers verifies every limit key the plans use has a
// resolver, so `WithinLimit` never silently evaluates a limit as 0 (which would
// let the limit be bypassed). Regression guard for GAP-03/GAP-05.
func TestUsageFor_AllKeysHaveResolvers(t *testing.T) {
	noop := func(context.Context, uuid.UUID) (int64, error) { return 0, nil }
	p := &UsageProviders{
		CountTransactions: noop,
		CurrentTokens:     noop,
		CountVisionScans:  noop,
		CountAIRequests:   noop,
	}

	// These are the numeric limit keys seeded for the Free/Pro/Business plans.
	for _, key := range []string{
		"max_transactions",
		"ai_tokens_per_cycle",
		"ocr_scans_per_cycle",
		"ai_requests_per_cycle",
	} {
		if p.UsageFor(context.Background(), key) == nil {
			t.Errorf("UsageFor(%q) = nil; limit would be bypassed", key)
		}
	}
}

func TestUsageFor_UnknownKeyReturnsNil(t *testing.T) {
	p := &UsageProviders{}
	if p.UsageFor(context.Background(), "max_wallets") != nil {
		t.Error("max_wallets has no resolver yet; expected nil (documented gap)")
	}
}

func TestUsageFor_AIRequestsResolves(t *testing.T) {
	calls := 0
	p := &UsageProviders{
		CountAIRequests: func(context.Context, uuid.UUID) (int64, error) {
			calls++
			return 7, nil
		},
	}
	resolver := p.UsageFor(context.Background(), "ai_requests_per_cycle")
	if resolver == nil {
		t.Fatal("ai_requests_per_cycle resolver is nil")
	}
	got, err := resolver(context.Background(), uuid.New(), "ai_requests_per_cycle")
	if err != nil {
		t.Fatalf("resolver error: %v", err)
	}
	if got != 7 {
		t.Fatalf("usage = %d, want 7", got)
	}
	if calls != 1 {
		t.Fatalf("resolver called %d times, want 1", calls)
	}
}

// TestUsageFor_NilProviderIsSafe verifies a missing source degrades to 0 rather
// than panicking (the middleware still enforces the limit).
func TestUsageFor_NilProviderIsSafe(t *testing.T) {
	p := &UsageProviders{}
	resolver := p.UsageFor(context.Background(), "ai_requests_per_cycle")
	if resolver == nil {
		t.Fatal("resolver should be non-nil even when the provider is nil")
	}
	got, err := resolver(context.Background(), uuid.New(), "ai_requests_per_cycle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Fatalf("usage = %d, want 0", got)
	}
}
