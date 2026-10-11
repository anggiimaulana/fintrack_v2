package billing

import "math"

// PricingEngine converts token usage to integer Rupiah using a pricing rule.
//
// Formula:
//
//	cost = base_per_1k × (tokens / 1000) × markup_multiplier × (1 + markup_percent/100) × modality_multiplier
//
// All arithmetic is integer-based on the final rounding step (round down) to
// avoid floating-point drift on money. Only the multipliers use float64, and the
// result is converted once at the end.
type PricingEngine struct {
	rule *PricingRule
}

// NewPricingEngine builds an engine from a rule.
func NewPricingEngine(rule *PricingRule) *PricingEngine {
	return &PricingEngine{rule: rule}
}

// Rule returns the underlying rule (may be nil).
func (e *PricingEngine) Rule() *PricingRule { return e.rule }

// CostIDR computes the integer Rupiah cost for `tokens` of the given modality.
// Returns 0 when the engine has no rule.
func (e *PricingEngine) CostIDR(tokens int64, modality Modality) int64 {
	if e.rule == nil || tokens <= 0 {
		return 0
	}

	mult := e.modalityMultiplier(modality)
	// base per 1000 tokens × (tokens/1000)
	base := float64(e.rule.BasePer1kIDR) * (float64(tokens) / 1000.0)
	withMarkup := base * e.rule.MarkupMultiplier
	if e.rule.MarkupPercent != 0 {
		withMarkup *= 1.0 + float64(e.rule.MarkupPercent)/100.0
	}
	total := withMarkup * mult

	// Round down to whole Rupiah (favours the user; deterministic).
	if total <= 0 {
		return 0
	}
	return int64(math.Floor(total))
}

func (e *PricingEngine) modalityMultiplier(modality Modality) float64 {
	if e.rule == nil || e.rule.ModalityMultipliers == nil {
		return 1.0
	}
	if v, ok := e.rule.ModalityMultipliers[string(modality)]; ok && v > 0 {
		return v
	}
	return 1.0
}
