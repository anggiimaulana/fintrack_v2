package featureflag

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository reads feature flags from the feature_flags table.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository builds a repository backed by the given pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// GetByKey returns the flag with the given key, or nil when absent.
func (r *PostgresRepository) GetByKey(ctx context.Context, key string) (*Flag, error) {
	var f Flag
	err := r.pool.QueryRow(ctx,
		`SELECT key, name, description, is_enabled, rollout, rules
         FROM feature_flags WHERE key = $1`, key).
		Scan(&f.Key, &f.Name, &f.Description, &f.IsEnabled, &f.Rollout, &f.Rules)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// List returns all flags ordered by key.
func (r *PostgresRepository) List(ctx context.Context) ([]Flag, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT key, name, description, is_enabled, rollout, rules
         FROM feature_flags ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Flag
	for rows.Next() {
		var f Flag
		if err := rows.Scan(&f.Key, &f.Name, &f.Description, &f.IsEnabled, &f.Rollout, &f.Rules); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
