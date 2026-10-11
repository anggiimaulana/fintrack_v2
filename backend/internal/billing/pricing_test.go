package billing

import "testing"

func baseRule() *PricingRule {
	return &PricingRule{
		BasePer1kIDR:     300,
		MarkupMultiplier: 1.25,
		MarkupPercent:    0,
		ModalityMultipliers: map[string]float64{
			"chat": 1.0, "vision": 2.5, "voice": 3.0, "embedding": 0.5, "reasoning": 2.0,
		},
	}
}

// TestPricing_ChatBaseline verifies the base chat cost.
func TestPricing_ChatBaseline(t *testing.T) {
	e := NewPricingEngine(baseRule())
	// 1000 tokens chat: 300 * 1 * 1.25 * 1.0 = 375
	if got := e.CostIDR(1000, ModalityChat); got != 375 {
		t.Fatalf("expected 375, got %d", got)
	}
}

// TestPricing_VisionMultiplier verifies modality scaling + floor rounding.
func TestPricing_VisionMultiplier(t *testing.T) {
	e := NewPricingEngine(baseRule())
	// 1000 tokens vision: 300 * 1.25 * 2.5 = 937.5 -> floor 937
	if got := e.CostIDR(1000, ModalityVision); got != 937 {
		t.Fatalf("expected 937, got %d", got)
	}
}

// TestPricing_EmbeddingCheap verifies embedding discount.
func TestPricing_EmbeddingCheap(t *testing.T) {
	e := NewPricingEngine(baseRule())
	// 1000 tokens embedding: 300 * 1.25 * 0.5 = 187.5 -> 187
	if got := e.CostIDR(1000, ModalityEmbedding); got != 187 {
		t.Fatalf("expected 187, got %d", got)
	}
}

// TestPricing_MarkupPercent verifies percentage markup.
func TestPricing_MarkupPercent(t *testing.T) {
	r := baseRule()
	r.MarkupPercent = 10
	e := NewPricingEngine(r)
	// 1000 chat: 300 * 1.25 * 1.10 = 412.5 -> 412
	if got := e.CostIDR(1000, ModalityChat); got != 412 {
		t.Fatalf("expected 412, got %d", got)
	}
}

// TestPricing_ZeroAndUnknown verifies edge cases.
func TestPricing_ZeroAndUnknown(t *testing.T) {
	e := NewPricingEngine(baseRule())
	if e.CostIDR(0, ModalityChat) != 0 {
		t.Fatal("zero tokens must cost 0")
	}
	if e.CostIDR(-5, ModalityChat) != 0 {
		t.Fatal("negative tokens must cost 0")
	}
	// Unknown modality falls back to 1.0 multiplier.
	if got := e.CostIDR(1000, Modality("unknown")); got != 375 {
		t.Fatalf("unknown modality should use 1.0 -> 375, got %d", got)
	}
}

// TestPricing_NilRule verifies a nil rule yields 0 (no panic).
func TestPricing_NilRule(t *testing.T) {
	e := NewPricingEngine(nil)
	if e.CostIDR(1000, ModalityChat) != 0 {
		t.Fatal("nil rule must cost 0")
	}
}
