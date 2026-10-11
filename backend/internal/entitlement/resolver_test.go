package entitlement

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
)

// fakeSource is an in-memory entitlement source.
type fakeSource struct {
	plan     *billing.Plan
	status   billing.SubscriptionStatus
	features map[string]bool
	limits   map[string]int64
	calls    int
}

func (f *fakeSource) GetActivePlanForUser(context.Context, uuid.UUID) (*billing.Plan, billing.SubscriptionStatus, error) {
	f.calls++
	return f.plan, f.status, nil
}
func (f *fakeSource) GetEntitlements(context.Context, uuid.UUID) (map[string]bool, error) {
	return f.features, nil
}
func (f *fakeSource) GetLimits(context.Context, uuid.UUID) (map[string]int64, error) {
	return f.limits, nil
}

func newSource() *fakeSource {
	return &fakeSource{
		plan:     &billing.Plan{ID: uuid.New(), Code: "pro", Name: "Pro"},
		status:   billing.StatusActive,
		features: map[string]bool{"ai_chat": true, "ocr_receipt": true},
		limits:   map[string]int64{"ai_tokens_per_cycle": 150000, "max_transactions": -1},
	}
}

// TestResolver_ResolvesFeaturesAndLimits verifies basic resolution.
func TestResolver_ResolvesFeaturesAndLimits(t *testing.T) {
	src := newSource()
	r := NewResolver(src, nil)
	res, err := r.Resolve(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !res.Feature("ai_chat") || res.Feature("voice_input") {
		t.Fatalf("unexpected features: %+v", res.Features)
	}
	if res.Limit("ai_tokens_per_cycle") != 150000 {
		t.Fatalf("unexpected limit: %d", res.Limit("ai_tokens_per_cycle"))
	}
	if res.PlanCode != "pro" {
		t.Fatalf("expected pro, got %s", res.PlanCode)
	}
}

// TestResolver_WithinLimit verifies unlimited + bounded semantics.
func TestResolver_WithinLimit(t *testing.T) {
	r := NewResolver(newSource(), nil)
	res, _ := r.Resolve(context.Background(), uuid.New())
	if !res.WithinLimit("max_transactions", 999999) {
		t.Fatal("unlimited must always pass")
	}
	if !res.WithinLimit("ai_tokens_per_cycle", 100) {
		t.Fatal("100 < 150000 must pass")
	}
	if res.WithinLimit("ai_tokens_per_cycle", 150000) {
		t.Fatal("current == limit must fail")
	}
}

// TestResolver_NoPlanDeniesAll verifies fail-closed when there is no plan.
func TestResolver_NoPlanDeniesAll(t *testing.T) {
	src := &fakeSource{}
	r := NewResolver(src, nil)
	res, err := r.Resolve(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res.Feature("ai_chat") {
		t.Fatal("no plan must deny all features")
	}
}

// TestResolver_L1Cache verifies the second resolve is served from L1 (no extra
// source call).
func TestResolver_L1Cache(t *testing.T) {
	src := newSource()
	r := NewResolver(src, nil)
	u := uuid.New()
	_, _ = r.Resolve(context.Background(), u)
	_, _ = r.Resolve(context.Background(), u)
	if src.calls != 1 {
		t.Fatalf("expected 1 source call (L1 cache), got %d", src.calls)
	}
}

// TestResolver_InvalidateBumpsVersion verifies version bump invalidates cache.
func TestResolver_InvalidateBumpsVersion(t *testing.T) {
	src := newSource()
	cache := infraRedis.NewCacheEngine(nil) // in-memory backend
	r := NewResolver(src, cache)
	u := uuid.New()

	first, _ := r.Resolve(context.Background(), u)
	if first.Version != 0 {
		t.Fatalf("expected version 0, got %d", first.Version)
	}

	r.Invalidate(context.Background(), u)

	second, _ := r.Resolve(context.Background(), u)
	if second.Version != 1 {
		t.Fatalf("expected version 1 after invalidate, got %d", second.Version)
	}
	if src.calls < 2 {
		t.Fatalf("invalidate must force a re-resolve (calls=%d)", src.calls)
	}
}

// TestResolver_L1TTLExpiry verifies L1 entries expire.
func TestResolver_L1TTLExpiry(t *testing.T) {
	src := newSource()
	r := NewResolver(src, nil)
	r.SetL1TTL(time.Nanosecond)
	u := uuid.New()
	_, _ = r.Resolve(context.Background(), u)
	time.Sleep(2 * time.Millisecond)
	_, _ = r.Resolve(context.Background(), u)
	if src.calls < 2 {
		t.Fatalf("L1 entry should have expired (calls=%d)", src.calls)
	}
}
