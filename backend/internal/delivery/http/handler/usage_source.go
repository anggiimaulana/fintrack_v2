package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/entitlement"
)

// UsageProvidersFn adapts a set of per-key counters into an entitlement.UsageSource.
//
// It is the single place that knows how each limit key's consumption is
// measured; the entitlement handler stays free of repository dependencies. Keys
// without a counter are omitted from the payload (the UI shows "—" rather than a
// misleading 0), and a failing counter is skipped rather than failing the whole
// request.
type UsageProvidersFn struct {
	// Counters maps a limit key to its usage function. A nil function (or an
	// unknown key) is treated as "not measurable".
	Counters map[string]func(ctx context.Context, userID uuid.UUID) (int64, error)
	// Period returns the active quota cycle, or nil when none is tracked.
	Period func(ctx context.Context, userID uuid.UUID) (*entitlement.Period, error)
}

// Usage implements entitlement.UsageSource.
func (u *UsageProvidersFn) Usage(ctx context.Context, userID uuid.UUID, limits map[string]int64) (map[string]entitlement.UsageEntry, *entitlement.Period, error) {
	out := map[string]entitlement.UsageEntry{}
	for key, limit := range limits {
		counter := u.Counters[key]
		if counter == nil {
			continue
		}
		used, err := counter(ctx, userID)
		if err != nil {
			continue
		}
		entry := entitlement.UsageEntry{Limit: limit}
		if billing.IsUnlimited(limit) {
			entry.Unlimited = true
			entry.Used = used
			entry.Remaining = -1
		} else {
			entry.Used = used
			entry.Remaining = limit - used
			if entry.Remaining < 0 {
				entry.Remaining = 0
			}
		}
		out[key] = entry
	}

	var period *entitlement.Period
	if u.Period != nil {
		if p, err := u.Period(ctx, userID); err == nil {
			period = p
		}
	}
	return out, period, nil
}

// compile-time assertion.
var _ entitlement.UsageSource = (*UsageProvidersFn)(nil)
