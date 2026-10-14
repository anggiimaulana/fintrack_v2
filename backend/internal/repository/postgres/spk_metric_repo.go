package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

// SPKMetricRepo stores the super-admin default SPK model and per-user overrides.
//
// Two tables back it (see migration 0019): `spk_metric_defaults` holds versioned
// defaults (one active) and `spk_metric_overrides` holds at most one row per
// user. Resolution (override → default → compiled fallback) lives in the
// usecase so the precedence rule is testable without a database.
type SPKMetricRepo struct {
	pool *pgxpool.Pool
}

func NewSPKMetricRepo(pool *pgxpool.Pool) *SPKMetricRepo {
	return &SPKMetricRepo{pool: pool}
}

func (r *SPKMetricRepo) GetActiveDefault(ctx context.Context) (*domain.SPKMetricModel, error) {
	const query = `
		SELECT name, indicators, status_thresholds
		FROM spk_metric_defaults
		WHERE is_active
		LIMIT 1`

	var (
		name       string
		indicators []byte
		thresholds []byte
	)
	err := q(ctx, r.pool).QueryRow(ctx, query).Scan(&name, &indicators, &thresholds)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get active spk default: %w", err)
	}
	return decodeMetricModel(name, "default", indicators, thresholds)
}

func (r *SPKMetricRepo) GetUserOverride(ctx context.Context, userID string) (*domain.SPKMetricModel, error) {
	const query = `
		SELECT indicators, status_thresholds
		FROM spk_metric_overrides
		WHERE user_id = $1`

	var (
		indicators []byte
		thresholds []byte
	)
	err := q(ctx, r.pool).QueryRow(ctx, query, userID).Scan(&indicators, &thresholds)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get spk user override: %w", err)
	}
	return decodeMetricModel("", "user", indicators, thresholds)
}

func (r *SPKMetricRepo) UpsertUserOverride(ctx context.Context, userID string, m domain.SPKMetricModel) error {
	indicators, err := json.Marshal(m.Indicators)
	if err != nil {
		return fmt.Errorf("marshal spk indicators: %w", err)
	}
	thresholds, err := json.Marshal(m.Thresholds)
	if err != nil {
		return fmt.Errorf("marshal spk thresholds: %w", err)
	}

	const query = `
		INSERT INTO spk_metric_overrides (user_id, indicators, status_thresholds, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE
		SET indicators = EXCLUDED.indicators,
		    status_thresholds = EXCLUDED.status_thresholds,
		    updated_at = now()`
	if _, err := q(ctx, r.pool).Exec(ctx, query, userID, indicators, thresholds); err != nil {
		return fmt.Errorf("upsert spk user override: %w", err)
	}
	return nil
}

func (r *SPKMetricRepo) DeleteUserOverride(ctx context.Context, userID string) error {
	const query = `DELETE FROM spk_metric_overrides WHERE user_id = $1`
	if _, err := q(ctx, r.pool).Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("delete spk user override: %w", err)
	}
	return nil
}

// ReplaceDefault deactivates the current active default and inserts a new active
// one atomically, so readers never observe zero active defaults.
func (r *SPKMetricRepo) ReplaceDefault(ctx context.Context, name, description string, m domain.SPKMetricModel, updatedBy *string) (*domain.SPKMetricModel, error) {
	indicators, err := json.Marshal(m.Indicators)
	if err != nil {
		return nil, fmt.Errorf("marshal spk indicators: %w", err)
	}
	thresholds, err := json.Marshal(m.Thresholds)
	if err != nil {
		return nil, fmt.Errorf("marshal spk thresholds: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin spk default tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Deactivate first: the partial unique index allows only one active row, so
	// the new insert must not race it.
	if _, err := tx.Exec(ctx, `UPDATE spk_metric_defaults SET is_active = false WHERE is_active`); err != nil {
		return nil, fmt.Errorf("deactivate spk default: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO spk_metric_defaults (name, description, indicators, status_thresholds, is_active, updated_by)
		VALUES ($1, $2, $3, $4, true, $5)`,
		name, description, indicators, thresholds, updatedBy); err != nil {
		return nil, fmt.Errorf("insert spk default: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit spk default tx: %w", err)
	}

	out := m
	out.Source = "default"
	out.Name = name
	return &out, nil
}

// decodeMetricModel unmarshals the two JSONB columns into a model, tolerating a
// malformed/empty payload by returning the model with whatever decoded (the
// usecase validates before persisting, so this is defence in depth).
func decodeMetricModel(name, source string, indicators, thresholds []byte) (*domain.SPKMetricModel, error) {
	m := &domain.SPKMetricModel{Source: source, Name: name}
	if len(indicators) > 0 {
		if err := json.Unmarshal(indicators, &m.Indicators); err != nil {
			return nil, fmt.Errorf("unmarshal spk indicators: %w", err)
		}
	}
	if len(thresholds) > 0 {
		if err := json.Unmarshal(thresholds, &m.Thresholds); err != nil {
			return nil, fmt.Errorf("unmarshal spk thresholds: %w", err)
		}
	}
	return m, nil
}
