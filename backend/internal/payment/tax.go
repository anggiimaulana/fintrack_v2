package payment

import (
	"context"
	"math"
	"strings"
)

// TaxCalculator computes integer-Rupiah tax from configurable settings.
//
//	tax = floor(subtotal × rate_percent / 100)
//
// Tax is off by default: `tax_enabled` must be explicitly true before any tax
// is added, so a deployment that has not opted in never charges PPN by accident.
// Both the enabled flag and the rate are read at invoice time, so an operator
// change applies to the next invoice without a redeploy.
type TaxCalculator struct {
	settings SettingsRepository
	// fallbackRate is used when settings is unavailable (default 0 = PPN off).
	fallbackRate int64
}

// NewTaxCalculator builds the calculator.
func NewTaxCalculator(settings SettingsRepository) *TaxCalculator {
	return &TaxCalculator{settings: settings}
}

// Enabled reports whether tax should be applied. Default false: a missing
// setting means "no tax", so PPN is opt-in rather than opt-out.
func (t *TaxCalculator) Enabled(ctx context.Context) bool {
	if t.settings == nil {
		return false
	}
	v, err := t.settings.GetString(ctx, "tax_enabled")
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "on", "1", "yes":
		return true
	default:
		return false
	}
}

// Label returns the display label for the tax (e.g. "PPN"), defaulting to "PPN".
func (t *TaxCalculator) Label(ctx context.Context) string {
	if t.settings == nil {
		return "PPN"
	}
	v, err := t.settings.GetString(ctx, "tax_label")
	if err != nil || strings.TrimSpace(v) == "" {
		return "PPN"
	}
	return strings.TrimSpace(v)
}

// RatePercent returns the configured tax rate (integer percent). It is 0 when
// tax is disabled, so callers that only read the rate cannot charge by mistake.
func (t *TaxCalculator) RatePercent(ctx context.Context) int64 {
	if !t.Enabled(ctx) {
		return 0
	}
	if t.settings != nil {
		if v, err := t.settings.GetInt(ctx, "tax_rate_percent"); err == nil && v >= 0 {
			return v
		}
	}
	return t.fallbackRate
}

// TaxIDR returns the tax amount for a subtotal, floored to whole Rupiah.
func (t *TaxCalculator) TaxIDR(ctx context.Context, subtotal int64) int64 {
	if subtotal <= 0 {
		return 0
	}
	rate := t.RatePercent(ctx)
	if rate <= 0 {
		return 0
	}
	return int64(math.Floor(float64(subtotal) * float64(rate) / 100.0))
}

// Total returns subtotal + tax.
func (t *TaxCalculator) Total(ctx context.Context, subtotal int64) (tax, total int64) {
	tax = t.TaxIDR(ctx, subtotal)
	return tax, subtotal + tax
}
