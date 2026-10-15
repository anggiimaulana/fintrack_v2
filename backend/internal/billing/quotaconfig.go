package billing

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// SettingsQuotaConfig supplies the operator-configurable quota policy: the
// rounding mode, the minimum charge, and the unlimited sentinel. Every value is
// read from `settings` (DB) with a short cache, so a Super Admin change applies
// without a redeploy.
//
// A nil reader keeps the safe defaults (ceil, no minimum, 1e9 sentinel), which
// is also the fallback when a setting is missing — the system must never break
// because configuration is absent.
type SettingsQuotaConfig struct {
	settings SettingsReader
	ttl      time.Duration

	mu      sync.RWMutex
	mode    RoundingMode
	minChar int64
	sentl   int64
	expires time.Time
}

// NewSettingsQuotaConfig builds the reader. settings may be nil (defaults only).
func NewSettingsQuotaConfig(settings SettingsReader) *SettingsQuotaConfig {
	return &SettingsQuotaConfig{settings: settings, ttl: 30 * time.Second}
}

// WithTTL overrides the cache TTL (tests).
func (c *SettingsQuotaConfig) WithTTL(d time.Duration) *SettingsQuotaConfig {
	c.ttl = d
	return c
}

func (c *SettingsQuotaConfig) load(ctx context.Context) {
	c.mu.RLock()
	if time.Now().Before(c.expires) {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	mode := RoundCeil
	minCharge := int64(1)
	sentinel := int64(1_000_000_000)
	if c.settings != nil {
		if v, err := c.settings.GetString(ctx, "quota_rounding_mode"); err == nil {
			mode = ParseRoundingMode(trimQuotes(v))
		}
		if v, err := c.settings.GetString(ctx, "quota_min_charge_units"); err == nil {
			if n, perr := strconv.ParseInt(trimQuotes(v), 10, 64); perr == nil && n >= 0 {
				minCharge = n
			}
		}
		if v, err := c.settings.GetString(ctx, "quota_unlimited_sentinel"); err == nil {
			if n, perr := strconv.ParseInt(trimQuotes(v), 10, 64); perr == nil && n > 0 {
				sentinel = n
			}
		}
	}

	c.mu.Lock()
	c.mode, c.minChar, c.sentl = mode, minCharge, sentinel
	c.expires = time.Now().Add(c.ttl)
	c.mu.Unlock()
}

// Mode implements RoundingReader.
func (c *SettingsQuotaConfig) Mode(ctx context.Context) RoundingMode {
	c.load(ctx)
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mode
}

// MinCharge implements RoundingReader.
func (c *SettingsQuotaConfig) MinCharge(ctx context.Context) int64 {
	c.load(ctx)
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.minChar
}

// UnlimitedSentinel returns the value substituted for a plan limit of -1.
func (c *SettingsQuotaConfig) UnlimitedSentinel(ctx context.Context) int64 {
	c.load(ctx)
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sentl
}

// trimQuotes strips a JSON string's surrounding double quotes. Settings are
// stored as JSONB, so a string value arrives as `"kuota"` rather than `kuota`.
func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
