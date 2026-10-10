package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

type IncomeCategoryRepo struct {
	pool *pgxpool.Pool
}

func NewIncomeCategoryRepo(pool *pgxpool.Pool) *IncomeCategoryRepo {
	return &IncomeCategoryRepo{pool: pool}
}

func (r *IncomeCategoryRepo) Create(ctx context.Context, c *domain.IncomeCategory) error {
	query := `
		INSERT INTO income_categories (name, icon, color, description, sort_order, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		c.Name, c.Icon, c.Color, c.Description, c.SortOrder, c.CreatedBy,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *IncomeCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.IncomeCategory, error) {
	query := `SELECT id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at
		FROM income_categories WHERE id = $1`
	c := &domain.IncomeCategory{}
	err := q(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Icon, &c.Color, &c.Description, &c.IsActive,
		&c.SortOrder, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get income category: %w", err)
	}
	return c, nil
}

func (r *IncomeCategoryRepo) List(ctx context.Context, includeInactive bool) ([]domain.IncomeCategory, error) {
	query := `SELECT id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at
		FROM income_categories`
	if !includeInactive {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY sort_order ASC, name ASC`

	rows, err := q(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list income categories: %w", err)
	}
	defer rows.Close()

	var categories []domain.IncomeCategory
	for rows.Next() {
		var c domain.IncomeCategory
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Icon, &c.Color, &c.Description, &c.IsActive,
			&c.SortOrder, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan income category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *IncomeCategoryRepo) Update(ctx context.Context, c *domain.IncomeCategory) error {
	query := `UPDATE income_categories SET
		name = COALESCE($2, name), icon = COALESCE($3, icon), color = COALESCE($4, color),
		description = $5, sort_order = COALESCE($6, sort_order)
		WHERE id = $1 RETURNING updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		c.ID, c.Name, c.Icon, c.Color, c.Description, c.SortOrder,
	).Scan(&c.UpdatedAt)
}

func (r *IncomeCategoryRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE income_categories SET is_active = false WHERE id = $1`, id)
	return err
}

type ExpenseCategoryRepo struct {
	pool *pgxpool.Pool
}

func NewExpenseCategoryRepo(pool *pgxpool.Pool) *ExpenseCategoryRepo {
	return &ExpenseCategoryRepo{pool: pool}
}

func (r *ExpenseCategoryRepo) Create(ctx context.Context, c *domain.ExpenseCategory) error {
	query := `
		INSERT INTO expense_categories (name, icon, color, description, sort_order, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		c.Name, c.Icon, c.Color, c.Description, c.SortOrder, c.CreatedBy,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *ExpenseCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ExpenseCategory, error) {
	query := `SELECT id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at
		FROM expense_categories WHERE id = $1`
	c := &domain.ExpenseCategory{}
	err := q(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Icon, &c.Color, &c.Description, &c.IsActive,
		&c.SortOrder, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get expense category: %w", err)
	}
	return c, nil
}

func (r *ExpenseCategoryRepo) List(ctx context.Context, includeInactive bool) ([]domain.ExpenseCategory, error) {
	query := `SELECT id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at
		FROM expense_categories`
	if !includeInactive {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY sort_order ASC, name ASC`

	rows, err := q(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list expense categories: %w", err)
	}
	defer rows.Close()

	var categories []domain.ExpenseCategory
	for rows.Next() {
		var c domain.ExpenseCategory
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Icon, &c.Color, &c.Description, &c.IsActive,
			&c.SortOrder, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan expense category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *ExpenseCategoryRepo) Update(ctx context.Context, c *domain.ExpenseCategory) error {
	query := `UPDATE expense_categories SET
		name = COALESCE($2, name), icon = COALESCE($3, icon), color = COALESCE($4, color),
		description = $5, sort_order = COALESCE($6, sort_order)
		WHERE id = $1 RETURNING updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		c.ID, c.Name, c.Icon, c.Color, c.Description, c.SortOrder,
	).Scan(&c.UpdatedAt)
}

func (r *ExpenseCategoryRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE expense_categories SET is_active = false WHERE id = $1`, id)
	return err
}

type ExpenseSubcategoryRepo struct {
	pool *pgxpool.Pool
}

func NewExpenseSubcategoryRepo(pool *pgxpool.Pool) *ExpenseSubcategoryRepo {
	return &ExpenseSubcategoryRepo{pool: pool}
}

func (r *ExpenseSubcategoryRepo) Create(ctx context.Context, s *domain.ExpenseSubcategory) error {
	query := `
		INSERT INTO expense_subcategories (category_id, name, icon, color, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		s.CategoryID, s.Name, s.Icon, s.Color, s.SortOrder,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *ExpenseSubcategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ExpenseSubcategory, error) {
	query := `SELECT id, category_id, name, icon, color, is_active, sort_order, created_at, updated_at
		FROM expense_subcategories WHERE id = $1`
	s := &domain.ExpenseSubcategory{}
	err := q(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&s.ID, &s.CategoryID, &s.Name, &s.Icon, &s.Color, &s.IsActive,
		&s.SortOrder, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subcategory: %w", err)
	}
	return s, nil
}

func (r *ExpenseSubcategoryRepo) ListByCategory(ctx context.Context, categoryID uuid.UUID, includeInactive bool) ([]domain.ExpenseSubcategory, error) {
	query := `SELECT id, category_id, name, icon, color, is_active, sort_order, created_at, updated_at
		FROM expense_subcategories WHERE category_id = $1`
	if !includeInactive {
		query += ` AND is_active = true`
	}
	query += ` ORDER BY sort_order ASC, name ASC`

	rows, err := q(ctx, r.pool).Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to list subcategories: %w", err)
	}
	defer rows.Close()

	var subcategories []domain.ExpenseSubcategory
	for rows.Next() {
		var s domain.ExpenseSubcategory
		if err := rows.Scan(
			&s.ID, &s.CategoryID, &s.Name, &s.Icon, &s.Color, &s.IsActive,
			&s.SortOrder, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan subcategory: %w", err)
		}
		subcategories = append(subcategories, s)
	}
	return subcategories, nil
}

func (r *ExpenseSubcategoryRepo) Update(ctx context.Context, s *domain.ExpenseSubcategory) error {
	query := `UPDATE expense_subcategories SET
		name = COALESCE($2, name), icon = COALESCE($3, icon), color = COALESCE($4, color),
		sort_order = COALESCE($5, sort_order)
		WHERE id = $1 RETURNING updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		s.ID, s.Name, s.Icon, s.Color, s.SortOrder,
	).Scan(&s.UpdatedAt)
}

func (r *ExpenseSubcategoryRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE expense_subcategories SET is_active = false WHERE id = $1`, id)
	return err
}
