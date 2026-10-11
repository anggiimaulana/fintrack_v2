package middleware

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// UsageProviders holds the data sources the usage resolvers need. Keeping them
// behind a narrow interface (rather than pulling repositories into the
// middleware) makes the resolvers unit-testable without a database.
type UsageProviders struct {
	// CountTransactions returns the user's live (non-soft-deleted) transaction
	// count — the usage for limit key `max_transactions`.
	CountTransactions func(ctx context.Context, userID uuid.UUID) (int64, error)
	// CountVisionScans returns vision ledger entries in the current period —
	// the usage for limit key `ocr_scans_per_cycle`.
	CountVisionScans func(ctx context.Context, userID uuid.UUID) (int64, error)
	// CurrentTokens returns the tokens consumed in the current period
	// (used + reserved) — the usage for limit key `ai_tokens_per_cycle`.
	CurrentTokens func(ctx context.Context, userID uuid.UUID) (int64, error)
	// Now supplies the current time (injectable for tests).
	Now func() time.Time
}

// UsageFor returns the UsageResolver for a limit key, or nil when the key has no
// resolver in this slice. A nil resolver means "no usage can be computed" — the
// middleware still enforces the limit, it just evaluates it as 0.
func (p *UsageProviders) UsageFor(limitKey string) UsageResolver {
	switch limitKey {
	case "max_transactions":
		return p.countTransactions
	case "ai_tokens_per_cycle":
		return p.currentTokens
	case "ocr_scans_per_cycle":
		return p.countVisionScans
	default:
		return nil
	}
}

func (p *UsageProviders) countTransactions(ctx context.Context, userID uuid.UUID, _ string) (int64, error) {
	if p.CountTransactions == nil {
		return 0, nil
	}
	return p.CountTransactions(ctx, userID)
}

func (p *UsageProviders) currentTokens(ctx context.Context, userID uuid.UUID, _ string) (int64, error) {
	if p.CurrentTokens == nil {
		return 0, nil
	}
	return p.CurrentTokens(ctx, userID)
}

func (p *UsageProviders) countVisionScans(ctx context.Context, userID uuid.UUID, _ string) (int64, error) {
	if p.CountVisionScans == nil {
		return 0, nil
	}
	return p.CountVisionScans(ctx, userID)
}

// TokenPeriodUsage reads token usage from a token-period repository.
//
// Contract (see the P07 entitlement spec §3.2): a user with no current period
// has consumed nothing, so usage is 0 and the request is permitted. Only a real
// query error is propagated — the middleware turns that into a 500 rather than
// silently allowing the request. Conflating "no data" with "error" would deny
// every user who has not yet made an AI call.
func TokenPeriodUsage(repo TokenPeriodReader) func(ctx context.Context, userID uuid.UUID) (int64, error) {
	return func(ctx context.Context, userID uuid.UUID) (int64, error) {
		period, err := repo.GetCurrentPeriod(ctx, userID, time.Now().UTC().Unix())
		if err != nil {
			return 0, err
		}
		if period == nil {
			return 0, nil // no period yet -> zero usage
		}
		return period.UsedTokens + period.ReservedTokens, nil
	}
}

// TokenPeriodReader is the read-only slice of billing.TokenRepository needed to
// evaluate token usage.
type TokenPeriodReader interface {
	GetCurrentPeriod(ctx context.Context, userID uuid.UUID, nowUnix int64) (*billing.TokenPeriod, error)
}
