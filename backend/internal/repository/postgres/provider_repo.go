package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

type ProviderRepo struct {
	pool *pgxpool.Pool
}

func NewProviderRepo(pool *pgxpool.Pool) *ProviderRepo {
	return &ProviderRepo{pool: pool}
}

func (r *ProviderRepo) List(ctx context.Context) ([]domain.AIProvider, error) {
	query := `SELECT id, name, display_name, supports_fast, supports_reasoning, is_active, created_at
		FROM ai_providers ORDER BY display_name ASC`

	rows, err := q(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	defer rows.Close()

	var providers []domain.AIProvider
	for rows.Next() {
		var p domain.AIProvider
		if err := rows.Scan(
			&p.ID, &p.Name, &p.DisplayName, &p.SupportsFast, &p.SupportsReasoning, &p.IsActive, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, p)
	}
	return providers, nil
}

func (r *ProviderRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.AIProvider, error) {
	query := `SELECT id, name, display_name, supports_fast, supports_reasoning, is_active, created_at
		FROM ai_providers WHERE id = $1`

	p := &domain.AIProvider{}
	err := q(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.SupportsFast, &p.SupportsReasoning, &p.IsActive, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	return p, nil
}

func (r *ProviderRepo) GetByName(ctx context.Context, name string) (*domain.AIProvider, error) {
	query := `SELECT id, name, display_name, supports_fast, supports_reasoning, is_active, created_at
		FROM ai_providers WHERE name = $1`

	p := &domain.AIProvider{}
	err := q(ctx, r.pool).QueryRow(ctx, query, name).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.SupportsFast, &p.SupportsReasoning, &p.IsActive, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	return p, nil
}

func (r *ProviderRepo) Create(ctx context.Context, provider *domain.AIProvider) error {
	query := `
		INSERT INTO ai_providers (name, display_name, supports_fast, supports_reasoning, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		provider.Name, provider.DisplayName, provider.SupportsFast, provider.SupportsReasoning, provider.IsActive,
	).Scan(&provider.ID, &provider.CreatedAt)
}

func (r *ProviderRepo) Update(ctx context.Context, provider *domain.AIProvider) error {
	query := `UPDATE ai_providers SET
		name = COALESCE($2, name),
		display_name = COALESCE($3, display_name),
		supports_fast = COALESCE($4, supports_fast),
		supports_reasoning = COALESCE($5, supports_reasoning),
		is_active = COALESCE($6, is_active)
		WHERE id = $1
		RETURNING created_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		provider.ID, provider.Name, provider.DisplayName, provider.SupportsFast, provider.SupportsReasoning, provider.IsActive,
	).Scan(&provider.CreatedAt)
}
