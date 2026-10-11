package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/payment"
)

// SettingsRepo implements payment.SettingsRepository over the settings table.
type SettingsRepo struct {
	pool *pgxpool.Pool
}

// NewSettingsRepo builds the repository.
func NewSettingsRepo(pool *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{pool: pool}
}

// GetString returns the JSON-decoded string value for key ("" when absent).
func (r *SettingsRepo) GetString(ctx context.Context, key string) (string, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&raw)
	if err != nil {
		return "", nil // absent -> empty, not an error
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		// Value may be stored as a bare string; return as-is.
		return string(raw), nil
	}
	return s, nil
}

// GetInt returns the JSON-decoded integer value for key (0 when absent).
func (r *SettingsRepo) GetInt(ctx context.Context, key string) (int64, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&raw)
	if err != nil {
		return 0, nil
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, nil
	}
	return n, nil
}

// Set upserts a setting value.
func (r *SettingsRepo) Set(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, now())
         ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, key, raw)
	return err
}

// compile-time assertion.
var _ payment.SettingsRepository = (*SettingsRepo)(nil)
