package billing

import (
	"math"
	"sync"
)

// PricingEngine converts token usage to integer Rupiah using a pricing rule.
//
// Formula:
//
//	cost = base_per_1k × (tokens / 1000) × markup_multiplier × (1 + markup_percent/100) × modality_multiplier
//
// All arithmetic is integer-based on the final rounding step (round down) to
// avoid floating-point drift on money. Only the multipliers use float64, and the
// result is converted once at the end.
//
// The rule is swappable at runtime (SetRule) so a Super Admin pricing change
// takes effect on the next commit without a restart. Reads and writes are
// guarded by an RWMutex because commits happen on concurrent request goroutines.
type PricingEngine struct {
	mu   sync.RWMutex
	rule *PricingRule
}

// NewPricingEngine builds an engine from a rule (which may be nil).
func NewPricingEngine(rule *PricingRule) *PricingEngine {
	return &PricingEngine{rule: rule}
}

// SetRule replaces the active rule atomically. Passing nil disables pricing
// (cost stays 0) — used when no rule is configured.
func (e *PricingEngine) SetRule(rule *PricingRule) {
	e.mu.Lock()
	e.rule = rule
	e.mu.Unlock()
}

// Rule returns the current rule (may be nil).
func (e *PricingEngine) Rule() *PricingRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rule
}

// CostIDR computes the integer Rupiah cost for `tokens` of the given modality.
// Returns 0 when the engine has no rule.
func (e *PricingEngine) CostIDR(tokens int64, modality Modality) int64 {
	e.mu.RLock()
	rule := e.rule
	e.mu.RUnlock()
	if rule == nil || tokens <= 0 {
		return 0
	}

	mult := modalityMultiplier(rule, modality)
	// Sell price normalised to per-1,000 tokens × (tokens/1000)
	base := float64(rule.SellPricePer1kIDR()) * (float64(tokens) / 1000.0)
	withMarkup := base * rule.MarkupMultiplier
	if rule.MarkupPercent != 0 {
		withMarkup *= 1.0 + float64(rule.MarkupPercent)/100.0
	}
	total := withMarkup * mult

	// Round down to whole Rupiah (favours the user; deterministic).
	if total <= 0 {
		return 0
	}
	return int64(math.Floor(total))
}

// modalityMultiplier returns the multiplier for a modality, defaulting to 1.0
// when the rule omits it (unknown modality, or no multipliers configured).
func modalityMultiplier(rule *PricingRule, modality Modality) float64 {
	if rule == nil || rule.ModalityMultipliers == nil {
		return 1.0
	}
	if v, ok := rule.ModalityMultipliers[string(modality)]; ok && v > 0 {
		return v
	}
	return 1.0
}

// ProviderCostIDR is what WE pay the upstream provider for `tokens` of the given
// modality, using the rule's cost basis. Returns 0 when no cost is configured.
func (e *PricingEngine) ProviderCostIDR(tokens int64, modality Modality) int64 {
	e.mu.RLock()
	rule := e.rule
	e.mu.RUnlock()
	if rule == nil || tokens <= 0 || rule.ProviderCostPer1kIDR <= 0 {
		return 0
	}
	mult := modalityMultiplier(rule, modality)
	total := float64(rule.ProviderCostPer1kIDR) * (float64(tokens) / 1000.0) * mult
	if total <= 0 {
		return 0
	}
	return int64(math.Floor(total))
}

// MarginIDR is the gross margin (sell − buy) in integer Rupiah for `tokens` of
// the given modality. Can be negative if the sell price is below cost.
func (e *PricingEngine) MarginIDR(tokens int64, modality Modality) int64 {
	return e.CostIDR(tokens, modality) - e.ProviderCostIDR(tokens, modality)
}
