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
	// CountAIRequests returns the number of AI calls made in the current period
	// — the usage for limit key `ai_requests_per_cycle`.
	CountAIRequests func(ctx context.Context, userID uuid.UUID) (int64, error)
	// CountVoiceToday returns the number of voice transcriptions made since the
	// start of the current day (in the configured reset timezone) — the usage
	// for limit key `voice_requests_per_day`.
	CountVoiceToday func(ctx context.Context, userID uuid.UUID) (int64, error)
	// Now supplies the current time (injectable for tests).
	Now func() time.Time
	// Catalog resolves a metric key to its configured resolver. When nil, the
	// built-in defaults are used so the middleware still works in tests.
	Catalog ResolverLookup
}

// ResolverLookup resolves a metric's configured resolver key from the catalog.
// Declared as an interface so this package does not import the catalog directly
// (keeping the middleware dependency-light and unit-testable).
type ResolverLookup interface {
	// ResolverKeyFor returns the resolver key for a metric, or "" when the
	// metric is unknown/inactive/not enforced.
	ResolverKeyFor(ctx context.Context, metricKey string) string
}

// resolverRegistry maps a resolver_key (stored in metric_definitions) to the
// function that computes usage. The CODE provides these capabilities; the
// DATABASE chooses which one each metric uses. Adding a new capability requires
// a deploy; pointing an existing metric at one does not.
func (p *UsageProviders) resolverRegistry() map[string]UsageResolver {
	return map[string]UsageResolver{
		"token_period_usage":     p.currentTokens,
		"ai_requests_in_period":  p.countAIRequests,
		"active_transactions":    p.countTransactions,
		"vision_scans_in_period": p.countVisionScans,
		"voice_requests_per_day": p.countVoiceToday,
	}
}

// defaultResolverKeys is the built-in mapping used when no catalog is wired
// (tests, or a deployment that predates the catalog). It mirrors the seed rows
// in migration 0029, so behaviour is identical either way.
var defaultResolverKeys = map[string]string{
	"ai_tokens_per_cycle":    "token_period_usage",
	"ai_requests_per_cycle":  "ai_requests_in_period",
	"max_transactions":       "active_transactions",
	"ocr_scans_per_cycle":    "vision_scans_in_period",
	"voice_requests_per_day": "voice_requests_per_day",
}

// UsageFor returns the UsageResolver for a limit key, or nil when the key has no
// resolver. A nil resolver means "no usage can be computed" — the middleware
// still enforces the limit, it just evaluates it as 0.
//
// The mapping is DATA (metric_definitions.resolver_key) when a catalog is
// configured, falling back to the built-in defaults otherwise.
func (p *UsageProviders) UsageFor(ctx context.Context, limitKey string) UsageResolver {
	key := ""
	if p.Catalog != nil {
		key = p.Catalog.ResolverKeyFor(ctx, limitKey)
	} else {
		key = defaultResolverKeys[limitKey]
	}
	if key == "" {
		return nil
	}
	return p.resolverRegistry()[key]
}

func (p *UsageProviders) countAIRequests(ctx context.Context, userID uuid.UUID, _ string) (int64, error) {
	if p.CountAIRequests == nil {
		return 0, nil
	}
	return p.CountAIRequests(ctx, userID)
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

func (p *UsageProviders) countVoiceToday(ctx context.Context, userID uuid.UUID, _ string) (int64, error) {
	if p.CountVoiceToday == nil {
		return 0, nil
	}
	return p.CountVoiceToday(ctx, userID)
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
