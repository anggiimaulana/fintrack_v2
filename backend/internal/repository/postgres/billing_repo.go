package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/billing"
)

// BillingRepo implements the billing repositories over PostgreSQL. Write
// methods honour the tx-aware querier so they participate in a transaction when
// one is present in the context (see TxManager.RunInTx).
type BillingRepo struct {
	pool *pgxpool.Pool
}

// NewBillingRepo builds the billing repository.
func NewBillingRepo(pool *pgxpool.Pool) *BillingRepo {
	return &BillingRepo{pool: pool}
}

// ─── PlanRepository ───────────────────────────────────────────

func (r *BillingRepo) GetPlanByCode(ctx context.Context, code string) (*billing.Plan, error) {
	return r.scanPlan(q(ctx, r.pool).QueryRow(ctx, planSelect+` WHERE code = $1`, code))
}

func (r *BillingRepo) GetPlanByID(ctx context.Context, id uuid.UUID) (*billing.Plan, error) {
	return r.scanPlan(q(ctx, r.pool).QueryRow(ctx, planSelect+` WHERE id = $1`, id))
}

func (r *BillingRepo) ListActivePlans(ctx context.Context) ([]*billing.Plan, error) {
	rows, err := q(ctx, r.pool).Query(ctx, planSelect+` WHERE is_active = true ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*billing.Plan
	for rows.Next() {
		p, err := r.scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

const planSelect = `SELECT id, code, name, COALESCE(description,''), price_monthly_idr, price_yearly_idr,
    billing_period_days, is_active, is_public, fallback_gateway, sort_order, created_at, updated_at
    FROM plans`

type rowScanner interface{ Scan(dest ...any) error }

func (r *BillingRepo) scanPlan(row rowScanner) (*billing.Plan, error) {
	p := &billing.Plan{}
	err := row.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.PriceMonthlyIDR, &p.PriceYearlyIDR,
		&p.BillingPeriodDays, &p.IsActive, &p.IsPublic, &p.FallbackGateway, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, billing.ErrPlanNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *BillingRepo) GetEntitlements(ctx context.Context, planID uuid.UUID) (map[string]bool, error) {
	rows, err := q(ctx, r.pool).Query(ctx,
		`SELECT feature_key, is_enabled FROM plan_entitlements WHERE plan_id = $1`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		var v bool
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (r *BillingRepo) GetLimits(ctx context.Context, planID uuid.UUID) (map[string]int64, error) {
	rows, err := q(ctx, r.pool).Query(ctx,
		`SELECT limit_key, limit_value FROM plan_limits WHERE plan_id = $1`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var k string
		var v int64
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// ─── SubscriptionRepository ───────────────────────────────────

const subSelect = `SELECT id, user_id, plan_id, status, current_period_start, current_period_end,
    cancel_at_period_end, canceled_at, gateway, gateway_subscription_id, metadata, created_at, updated_at
    FROM subscriptions`

func (r *BillingRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*billing.Subscription, error) {
	s := &billing.Subscription{}
	err := q(ctx, r.pool).QueryRow(ctx, subSelect+` WHERE user_id = $1`, userID).
		Scan(&s.ID, &s.UserID, &s.PlanID, &s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
			&s.CancelAtPeriodEnd, &s.CanceledAt, &s.Gateway, &s.GatewaySubscriptionID, &s.Metadata, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, billing.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *BillingRepo) Create(ctx context.Context, s *billing.Subscription) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Metadata == nil {
		s.Metadata = map[string]any{}
	}
	err := q(ctx, r.pool).QueryRow(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id, status, current_period_start, current_period_end,
            cancel_at_period_end, gateway, gateway_subscription_id, metadata)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING created_at, updated_at`,
		s.ID, s.UserID, s.PlanID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.CancelAtPeriodEnd, s.Gateway, s.GatewaySubscriptionID, s.Metadata).
		Scan(&s.CreatedAt, &s.UpdatedAt)
	return err
}

func (r *BillingRepo) Update(ctx context.Context, s *billing.Subscription) error {
	if s.Metadata == nil {
		s.Metadata = map[string]any{}
	}
	tag, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE subscriptions SET plan_id=$2, status=$3, current_period_start=$4, current_period_end=$5,
            cancel_at_period_end=$6, canceled_at=$7, gateway=$8, metadata=$9, updated_at=now()
         WHERE id=$1`,
		s.ID, s.PlanID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.CancelAtPeriodEnd, s.CanceledAt, s.Gateway, s.Metadata)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return billing.ErrSubscriptionNotFound
	}
	return nil
}

func (r *BillingRepo) RecordEvent(ctx context.Context, ev *billing.SubscriptionEvent) error {
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	if ev.PreviousState == nil {
		ev.PreviousState = map[string]any{}
	}
	if ev.NewState == nil {
		ev.NewState = map[string]any{}
	}
	_, err := q(ctx, r.pool).Exec(ctx,
		`INSERT INTO subscription_events (id, subscription_id, user_id, event_type, previous_state, new_state, reason, actor_id)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		ev.ID, ev.SubscriptionID, ev.UserID, ev.EventType, ev.PreviousState, ev.NewState, ev.Reason, ev.ActorID)
	return err
}

func (r *BillingRepo) ListEvents(ctx context.Context, userID uuid.UUID, limit int) ([]*billing.SubscriptionEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := q(ctx, r.pool).Query(ctx,
		`SELECT id, subscription_id, user_id, event_type, previous_state, new_state, reason, actor_id, created_at
         FROM subscription_events WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*billing.SubscriptionEvent
	for rows.Next() {
		e := &billing.SubscriptionEvent{}
		var created time.Time
		if err := rows.Scan(&e.ID, &e.SubscriptionID, &e.UserID, &e.EventType, &e.PreviousState, &e.NewState, &e.Reason, &e.ActorID, &created); err != nil {
			return nil, err
		}
		e.CreatedAt = created.Format(time.RFC3339)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ─── TokenRepository ──────────────────────────────────────────

func (r *BillingRepo) GetOrCreatePeriod(ctx context.Context, userID uuid.UUID, periodStart, periodEnd int64, quota int64) (*billing.TokenPeriod, error) {
	ps := time.Unix(periodStart, 0).UTC()
	pe := time.Unix(periodEnd, 0).UTC()
	_, err := q(ctx, r.pool).Exec(ctx,
		`INSERT INTO token_periods (user_id, period_start, period_end, quota_tokens)
         VALUES ($1,$2,$3,$4) ON CONFLICT (user_id, period_start) DO NOTHING`,
		userID, ps, pe, quota)
	if err != nil {
		return nil, err
	}
	return r.getPeriodByStart(ctx, userID, ps)
}

func (r *BillingRepo) getPeriodByStart(ctx context.Context, userID uuid.UUID, ps time.Time) (*billing.TokenPeriod, error) {
	p := &billing.TokenPeriod{}
	err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT id, user_id, period_start, period_end, quota_tokens, used_tokens, reserved_tokens, created_at, updated_at
         FROM token_periods WHERE user_id=$1 AND period_start=$2`, userID, ps).
		Scan(&p.ID, &p.UserID, &p.PeriodStart, &p.PeriodEnd, &p.QuotaTokens, &p.UsedTokens, &p.ReservedTokens, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *BillingRepo) GetPeriodForUpdate(ctx context.Context, userID uuid.UUID, nowUnix int64) (*billing.TokenPeriod, error) {
	return r.currentPeriod(ctx, userID, nowUnix, true)
}

// GetCurrentPeriod reads the current period without locking. Used for
// pre-flight checks (entitlement usage) where a row lock is inappropriate.
func (r *BillingRepo) GetCurrentPeriod(ctx context.Context, userID uuid.UUID, nowUnix int64) (*billing.TokenPeriod, error) {
	return r.currentPeriod(ctx, userID, nowUnix, false)
}

// currentPeriod selects the row covering `nowUnix`, optionally locking it.
// Returns (nil, nil) when no period covers the instant.
func (r *BillingRepo) currentPeriod(ctx context.Context, userID uuid.UUID, nowUnix int64, forUpdate bool) (*billing.TokenPeriod, error) {
	now := time.Unix(nowUnix, 0).UTC()
	query := `SELECT id, user_id, period_start, period_end, quota_tokens, used_tokens, reserved_tokens, created_at, updated_at
         FROM token_periods
         WHERE user_id=$1 AND period_start <= $2 AND period_end >= $2
         ORDER BY period_start DESC LIMIT 1`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	p := &billing.TokenPeriod{}
	err := q(ctx, r.pool).QueryRow(ctx, query, userID, now).
		Scan(&p.ID, &p.UserID, &p.PeriodStart, &p.PeriodEnd, &p.QuotaTokens, &p.UsedTokens, &p.ReservedTokens, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // caller creates the period / treats as zero usage
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *BillingRepo) AdjustPeriod(ctx context.Context, periodID uuid.UUID, usedDelta, reservedDelta int64) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE token_periods SET used_tokens = used_tokens + $2, reserved_tokens = reserved_tokens + $3, updated_at = now()
         WHERE id = $1`, periodID, usedDelta, reservedDelta)
	return err
}

// CountAIRequestsInPeriod counts AI reserve entries (one per AI call: chat,
// vision, or voice) for the user's current period. Used as the usage value for
// the `ai_requests_per_cycle` limit. A user with no current period has made no
// requests, so it returns 0.
//
// Reserve entries are counted (not commits) because a reserve marks the start of
// a request; counting commits would miss in-flight or failed calls.
func (r *BillingRepo) CountAIRequestsInPeriod(ctx context.Context, userID uuid.UUID, nowUnix int64) (int64, error) {
	now := time.Unix(nowUnix, 0).UTC()
	var count int64
	err := q(ctx, r.pool).QueryRow(ctx, `
		SELECT COUNT(*) FROM token_ledger l
		JOIN token_periods p ON p.id = l.period_id
		WHERE l.user_id = $1 AND l.entry_type = $2
		  AND p.period_start <= $3 AND p.period_end >= $3`,
		userID, string(billing.EntryReserve), now).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountVisionScansInPeriod counts vision-modality ledger entries for the user's
// current period. Used as the usage value for the `ocr_scans_per_cycle` limit.
// A user with no current period has made no scans, so it returns 0.
func (r *BillingRepo) CountVisionScansInPeriod(ctx context.Context, userID uuid.UUID, nowUnix int64) (int64, error) {
	now := time.Unix(nowUnix, 0).UTC()
	var count int64
	err := q(ctx, r.pool).QueryRow(ctx, `
		SELECT COUNT(*) FROM token_ledger l
		JOIN token_periods p ON p.id = l.period_id
		WHERE l.user_id = $1 AND l.modality = $2
		  AND p.period_start <= $3 AND p.period_end >= $3`,
		userID, string(billing.ModalityVision), now).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *BillingRepo) InsertLedger(ctx context.Context, e *billing.TokenLedgerEntry) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	err := q(ctx, r.pool).QueryRow(ctx,
		`INSERT INTO token_ledger (id, period_id, user_id, reservation_id, entry_type, amount, modality, cost_idr, idempotency_key, reference_id, notes)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING created_at`,
		e.ID, e.PeriodID, e.UserID, e.ReservationID, e.EntryType, e.Amount, e.Modality, e.CostIDR, e.IdempotencyKey, e.ReferenceID, e.Notes).
		Scan(&e.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return billing.ErrDuplicateIdempotencyKey
		}
		return err
	}
	return nil
}

func (r *BillingRepo) GetLedgerByIdempotencyKey(ctx context.Context, key string) (*billing.TokenLedgerEntry, error) {
	e := &billing.TokenLedgerEntry{}
	err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT id, period_id, user_id, reservation_id, entry_type, amount, modality, cost_idr, idempotency_key, reference_id, notes, created_at
         FROM token_ledger WHERE idempotency_key = $1`, key).
		Scan(&e.ID, &e.PeriodID, &e.UserID, &e.ReservationID, &e.EntryType, &e.Amount, &e.Modality, &e.CostIDR, &e.IdempotencyKey, &e.ReferenceID, &e.Notes, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

// GetLedgerByID returns a ledger entry by primary key.
func (r *BillingRepo) GetLedgerByID(ctx context.Context, id uuid.UUID) (*billing.TokenLedgerEntry, error) {
	e := &billing.TokenLedgerEntry{}
	err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT id, period_id, user_id, reservation_id, entry_type, amount, modality, cost_idr, idempotency_key, reference_id, notes, created_at
         FROM token_ledger WHERE id = $1`, id).
		Scan(&e.ID, &e.PeriodID, &e.UserID, &e.ReservationID, &e.EntryType, &e.Amount, &e.Modality, &e.CostIDR, &e.IdempotencyKey, &e.ReferenceID, &e.Notes, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

// IsReservationResolved reports whether a commit/release exists for the reservation.
func (r *BillingRepo) IsReservationResolved(ctx context.Context, reservationID uuid.UUID) (bool, error) {
	var exists bool
	err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM token_ledger WHERE reservation_id = $1 AND entry_type IN ('commit','release'))`,
		reservationID).Scan(&exists)
	return exists, err
}

// ListExpiredReservations finds reserve entries older than cutoff that have no
// matching commit/release referencing them.
func (r *BillingRepo) ListExpiredReservations(ctx context.Context, cutoff time.Time) ([]*billing.TokenLedgerEntry, error) {
	rows, err := q(ctx, r.pool).Query(ctx,
		`SELECT r.id, r.period_id, r.user_id, r.entry_type, r.amount, r.modality, r.cost_idr, r.idempotency_key, r.created_at
         FROM token_ledger r
         WHERE r.entry_type = 'reserve' AND r.created_at < $1
           AND NOT EXISTS (
               SELECT 1 FROM token_ledger x
               WHERE x.reservation_id = r.id AND x.entry_type IN ('commit','release')
           )
         ORDER BY r.created_at ASC LIMIT 500`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*billing.TokenLedgerEntry
	for rows.Next() {
		e := &billing.TokenLedgerEntry{}
		if err := rows.Scan(&e.ID, &e.PeriodID, &e.UserID, &e.EntryType, &e.Amount, &e.Modality, &e.CostIDR, &e.IdempotencyKey, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ─── PricingRepository ────────────────────────────────────────

func (r *BillingRepo) GetActiveRule(ctx context.Context) (*billing.PricingRule, error) {
	pr := &billing.PricingRule{}
	err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT id, rule_name, base_per_1k_idr, markup_multiplier::float8, markup_percent, modality_multipliers, is_active, created_at, updated_at
         FROM pricing_rules WHERE is_active = true ORDER BY created_at ASC LIMIT 1`).
		Scan(&pr.ID, &pr.RuleName, &pr.BasePer1kIDR, &pr.MarkupMultiplier, &pr.MarkupPercent, &pr.ModalityMultipliers, &pr.IsActive, &pr.CreatedAt, &pr.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, billing.ErrNoPricingRule
	}
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// compile-time assertions.
var (
	_ billing.PlanRepository         = (*BillingRepo)(nil)
	_ billing.SubscriptionRepository = (*BillingRepo)(nil)
	_ billing.TokenRepository        = (*BillingRepo)(nil)
	_ billing.PricingRepository      = (*BillingRepo)(nil)
)

var _ = fmt.Sprintf

// ListDueScheduledDowngrades returns the user ids whose scheduled end-of-cycle
// downgrade is now due: a `metadata.scheduled_plan_code` is set and the current
// period has ended. The partial index on cancel_at_period_end keeps this cheap.
func (r *BillingRepo) ListDueScheduledDowngrades(ctx context.Context, now time.Time) ([]uuid.UUID, error) {
	rows, err := q(ctx, r.pool).Query(ctx,
		`SELECT user_id FROM subscriptions
         WHERE cancel_at_period_end = true
           AND metadata ? 'scheduled_plan_code'
           AND current_period_end IS NOT NULL
           AND current_period_end <= $1`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// SetSubscriptionStatus transitions a user's subscription to a new status. Used
// by the grace/dunning worker to enforce the lifecycle (F3). A missing
// subscription is a no-op, not an error: users without a row fall back to the
// default plan anyway.
func (r *BillingRepo) SetSubscriptionStatus(ctx context.Context, userID uuid.UUID, status billing.SubscriptionStatus) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE subscriptions SET status = $2 WHERE user_id = $1`, userID, status)
	return err
}
