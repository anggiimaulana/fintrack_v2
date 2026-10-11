package payment

import (
	"context"
	"math"
)

// TaxCalculator computes integer-Rupiah tax from a configurable rate.
//
//	tax = floor(subtotal × rate_percent / 100)
type TaxCalculator struct {
	settings SettingsRepository
	// fallbackRate is used when settings is unavailable (default 0 = PPN off).
	fallbackRate int64
}

// NewTaxCalculator builds the calculator.
func NewTaxCalculator(settings SettingsRepository) *TaxCalculator {
	return &TaxCalculator{settings: settings}
}

// RatePercent returns the configured tax rate (integer percent).
func (t *TaxCalculator) RatePercent(ctx context.Context) int64 {
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
