package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

type SavingsCategoryRepo struct {
	pool *pgxpool.Pool
}

func NewSavingsCategoryRepo(pool *pgxpool.Pool) *SavingsCategoryRepo {
	return &SavingsCategoryRepo{pool: pool}
}

func (r *SavingsCategoryRepo) Create(ctx context.Context, c *domain.SavingsCategory) error {
	query := `
		INSERT INTO savings_categories (name, icon, color, description, sort_order, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, is_active, created_at, updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		c.Name, c.Icon, c.Color, c.Description, c.SortOrder, c.CreatedBy,
	).Scan(&c.ID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
}

func (r *SavingsCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.SavingsCategory, error) {
	query := `
		SELECT id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at
		FROM savings_categories WHERE id = $1`

	c := &domain.SavingsCategory{}
	err := q(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Icon, &c.Color, &c.Description, &c.IsActive, &c.SortOrder,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get savings category: %w", err)
	}
	return c, nil
}

func (r *SavingsCategoryRepo) List(ctx context.Context, includeInactive bool) ([]domain.SavingsCategory, error) {
	query := `SELECT id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at
		FROM savings_categories`
	if !includeInactive {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY sort_order ASC, name ASC`

	rows, err := q(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list savings categories: %w", err)
	}
	defer rows.Close()

	var categories []domain.SavingsCategory
	for rows.Next() {
		var c domain.SavingsCategory
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Icon, &c.Color, &c.Description, &c.IsActive, &c.SortOrder,
			&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan savings category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *SavingsCategoryRepo) Update(ctx context.Context, c *domain.SavingsCategory) error {
	query := `UPDATE savings_categories SET
		name = COALESCE($2, name),
		icon = COALESCE($3, icon),
		color = COALESCE($4, color),
		description = COALESCE($5, description),
		sort_order = COALESCE($6, sort_order),
		is_active = COALESCE($7, is_active)
		WHERE id = $1
		RETURNING updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		c.ID, c.Name, c.Icon, c.Color, c.Description, c.SortOrder, c.IsActive,
	).Scan(&c.UpdatedAt)
}

func (r *SavingsCategoryRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE savings_categories SET is_active = false WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
