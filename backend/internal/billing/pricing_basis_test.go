package billing

import "testing"

// TestPricing_Per10kUnit verifies that a "per_10k" rule is normalised to a
// per-1,000-token price (divide by 10, integer floor) before costing.
func TestPricing_Per10kUnit(t *testing.T) {
	r := &PricingRule{
		BasePer1kIDR:        3000, // Rp3.000 per 10.000 token
		PriceUnit:           PriceUnitPer10k,
		MarkupMultiplier:    1.0,
		ModalityMultipliers: map[string]float64{"chat": 1.0},
	}
	if got := r.SellPricePer1kIDR(); got != 300 {
		t.Fatalf("expected normalised 300/1k, got %d", got)
	}
	e := NewPricingEngine(r)
	// 1000 chat tokens at 300/1k, markup 1.0 -> 300
	if got := e.CostIDR(1000, ModalityChat); got != 300 {
		t.Fatalf("expected 300, got %d", got)
	}
}

// TestPricing_Per1kUnitUnchanged guards the default unit path.
func TestPricing_Per1kUnitUnchanged(t *testing.T) {
	r := &PricingRule{BasePer1kIDR: 300, PriceUnit: PriceUnitPer1k, MarkupMultiplier: 1.0}
	if got := r.SellPricePer1kIDR(); got != 300 {
		t.Fatalf("expected 300/1k, got %d", got)
	}
}

// TestPricing_ProviderCostAndMargin verifies the cost basis and margin maths.
func TestPricing_ProviderCostAndMargin(t *testing.T) {
	r := &PricingRule{
		BasePer1kIDR:         300,
		PriceUnit:            PriceUnitPer1k,
		ProviderCostPer1kIDR: 200,
		MarkupMultiplier:     1.25,
		ModalityMultipliers:  map[string]float64{"chat": 1.0},
	}
	e := NewPricingEngine(r)
	// sell: 300 * 1.25 = 375 ; buy: 200 ; margin 175
	if got := e.CostIDR(1000, ModalityChat); got != 375 {
		t.Fatalf("expected sell 375, got %d", got)
	}
	if got := e.ProviderCostIDR(1000, ModalityChat); got != 200 {
		t.Fatalf("expected provider cost 200, got %d", got)
	}
	if got := e.MarginIDR(1000, ModalityChat); got != 175 {
		t.Fatalf("expected margin 175, got %d", got)
	}
}

// TestPricing_ProviderCostUnsetIsZero ensures an unset cost basis yields 0 and a
// margin equal to the sell price (never negative from a missing basis).
func TestPricing_ProviderCostUnsetIsZero(t *testing.T) {
	e := NewPricingEngine(baseRule())
	if got := e.ProviderCostIDR(1000, ModalityChat); got != 0 {
		t.Fatalf("expected 0 provider cost, got %d", got)
	}
	if got := e.MarginIDR(1000, ModalityChat); got != 375 {
		t.Fatalf("expected margin == sell (375), got %d", got)
	}
}

// TestPricing_NegativeMargin verifies a sell price below cost reports negative.
func TestPricing_NegativeMargin(t *testing.T) {
	r := &PricingRule{
		BasePer1kIDR:         100,
		PriceUnit:            PriceUnitPer1k,
		ProviderCostPer1kIDR: 200,
		MarkupMultiplier:     1.0,
	}
	e := NewPricingEngine(r)
	if got := e.MarginIDR(1000, ModalityChat); got != -100 {
		t.Fatalf("expected margin -100, got %d", got)
	}
}

// TestSellPricePer1k_NilRuleIsZero guards the nil receiver.
func TestSellPricePer1k_NilRuleIsZero(t *testing.T) {
	var r *PricingRule
	if got := r.SellPricePer1kIDR(); got != 0 {
		t.Fatalf("expected 0 for nil rule, got %d", got)
	}
}
