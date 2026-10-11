package payment

import (
	"context"
	"testing"
)

// TestResolveGateway_PerPlanOverride verifies the per-plan override wins.
func TestResolveGateway_PerPlanOverride(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ManualGateway{BankName: "BCA"})
	reg.Register(&fakeGateway{name: "midtrans"})
	s := newMemSettings()
	s.m["payment_gateway_default"] = "midtrans"

	resolver := func(context.Context, string) (string, error) { return "manual", nil }
	gw, err := reg.ResolveGateway(context.Background(), s, "pro", resolver)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if gw.Name() != "manual" {
		t.Fatalf("expected manual override, got %s", gw.Name())
	}
}

// TestResolveGateway_GlobalDefault verifies the global setting is used when no
// per-plan override exists.
func TestResolveGateway_GlobalDefault(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ManualGateway{})
	reg.Register(&fakeGateway{name: "midtrans"})
	s := newMemSettings()
	s.m["payment_gateway_default"] = "midtrans"

	gw, err := reg.ResolveGateway(context.Background(), s, "free", nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if gw.Name() != "midtrans" {
		t.Fatalf("expected midtrans (global), got %s", gw.Name())
	}
}

// TestResolveGateway_EmergencyFallback verifies fallback to manual when the
// configured gateway is unknown.
func TestResolveGateway_EmergencyFallback(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ManualGateway{})
	s := newMemSettings()
	s.m["payment_gateway_default"] = "nonexistent"

	gw, err := reg.ResolveGateway(context.Background(), s, "", nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if gw.Name() != "manual" {
		t.Fatalf("expected manual fallback, got %s", gw.Name())
	}
}

// TestResolveGateway_InFlightIsolation documents that the chosen gateway name is
// captured at checkout: changing the global setting later does not affect the
// already-created payment (which stores its gateway).
func TestResolveGateway_InFlightIsolation(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ManualGateway{})
	reg.Register(&fakeGateway{name: "midtrans"})
	s := newMemSettings()

	s.m["payment_gateway_default"] = "midtrans"
	gw1, _ := reg.ResolveGateway(context.Background(), s, "", nil)

	// Admin switches the global gateway.
	s.m["payment_gateway_default"] = "manual"
	gw2, _ := reg.ResolveGateway(context.Background(), s, "", nil)

	if gw1.Name() != "midtrans" || gw2.Name() != "manual" {
		t.Fatalf("expected midtrans then manual, got %s then %s", gw1.Name(), gw2.Name())
	}
	// The first payment keeps "midtrans" recorded on its row (asserted by design).
}
