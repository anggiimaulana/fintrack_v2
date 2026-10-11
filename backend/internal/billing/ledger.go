package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// QuotaResolver returns the current quota cycle for a user: the period bounds
// and the quota granted for that cycle. Implemented by the subscription layer.
type QuotaResolver interface {
	ResolveQuota(ctx context.Context, userID uuid.UUID, at time.Time) (periodStart, periodEnd time.Time, quota int64, err error)
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
	tokens  TokenRepository
	tx      TxRunner
	quota   QuotaResolver
	pricing *PricingEngine
	now     func() time.Time
	// onCycleReset is called when a new token period is created for a user
	// (i.e. a new billing cycle started). Optional; used to emit quota_reset.
	onCycleReset func(ctx context.Context, userID uuid.UUID, start, end time.Time, quota int64)
}

// NewLedgerService builds the ledger service.
func NewLedgerService(tokens TokenRepository, tx TxRunner, quota QuotaResolver, pricing *PricingEngine) *LedgerService {
	return &LedgerService{tokens: tokens, tx: tx, quota: quota, pricing: pricing, now: func() time.Time { return time.Now().UTC() }}
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
	Amount    int64
	Modality  Modality
	CreatedAt time.Time
}

// Reserve locks `estTokens` from the user's current cycle quota.
//
// Idempotency: replaying with the same idempotencyKey returns the original
// reservation without mutating state.
//
// Errors: ErrInsufficientQuota when the remaining quota is too small.
func (s *LedgerService) Reserve(ctx context.Context, userID uuid.UUID, estTokens int64, modality Modality, idempotencyKey string) (*Reservation, error) {
	if estTokens < 0 {
		estTokens = 0
	}

	// Idempotent replay: return the existing reservation.
	if existing, err := s.tokens.GetLedgerByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return &Reservation{
			ID: existing.ID, PeriodID: existing.PeriodID, Amount: existing.Amount,
			Modality: Modality(existing.Modality), CreatedAt: existing.CreatedAt,
		}, nil
	}

	var res *Reservation
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		period, err := s.ensurePeriodLocked(txCtx, userID)
		if err != nil {
			return err
		}
		if period.Available() < estTokens {
			return ErrInsufficientQuota
		}
		if err := s.tokens.AdjustPeriod(txCtx, period.ID, 0, estTokens); err != nil {
			return err
		}
		entry := &TokenLedgerEntry{
			ID:             uuid.New(),
			PeriodID:       period.ID,
			UserID:         userID,
			EntryType:      EntryReserve,
			Amount:         estTokens,
			Modality:       string(modality),
			IdempotencyKey: idempotencyKey,
		}
		// The reserve entry references itself so commit/release can point back.
		entry.ReservationID = &entry.ID
		if err := s.tokens.InsertLedger(txCtx, entry); err != nil {
			return err
		}
		res = &Reservation{ID: entry.ID, PeriodID: period.ID, Amount: estTokens, Modality: modality, CreatedAt: entry.CreatedAt}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Commit finalizes a reservation with the actual token usage. The difference
// between reserved and actual is released back to available quota.
//
// Idempotency: pass a stable idempotencyKey (e.g. "commit:<reservationID>").
func (s *LedgerService) Commit(ctx context.Context, userID, reservationID uuid.UUID, actualTokens int64, idempotencyKey string) error {
	if actualTokens < 0 {
		actualTokens = 0
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
		period, err := s.tokens.GetPeriodForUpdate(txCtx, userID, s.now().Unix())
		if err != nil {
			return err
		}
		if period == nil {
			return ErrSubscriptionNotFound
		}
		// Release the whole reservation, then charge the actual amount.
		if err := s.tokens.AdjustPeriod(txCtx, period.ID, actualTokens, -reserve.Amount); err != nil {
			return err
		}
		entry := &TokenLedgerEntry{
			ID:             uuid.New(),
			PeriodID:       period.ID,
			UserID:         userID,
			ReservationID:  &reservationID,
			EntryType:      EntryCommit,
			Amount:         actualTokens,
			Modality:       reserve.Modality,
			IdempotencyKey: idempotencyKey,
		}
		if s.pricing != nil {
			entry.CostIDR = s.pricing.CostIDR(actualTokens, Modality(reserve.Modality))
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
		period, err := s.tokens.GetPeriodForUpdate(txCtx, userID, s.now().Unix())
		if err != nil {
			return err
		}
		if period == nil {
			return ErrSubscriptionNotFound
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

// ensurePeriodLocked resolves the current cycle and returns the locked period,
// creating it if necessary.
func (s *LedgerService) ensurePeriodLocked(ctx context.Context, userID uuid.UUID) (*TokenPeriod, error) {
	period, err := s.tokens.GetPeriodForUpdate(ctx, userID, s.now().Unix())
	if err != nil {
		return nil, err
	}
	if period != nil {
		return period, nil
	}
	// No period yet: resolve quota and create it, then lock again.
	start, end, quota, err := s.quota.ResolveQuota(ctx, userID, s.now())
	if err != nil {
		return nil, err
	}
	if _, err := s.tokens.GetOrCreatePeriod(ctx, userID, start.Unix(), end.Unix(), quota); err != nil {
		return nil, err
	}
	// A new period means a new cycle began for this user.
	if s.onCycleReset != nil {
		s.onCycleReset(ctx, userID, start, end, quota)
	}
	return s.tokens.GetPeriodForUpdate(ctx, userID, s.now().Unix())
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
