package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

// BudgetRepo stores per-user category budgets.
type BudgetRepo struct {
	pool *pgxpool.Pool
}

func NewBudgetRepo(pool *pgxpool.Pool) *BudgetRepo {
	return &BudgetRepo{pool: pool}
}

func (r *BudgetRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Budget, error) {
	const query = `
		SELECT b.id, b.user_id, b.category_id, COALESCE(ec.name, 'Tanpa Kategori'), b.amount_idr, b.created_at, b.updated_at
		FROM budgets b
		LEFT JOIN expense_categories ec ON ec.id = b.category_id
		WHERE b.user_id = $1
		ORDER BY ec.name`

	rows, err := q(ctx, r.pool).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()

	out := []domain.Budget{}
	for rows.Next() {
		var b domain.Budget
		if err := rows.Scan(&b.ID, &b.UserID, &b.CategoryID, &b.CategoryName, &b.AmountIDR, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan budget: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *BudgetRepo) Upsert(ctx context.Context, userID uuid.UUID, req domain.UpsertBudgetRequest) (*domain.Budget, error) {
	const query = `
		INSERT INTO budgets (user_id, category_id, amount_idr, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id, category_id) DO UPDATE
		SET amount_idr = EXCLUDED.amount_idr, updated_at = now()
		RETURNING id, user_id, category_id, amount_idr, created_at, updated_at`

	var b domain.Budget
	if err := q(ctx, r.pool).QueryRow(ctx, query, userID, req.CategoryID, req.AmountIDR).
		Scan(&b.ID, &b.UserID, &b.CategoryID, &b.AmountIDR, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, fmt.Errorf("upsert budget: %w", err)
	}
	return &b, nil
}

func (r *BudgetRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	const query = `DELETE FROM budgets WHERE id = $1 AND user_id = $2`
	if _, err := q(ctx, r.pool).Exec(ctx, query, id, userID); err != nil {
		return fmt.Errorf("delete budget: %w", err)
	}
	return nil
}
