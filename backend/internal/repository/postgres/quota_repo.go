package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/quota"
)

// QuotaRepo manages persistence for dual-cycle quotas, usage counters, and threshold configurations.
type QuotaRepo struct {
	pool *pgxpool.Pool
}

// NewQuotaRepo creates a new QuotaRepo.
func NewQuotaRepo(pool *pgxpool.Pool) *QuotaRepo {
	return &QuotaRepo{pool: pool}
}

var _ quota.CycleStore = (*QuotaRepo)(nil)
var _ quota.UsageCounterStore = (*QuotaRepo)(nil)

// GetPlanCycle retrieves cycle configuration for a plan and limit key.
func (r *QuotaRepo) GetPlanCycle(ctx context.Context, planID uuid.UUID, limitKey string) (*quota.PlanLimitCycle, error) {
	var row quota.PlanLimitCycle
	err := q(ctx, r.pool).QueryRow(ctx, `
		SELECT id, plan_id, limit_key, cycle_type, weekly_limit, degraded_mode,
		       throttle_rate_amount, throttle_rate_window_secs, fallback_model,
		       max_degraded_usage, created_at, updated_at
		FROM plan_limit_cycles
		WHERE plan_id = $1 AND limit_key = $2
	`, planID, limitKey).Scan(
		&row.ID, &row.PlanID, &row.LimitKey, &row.CycleType, &row.WeeklyLimit,
		&row.DegradedMode, &row.ThrottleRateAmount, &row.ThrottleRateWindowSecs,
		&row.FallbackModel, &row.MaxDegradedUsage, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// GetPlanCycles retrieves all cycle configurations for a plan.
func (r *QuotaRepo) GetPlanCycles(ctx context.Context, planID uuid.UUID) ([]quota.PlanLimitCycle, error) {
	rows, err := q(ctx, r.pool).Query(ctx, `
		SELECT id, plan_id, limit_key, cycle_type, weekly_limit, degraded_mode,
		       throttle_rate_amount, throttle_rate_window_secs, fallback_model,
		       max_degraded_usage, created_at, updated_at
		FROM plan_limit_cycles
		WHERE plan_id = $1
		ORDER BY limit_key ASC
	`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []quota.PlanLimitCycle
	for rows.Next() {
		var row quota.PlanLimitCycle
		if err := rows.Scan(
			&row.ID, &row.PlanID, &row.LimitKey, &row.CycleType, &row.WeeklyLimit,
			&row.DegradedMode, &row.ThrottleRateAmount, &row.ThrottleRateWindowSecs,
			&row.FallbackModel, &row.MaxDegradedUsage, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// UpsertPlanCycles replaces/updates plan limit cycles batch.
func (r *QuotaRepo) UpsertPlanCycles(ctx context.Context, planID uuid.UUID, cycles []quota.PlanLimitCycle) error {
	for _, c := range cycles {
		_, err := q(ctx, r.pool).Exec(ctx, `
			INSERT INTO plan_limit_cycles (
				plan_id, limit_key, cycle_type, weekly_limit, degraded_mode,
				throttle_rate_amount, throttle_rate_window_secs, fallback_model, max_degraded_usage
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (plan_id, limit_key) DO UPDATE SET
				cycle_type = EXCLUDED.cycle_type,
				weekly_limit = EXCLUDED.weekly_limit,
				degraded_mode = EXCLUDED.degraded_mode,
				throttle_rate_amount = EXCLUDED.throttle_rate_amount,
				throttle_rate_window_secs = EXCLUDED.throttle_rate_window_secs,
				fallback_model = EXCLUDED.fallback_model,
				max_degraded_usage = EXCLUDED.max_degraded_usage,
				updated_at = now()
		`, planID, c.LimitKey, c.CycleType, c.WeeklyLimit, c.DegradedMode,
			c.ThrottleRateAmount, c.ThrottleRateWindowSecs, c.FallbackModel, c.MaxDegradedUsage)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetWarningThreshold retrieves the warning threshold for a plan (or global fallback).
func (r *QuotaRepo) GetWarningThreshold(ctx context.Context, planID uuid.UUID) (int, error) {
	var threshold int
	err := q(ctx, r.pool).QueryRow(ctx, `
		SELECT warning_threshold
		FROM quota_notification_config
		WHERE plan_id = $1
	`, planID).Scan(&threshold)
	if err == nil && threshold > 0 {
		return threshold, nil
	}

	// Fallback to global config (plan_id IS NULL)
	err = q(ctx, r.pool).QueryRow(ctx, `
		SELECT warning_threshold
		FROM quota_notification_config
		WHERE plan_id IS NULL
	`).Scan(&threshold)
	if err == nil && threshold > 0 {
		return threshold, nil
	}
	return 80, nil
}

// GetOrCreateCounter ensures an active usage_counter row exists for user, key, cycle type, and window.
func (r *QuotaRepo) GetOrCreateCounter(
	ctx context.Context,
	userID uuid.UUID,
	limitKey, cycleType string,
	start, end time.Time,
) (*quota.UsageCounter, error) {
	// 1. Insert on conflict do nothing
	_, err := q(ctx, r.pool).Exec(ctx, `
		INSERT INTO usage_counters (user_id, limit_key, cycle_type, cycle_start, cycle_end, used, degraded_used)
		VALUES ($1, $2, $3, $4, $5, 0, 0)
		ON CONFLICT (user_id, limit_key, cycle_type, cycle_start) DO NOTHING
	`, userID, limitKey, cycleType, start, end)
	if err != nil {
		return nil, err
	}

	// 2. Select the row
	var row quota.UsageCounter
	err = q(ctx, r.pool).QueryRow(ctx, `
		SELECT id, user_id, limit_key, cycle_type, cycle_start, cycle_end, used, degraded_used, created_at, updated_at
		FROM usage_counters
		WHERE user_id = $1 AND limit_key = $2 AND cycle_type = $3 AND cycle_start = $4
	`, userID, limitKey, cycleType, start).Scan(
		&row.ID, &row.UserID, &row.LimitKey, &row.CycleType, &row.CycleStart,
		&row.CycleEnd, &row.Used, &row.DegradedUsed, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// 3. Sync live usage if ground truth is higher
	if cycleType == quota.CycleTypeWeekly {
		actualUsed, aerr := r.CountUsageInWindow(ctx, userID, limitKey, start, end)
		if aerr == nil && actualUsed > row.Used {
			_, _ = q(ctx, r.pool).Exec(ctx, `
				UPDATE usage_counters
				SET used = $1, updated_at = now()
				WHERE id = $2
			`, actualUsed, row.ID)
			row.Used = actualUsed
		}
	}

	return &row, nil
}
// IncrementUsage atomikally increments used counter, clamped to hardLimit if hardLimit > 0.
func (r *QuotaRepo) IncrementUsage(ctx context.Context, counterID uuid.UUID, delta int64, hardLimit int64) error {
	if delta <= 0 {
		return nil
	}
	if hardLimit > 0 {
		tag, err := q(ctx, r.pool).Exec(ctx, `
			UPDATE usage_counters
			SET used = used + $1, updated_at = now()
			WHERE id = $2 AND used + $1 <= $3
		`, delta, counterID, hardLimit)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("usage limit exceeded")
		}
		return nil
	}

	_, err := q(ctx, r.pool).Exec(ctx, `
		UPDATE usage_counters
		SET used = used + $1, updated_at = now()
		WHERE id = $2
	`, delta, counterID)
	return err
}

// IncrementDegradedUsage atomikally increments degraded_used counter.
func (r *QuotaRepo) IncrementDegradedUsage(ctx context.Context, counterID uuid.UUID, delta int64, maxDegraded int64) error {
	if delta <= 0 {
		return nil
	}
	if maxDegraded > 0 {
		tag, err := q(ctx, r.pool).Exec(ctx, `
			UPDATE usage_counters
			SET degraded_used = degraded_used + $1, updated_at = now()
			WHERE id = $2 AND degraded_used + $1 <= $3
		`, delta, counterID, maxDegraded)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("degraded usage cap exceeded")
		}
		return nil
	}

	_, err := q(ctx, r.pool).Exec(ctx, `
		UPDATE usage_counters
		SET degraded_used = degraded_used + $1, updated_at = now()
		WHERE id = $2
	`, delta, counterID)
	return err
}

// GetCurrentCounters returns all active counters for the user at the given time.
func (r *QuotaRepo) GetCurrentCounters(ctx context.Context, userID uuid.UUID, now time.Time) ([]quota.UsageCounter, error) {
	rows, err := q(ctx, r.pool).Query(ctx, `
		SELECT id, user_id, limit_key, cycle_type, cycle_start, cycle_end, used, degraded_used, created_at, updated_at
		FROM usage_counters
		WHERE user_id = $1 AND cycle_start <= $2 AND cycle_end > $2
		ORDER BY limit_key, cycle_type
	`, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []quota.UsageCounter
	for rows.Next() {
		var row quota.UsageCounter
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.LimitKey, &row.CycleType, &row.CycleStart,
			&row.CycleEnd, &row.Used, &row.DegradedUsed, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// SetWeeklyAnchor updates weekly_anchor in subscriptions.
func (r *QuotaRepo) SetWeeklyAnchor(ctx context.Context, userID uuid.UUID, anchor time.Time) error {
	_, err := q(ctx, r.pool).Exec(ctx, `
		UPDATE subscriptions
		SET weekly_anchor = $1, updated_at = now()
		WHERE user_id = $2
	`, anchor, userID)
	return err
}

// GetSubscriptionWeeklyAnchor gets the weekly_anchor for user's subscription.
func (r *QuotaRepo) GetSubscriptionWeeklyAnchor(ctx context.Context, userID uuid.UUID) (*time.Time, error) {
	var anchor *time.Time
	err := q(ctx, r.pool).QueryRow(ctx, `
		SELECT COALESCE(weekly_anchor, current_period_start)
		FROM subscriptions
		WHERE user_id = $1
	`, userID).Scan(&anchor)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return anchor, nil
}

// GetUserPlanID retrieves the user's active plan ID from subscriptions (or default free plan).
func (r *QuotaRepo) GetUserPlanID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var planID uuid.UUID
	err := q(ctx, r.pool).QueryRow(ctx, `
		SELECT plan_id FROM subscriptions WHERE user_id = $1
	`, userID).Scan(&planID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = q(ctx, r.pool).QueryRow(ctx, `SELECT id FROM plans WHERE code = 'free'`).Scan(&planID)
			return planID, nil
		}
		return uuid.Nil, err
	}
	return planID, nil
}
// CountUsageInWindow calculates the actual usage for a limit key within [start, end).
func (r *QuotaRepo) CountUsageInWindow(ctx context.Context, userID uuid.UUID, limitKey string, start, end time.Time) (int64, error) {
	switch limitKey {
	case "ai_tokens_per_cycle":
		var sum int64
		err := q(ctx, r.pool).QueryRow(ctx, `
			SELECT COALESCE(SUM(amount), 0)
			FROM token_ledger
			WHERE user_id = $1 AND entry_type = 'commit'
			  AND created_at >= $2 AND created_at < $3
		`, userID, start, end).Scan(&sum)
		if err != nil {
			return 0, err
		}
		// If the weekly window covers or precedes the start of the current token period,
		// check whether token_periods has higher recorded usage.
		var periodTokens int64
		_ = q(ctx, r.pool).QueryRow(ctx, `
			SELECT COALESCE(used_tokens + reserved_tokens, 0)
			FROM token_periods
			WHERE user_id = $1 AND period_start >= $2 AND period_start <= $3
		`, userID, start, end).Scan(&periodTokens)
		if periodTokens > sum {
			sum = periodTokens
		}
		return sum, nil

	case "ocr_scans_per_cycle":
		var count int64
		err := q(ctx, r.pool).QueryRow(ctx, `
			SELECT COUNT(*)
			FROM token_ledger
			WHERE user_id = $1 AND modality = 'vision' AND entry_type = 'reserve'
			  AND created_at >= $2 AND created_at < $3
		`, userID, start, end).Scan(&count)
		return count, err

	case "ai_requests_per_cycle":
		var count int64
		err := q(ctx, r.pool).QueryRow(ctx, `
			SELECT COUNT(*)
			FROM token_ledger
			WHERE user_id = $1 AND entry_type = 'reserve'
			  AND created_at >= $2 AND created_at < $3
		`, userID, start, end).Scan(&count)
		return count, err

	case "max_transactions":
		var count int64
		err := q(ctx, r.pool).QueryRow(ctx, `
			SELECT COUNT(*)
			FROM transactions
			WHERE user_id = $1 AND deleted_at IS NULL
			  AND created_at >= $2 AND created_at < $3
		`, userID, start, end).Scan(&count)
		return count, err

	case "voice_requests_per_day":
		var count int64
		err := q(ctx, r.pool).QueryRow(ctx, `
			SELECT COUNT(*)
			FROM voice_requests
			WHERE user_id = $1
			  AND created_at >= $2 AND created_at < $3
		`, userID, start, end).Scan(&count)
		return count, err

	default:
		return 0, nil
	}
}
