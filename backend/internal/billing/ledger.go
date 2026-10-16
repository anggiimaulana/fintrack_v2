package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// QuotaResolver returns the current quota cycle for a user: the period bounds,
// the quota granted for that cycle, and the token→unit ratio to snapshot.
// Implemented by the subscription layer.
type QuotaResolver interface {
	ResolveQuota(ctx context.Context, userID uuid.UUID, at time.Time) (periodStart, periodEnd time.Time, quota int64, ratio int64, err error)
}

// QuotaResolverFunc adapts a function to QuotaResolver.
type QuotaResolverFunc func(ctx context.Context, userID uuid.UUID, at time.Time) (periodStart, periodEnd time.Time, quota int64, ratio int64, err error)

func (f QuotaResolverFunc) ResolveQuota(ctx context.Context, userID uuid.UUID, at time.Time) (time.Time, time.Time, int64, int64, error) {
	return f(ctx, userID, at)
}

// RoundingReader supplies the operator-configured rounding mode and minimum
// charge. Implemented by a settings-backed reader; nil falls back to ceil/1.
type RoundingReader interface {
	// Mode returns the active rounding mode (default ceil when unset).
	Mode(ctx context.Context) RoundingMode
	// MinCharge returns the minimum units charged per request (0 = none).
	MinCharge(ctx context.Context) int64
}

// TxRunner runs fn inside a single DB transaction (implemented by
// postgres.TxManager). The ledger service uses it to keep row locks short.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// LedgerService implements the 2-phase token ledger: reserve -> commit/release.
// All mutations are append-only and idempotent. Row locks are held only for the
// brief DB transaction, never across the AI network call.
type LedgerService struct {
	tokens   TokenRepository
	tx       TxRunner
	quota    QuotaResolver
	pricing  *PricingEngine
	rounding RoundingReader
	now      func() time.Time
	// onCycleReset is called when a new token period is created for a user
	// (i.e. a new billing cycle started). Optional; used to emit quota_reset.
	onCycleReset func(ctx context.Context, userID uuid.UUID, start, end time.Time, quota int64)
}

// NewLedgerService builds the ledger service.
func NewLedgerService(tokens TokenRepository, tx TxRunner, quota QuotaResolver, pricing *PricingEngine) *LedgerService {
	return &LedgerService{tokens: tokens, tx: tx, quota: quota, pricing: pricing, now: func() time.Time { return time.Now().UTC() }}
}

// WithRounding installs the rounding-policy reader (P-13/P-15). Nil keeps the
// safe defaults (ceil, no minimum).
func (s *LedgerService) WithRounding(r RoundingReader) *LedgerService {
	s.rounding = r
	return s
}

// roundPolicy returns the active rounding mode and minimum charge, falling back
// to ceil / no-minimum when no reader is configured.
func (s *LedgerService) roundPolicy(ctx context.Context) (RoundingMode, int64) {
	if s.rounding == nil {
		return RoundCeil, 0
	}
	return s.rounding.Mode(ctx), s.rounding.MinCharge(ctx)
}

// toUnits converts provider tokens to plan-quota units using the period's
// snapshot ratio and the operator-configured rounding policy.
func (s *LedgerService) toUnits(ctx context.Context, rawTokens, ratio int64) int64 {
	mode, minCharge := s.roundPolicy(ctx)
	units := RoundUnits(rawTokens, ratio, mode)
	return ApplyMinCharge(units, minCharge, rawTokens)
}

// WithClock overrides the time source (tests).
func (s *LedgerService) WithClock(now func() time.Time) *LedgerService { s.now = now; return s }

// OnCycleReset registers a callback invoked when a new token period is created.
// It fires only on creation — not on every period lookup — so consumers see one
// reset per cycle rather than one per AI call.
func (s *LedgerService) OnCycleReset(fn func(ctx context.Context, userID uuid.UUID, start, end time.Time, quota int64)) *LedgerService {
	s.onCycleReset = fn
	return s
}

// Reservation is the handle returned by Reserve.
type Reservation struct {
	ID        uuid.UUID
	PeriodID  uuid.UUID
	Amount    int64 // plan-quota units
	RawTokens int64 // estimated raw provider tokens
	Modality  Modality
	CreatedAt time.Time
}

// Reserve locks quota for an AI call.
//
// `estRawTokens` is the ESTIMATED PROVIDER TOKEN count (raw). The ledger
// converts it to plan-quota units using the period's snapshot ratio and the
// configured rounding policy, and returns a Reservation whose Amount is in
// UNITS. Callers must therefore pass RAW tokens, never pre-converted units —
// converting twice would drain quota far too slowly.
//
// Idempotency: replaying with the same idempotencyKey returns the original
// reservation without mutating state.
//
// Errors: ErrInsufficientQuota when the remaining quota is too small.
func (s *LedgerService) Reserve(ctx context.Context, userID uuid.UUID, estRawTokens int64, modality Modality, idempotencyKey string) (*Reservation, error) {
	if estRawTokens < 0 {
		estRawTokens = 0
	}

	// Idempotent replay: return the existing reservation.
	if existing, err := s.tokens.GetLedgerByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return &Reservation{
			ID: existing.ID, PeriodID: existing.PeriodID, Amount: existing.Amount,
			RawTokens: existing.RawTokensOrAmount(),
			Modality: Modality(existing.Modality), CreatedAt: existing.CreatedAt,
		}, nil
	}

	var res *Reservation
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		period, err := s.ensurePeriodLocked(txCtx, userID)
		if err != nil {
			return err
		}
		// Convert the RAW estimate to units using THIS cycle's snapshot ratio.
		estUnits := s.toUnits(txCtx, estRawTokens, period.QuotaRatio)
		if period.Available() < estUnits {
			return ErrInsufficientQuota
		}
		if err := s.tokens.AdjustPeriod(txCtx, period.ID, 0, estUnits); err != nil {
			return err
		}
		entry := &TokenLedgerEntry{
			ID:             uuid.New(),
			PeriodID:       period.ID,
			UserID:         userID,
			EntryType:      EntryReserve,
			Amount:         estUnits, // units
			RawTokens:      estRawTokens,
			Modality:       string(modality),
			IdempotencyKey: idempotencyKey,
		}
		// The reserve entry references itself so commit/release can point back.
		entry.ReservationID = &entry.ID
		if err := s.tokens.InsertLedger(txCtx, entry); err != nil {
			return err
		}
		res = &Reservation{ID: entry.ID, PeriodID: period.ID, Amount: estUnits, RawTokens: estRawTokens, Modality: modality, CreatedAt: entry.CreatedAt}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Commit finalizes a reservation with the ACTUAL PROVIDER TOKEN usage. The
// tokens are converted to plan-quota units using the period's snapshot ratio,
// and the difference between reserved units and actual units is released back
// to available quota.
//
// `rawTokens` is the provider-reported token count — the basis for both the
// quota charge (via the ratio) and the Rupiah cost (always from raw).
//
// Idempotency: pass a stable idempotencyKey (e.g. "commit:<reservationID>").
func (s *LedgerService) Commit(ctx context.Context, userID, reservationID uuid.UUID, rawTokens int64, idempotencyKey string) error {
	if rawTokens < 0 {
		rawTokens = 0
	}

	if existing, err := s.tokens.GetLedgerByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return err
	} else if existing != nil {
		return nil // already committed
	}

	reserve, err := s.findReservation(ctx, reservationID)
	if err != nil {
		return err
	}
	if reserve == nil {
		return ErrReservationNotFound
	}
	if s.resolved(ctx, reservationID) {
		return ErrReservationAlreadyResolved
	}

	return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Use the period that was actually locked by the reservation, so a cycle
		// turnover between reserve and commit does not decrement reserved_tokens on
		// the new period or charge used_tokens to the wrong bucket.
		period, err := s.tokens.GetPeriodByIDForUpdate(txCtx, reserve.PeriodID)
		if err != nil {
			return err
		}
		if period == nil {
			// Fallback: look up by current time if period row was not found by id.
			period, err = s.tokens.GetPeriodForUpdate(txCtx, userID, s.now().Unix())
			if err != nil {
				return err
			}
			if period == nil {
				return ErrSubscriptionNotFound
			}
		}
		// Re-check INSIDE the transaction, now that the period row is locked.
		// The pre-transaction check above is only a fast path: two concurrent
		// commits (with different idempotency keys) both pass it, then both
		// charge. Locking the period serialises them, so the loser sees the
		// winner's commit here and aborts instead of double-charging.
		if resolved, rerr := s.tokens.IsReservationResolved(txCtx, reservationID); rerr != nil {
			return rerr
		} else if resolved {
			return ErrReservationAlreadyResolved
		}
		// Convert the RAW actual to units for THIS cycle (snapshot ratio).
		units := s.toUnits(txCtx, rawTokens, period.QuotaRatio)
		// The reservation is an ESTIMATE; the provider may report more tokens
		// than were held. Charge at most what the period can still absorb so
		// `used_tokens` never exceeds `quota_tokens` (the balance must never go
		// negative). Overruns are logged, not silently lost, so the reserve
		// estimate can be tuned.
		if maxCharge := period.QuotaTokens - period.UsedTokens; units > maxCharge {
			logger.Warn("token commit exceeds remaining quota; clamping",
				zap.String("user_id", userID.String()),
				zap.Int64("raw_tokens", rawTokens),
				zap.Int64("requested_units", units),
				zap.Int64("charged_units", maxCharge),
				zap.Int64("reserved_units", reserve.Amount),
				zap.Int64("ratio", period.QuotaRatio),
				zap.String("modality", reserve.Modality))
			if maxCharge < 0 {
				maxCharge = 0
			}
			units = maxCharge
		}
		// Release the whole reservation, then charge the (clamped) actual units.
		if err := s.tokens.AdjustPeriod(txCtx, period.ID, units, -reserve.Amount); err != nil {
			return err
		}
		entry := &TokenLedgerEntry{
			ID:             uuid.New(),
			PeriodID:       period.ID,
			UserID:         userID,
			ReservationID:  &reservationID,
			EntryType:      EntryCommit,
			Amount:         units,     // units
			RawTokens:      rawTokens, // provider tokens (cost basis)
			Modality:       reserve.Modality,
			IdempotencyKey: idempotencyKey,
		}
		// Cost is ALWAYS derived from raw provider tokens, never from units —
		// the ratio must not change what the work actually costs.
		if s.pricing != nil {
			entry.CostIDR = s.pricing.CostIDR(rawTokens, Modality(reserve.Modality))
		}
		return s.tokens.InsertLedger(txCtx, entry)
	})
}

// Release returns a reservation's locked quota without charging it (AI failed).
func (s *LedgerService) Release(ctx context.Context, userID, reservationID uuid.UUID, idempotencyKey string) error {
	if existing, err := s.tokens.GetLedgerByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return err
	} else if existing != nil {
		return nil
	}

	reserve, err := s.findReservation(ctx, reservationID)
	if err != nil {
		return err
	}
	if reserve == nil {
		return ErrReservationNotFound
	}
	if s.resolved(ctx, reservationID) {
		return ErrReservationAlreadyResolved
	}

	return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		period, err := s.tokens.GetPeriodByIDForUpdate(txCtx, reserve.PeriodID)
		if err != nil {
			return err
		}
		if period == nil {
			period, err = s.tokens.GetPeriodForUpdate(txCtx, userID, s.now().Unix())
			if err != nil {
				return err
			}
			if period == nil {
				return ErrSubscriptionNotFound
			}
		}
		// Re-check inside the transaction (see Commit): the unlocked pre-check
		// lets two concurrent resolutions of the same reservation both proceed.
		if resolved, rerr := s.tokens.IsReservationResolved(txCtx, reservationID); rerr != nil {
			return rerr
		} else if resolved {
			return ErrReservationAlreadyResolved
		}
		if err := s.tokens.AdjustPeriod(txCtx, period.ID, 0, -reserve.Amount); err != nil {
			return err
		}
		entry := &TokenLedgerEntry{
			ID:             uuid.New(),
			PeriodID:       period.ID,
			UserID:         userID,
			ReservationID:  &reservationID,
			EntryType:      EntryRelease,
			Amount:         -reserve.Amount,
			Modality:       reserve.Modality,
			IdempotencyKey: idempotencyKey,
		}
		return s.tokens.InsertLedger(txCtx, entry)
	})
}

// Refund returns quota that was already charged (used_tokens) back to the
// user's available balance. Use this when a feature fails AFTER a commit — e.g.
// an OCR scan whose provider call returned no usable result. Without it the user
// is billed for work that never completed (quota consumed, feature did nothing).
//
// `rawTokens` is the provider token count to refund; it is converted to units
// using the period's snapshot ratio (symmetric with Commit).
//
// Idempotent on idempotencyKey. The refunded units are clamped to what is
// actually charged (used_tokens) so a refund can never make the balance
// negative.
func (s *LedgerService) Refund(ctx context.Context, userID uuid.UUID, rawTokens int64, modality Modality, idempotencyKey string) error {
	if rawTokens <= 0 {
		return nil
	}
	if existing, err := s.tokens.GetLedgerByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return err
	} else if existing != nil {
		return nil // already refunded
	}

	return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		period, err := s.tokens.GetPeriodForUpdate(txCtx, userID, s.now().Unix())
		if err != nil {
			return err
		}
		if period == nil {
			return ErrSubscriptionNotFound
		}
		units := s.toUnits(txCtx, rawTokens, period.QuotaRatio)
		// Never refund more than was actually charged.
		if units > period.UsedTokens {
			units = period.UsedTokens
		}
		if units <= 0 {
			return nil
		}
		if err := s.tokens.AdjustPeriod(txCtx, period.ID, -units, 0); err != nil {
			return err
		}
		entry := &TokenLedgerEntry{
			ID:             uuid.New(),
			PeriodID:       period.ID,
			UserID:         userID,
			EntryType:      EntryRefund,
			Amount:         -units,    // negative: reduces used_tokens
			RawTokens:      rawTokens, // provider tokens being returned
			Modality:       string(modality),
			IdempotencyKey: idempotencyKey,
		}
		return s.tokens.InsertLedger(txCtx, entry)
	})
}

// defaultCycleLength is the fallback cycle length when a resolved window is
// empty or unusable (a healthy plan always supplies billing_period_days).
const defaultCycleLength = 30 * 24 * time.Hour

// ensurePeriodLocked resolves the current cycle and returns the locked period,
// creating it when a new cycle has begun. It never returns (nil, nil): callers
// dereference the period, so a missing cycle is an error, not a nil.
//
// A period that already covers `now` IS the current cycle and is reused as-is:
// its quota and ratio are a SNAPSHOT for the cycle's life. A plan change is
// pushed into the running period by whoever applies it (SubscriptionActivator on
// payment, the lifecycle worker on expiry — see QuotaSyncer), never inferred
// here. This keeps quota enforcement a single, deliberate act and preserves the
// snapshot invariant the domain documents.
func (s *LedgerService) ensurePeriodLocked(ctx context.Context, userID uuid.UUID) (*TokenPeriod, error) {
	now := s.now()
	period, err := s.tokens.GetPeriodForUpdate(ctx, userID, now.Unix())
	if err != nil {
		return nil, err
	}
	if period != nil {
		return period, nil
	}

	// No period covers `now`: a new cycle must begin. Resolve its window + quota.
	start, end, quota, ratio, err := s.quota.ResolveQuota(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	if ratio < 1 {
		ratio = 1
	}
	// The resolved window must COVER now. It may not: an `active` subscription
	// whose current_period_end has lapsed keeps granting access (the entitlement
	// gate keys on status, not period end) and the cycle only advances when a
	// payment lands. Creating a stale window verbatim inserts a row that never
	// matches the current-time lookup, so the final lookup returns nil and the
	// caller panics dereferencing it. Roll the window forward so it always
	// covers now, preserving the cycle length.
	if start.After(now) || !end.After(now) {
		length := end.Sub(start)
		if length <= 0 {
			length = defaultCycleLength
		}
		logger.Warn("token period window lapsed; rolling forward",
			zap.String("user_id", userID.String()),
			zap.Time("resolved_start", start),
			zap.Time("resolved_end", end))
		start = now
		end = now.Add(length)
	}
	if _, err := s.tokens.GetOrCreatePeriod(ctx, userID, start.Unix(), end.Unix(), quota, ratio); err != nil {
		return nil, err
	}
	// A new period means a new cycle began for this user.
	if s.onCycleReset != nil {
		s.onCycleReset(ctx, userID, start, end, quota)
	}
	locked, err := s.tokens.GetPeriodForUpdate(ctx, userID, now.Unix())
	if err != nil {
		return nil, err
	}
	if locked == nil {
		// Defensive: unreachable after the roll-forward, but never hand a nil
		// period to a caller that will dereference it.
		return nil, ErrSubscriptionExpired
	}
	return locked, nil
}

// findReservation loads the reserve ledger entry for a reservation id.
func (s *LedgerService) findReservation(ctx context.Context, reservationID uuid.UUID) (*TokenLedgerEntry, error) {
	return s.tokens.GetLedgerByID(ctx, reservationID)
}

// resolved reports whether a reservation already has a commit/release.
func (s *LedgerService) resolved(ctx context.Context, reservationID uuid.UUID) bool {
	ok, err := s.tokens.IsReservationResolved(ctx, reservationID)
	return err == nil && ok
}
