package billing

import (
	"testing"
)

// TestPricingEngine_SetRuleIsLive verifies a rule swap (as done by the admin
// pricing endpoint and the refresh loop) changes the next computed cost without
// rebuilding the engine — F2/T-PRC-05.
func TestPricingEngine_SetRuleIsLive(t *testing.T) {
	e := NewPricingEngine(baseRule())
	if got := e.CostIDR(1000, ModalityChat); got != 375 {
		t.Fatalf("baseline: expected 375, got %d", got)
	}

	// A Super Admin raises the base rate; the very next commit must use it.
	newRule := baseRule()
	newRule.BasePer1kIDR = 400
	e.SetRule(newRule)
	// 400 * 1.25 = 500
	if got := e.CostIDR(1000, ModalityChat); got != 500 {
		t.Fatalf("after SetRule: expected 500, got %d", got)
	}
}

// TestPricingEngine_NoRuleIsZero verifies a nil rule yields 0 (never a panic).
func TestPricingEngine_NoRuleIsZero(t *testing.T) {
	e := NewPricingEngine(nil)
	if got := e.CostIDR(1000, ModalityChat); got != 0 {
		t.Fatalf("expected 0 with no rule, got %d", got)
	}
	e.SetRule(baseRule())
	if got := e.CostIDR(1000, ModalityChat); got != 375 {
		t.Fatalf("expected 375 after SetRule, got %d", got)
	}
}

// TestPricingEngine_ZeroTokens verifies the zero/negative guard.
func TestPricingEngine_ZeroTokens(t *testing.T) {
	e := NewPricingEngine(baseRule())
	if got := e.CostIDR(0, ModalityChat); got != 0 {
		t.Fatalf("expected 0 for zero tokens, got %d", got)
	}
	if got := e.CostIDR(-5, ModalityChat); got != 0 {
		t.Fatalf("expected 0 for negative tokens, got %d", got)
	}
}

// TestPricingEngine_UnknownModalityFallsBack verifies an unlisted modality uses
// multiplier 1.0 (F2/T-PRC-07: a rule without markup must not error).
func TestPricingEngine_UnknownModalityFallsBack(t *testing.T) {
	r := baseRule()
	r.ModalityMultipliers = nil
	e := NewPricingEngine(r)
	// 300 * 1.25 * 1.0 = 375
	if got := e.CostIDR(1000, ModalityChat); got != 375 {
		t.Fatalf("expected 375 with no multipliers, got %d", got)
	}
}
