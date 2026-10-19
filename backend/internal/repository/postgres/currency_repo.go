package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/currency"
)

// CurrencyRepo persists the dynamic currency catalog and append-only rates.
type CurrencyRepo struct {
	pool *pgxpool.Pool
}

func NewCurrencyRepo(pool *pgxpool.Pool) *CurrencyRepo { return &CurrencyRepo{pool: pool} }

const currencyCols = "code, name, symbol, locale, decimal_digits, is_active, is_base, sort_order, rate_source"

func scanCurrency(row interface{ Scan(...interface{}) error }) (currency.Meta, error) {
	var m currency.Meta
	err := row.Scan(&m.Code, &m.Name, &m.Symbol, &m.Locale, &m.DecimalDigits,
		&m.IsActive, &m.IsBase, &m.SortOrder, &m.RateSource)
	return m, err
}

func (r *CurrencyRepo) List(ctx context.Context, includeInactive bool) ([]currency.Meta, error) {
	query := `SELECT ` + currencyCols + ` FROM currencies`
	if !includeInactive {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY sort_order, code`

	rows, err := q(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}
	defer rows.Close()

	out := []currency.Meta{}
	for rows.Next() {
		m, err := scanCurrency(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan currency: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *CurrencyRepo) Get(ctx context.Context, code string) (*currency.Meta, error) {
	query := `SELECT ` + currencyCols + ` FROM currencies WHERE code = upper($1)`
	m, err := scanCurrency(q(ctx, r.pool).QueryRow(ctx, query, code))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get currency: %w", err)
	}
	return &m, nil
}

func (r *CurrencyRepo) RateAsOf(ctx context.Context, code string, at time.Time) (*currency.Rate, error) {
	query := `
		SELECT id, currency_code, rate_to_base, effective_at, source, note, created_by, created_at
		FROM currency_rates
		WHERE currency_code = upper($1) AND effective_at <= $2
		ORDER BY effective_at DESC, created_at DESC
		LIMIT 1`
	var rt currency.Rate
	err := q(ctx, r.pool).QueryRow(ctx, query, code, at).Scan(
		&rt.ID, &rt.CurrencyCode, &rt.RateToBase, &rt.EffectiveAt,
		&rt.Source, &rt.Note, &rt.CreatedBy, &rt.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get rate: %w", err)
	}
	return &rt, nil
}

func (r *CurrencyRepo) InsertRate(ctx context.Context, rt currency.Rate) error {
	query := `
		INSERT INTO currency_rates (currency_code, rate_to_base, effective_at, source, note, created_by)
		VALUES (upper($1), $2, $3, $4, $5, $6)
		ON CONFLICT (currency_code, effective_at) DO NOTHING`
	source := rt.Source
	if source == "" {
		source = "manual"
	}
	tag, err := q(ctx, r.pool).Exec(ctx, query,
		rt.CurrencyCode, rt.RateToBase, rt.EffectiveAt, source, rt.Note, rt.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to insert rate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return currency.ErrRateDuplicate
	}
	return nil
}

func (r *CurrencyRepo) ListRates(ctx context.Context, code string, limit int) ([]currency.Rate, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `
		SELECT id, currency_code, rate_to_base, effective_at, source, note, created_by, created_at
		FROM currency_rates
		WHERE currency_code = upper($1)
		ORDER BY effective_at DESC, created_at DESC
		LIMIT $2`
	rows, err := q(ctx, r.pool).Query(ctx, query, code, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list rates: %w", err)
	}
	defer rows.Close()

	out := []currency.Rate{}
	for rows.Next() {
		var rt currency.Rate
		if err := rows.Scan(&rt.ID, &rt.CurrencyCode, &rt.RateToBase, &rt.EffectiveAt,
			&rt.Source, &rt.Note, &rt.CreatedBy, &rt.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rate: %w", err)
		}
		out = append(out, rt)
	}
	return out, rows.Err()
}

// UpsertCurrency inserts or updates catalog metadata (Super Admin operation).
func (r *CurrencyRepo) UpsertCurrency(ctx context.Context, m currency.Meta) error {
	query := `
		INSERT INTO currencies (code, name, symbol, locale, decimal_digits, is_active, is_base, sort_order, rate_source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			symbol = EXCLUDED.symbol,
			locale = EXCLUDED.locale,
			decimal_digits = EXCLUDED.decimal_digits,
			is_active = EXCLUDED.is_active,
			sort_order = EXCLUDED.sort_order,
			rate_source = EXCLUDED.rate_source
		WHERE currencies.is_base = false`
	_, err := q(ctx, r.pool).Exec(ctx, query, m.Code, m.Name, m.Symbol, m.Locale,
		m.DecimalDigits, m.IsActive, m.IsBase, m.SortOrder, m.RateSource)
	if err != nil {
		return fmt.Errorf("failed to upsert currency: %w", err)
	}
	return nil
}

// CountRates reports how many rate rows exist for a currency (activation guard).
func (r *CurrencyRepo) CountRates(ctx context.Context, code string) (int, error) {
	var n int
	err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT COUNT(*) FROM currency_rates WHERE currency_code = upper($1)`, code).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("failed to count rates: %w", err)
	}
	return n, nil
}
