package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Budget is a per-category monthly spending ceiling.
type Budget struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	CategoryID uuid.UUID `json:"category_id" db:"category_id"`
	// CategoryName is denormalised for display (joined on read).
	CategoryName string    `json:"category_name"`
	AmountIDR    int64     `json:"amount_idr" db:"amount_idr"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// BudgetStatus is a budget joined with the actual spend for a period.
type BudgetStatus struct {
	Budget
	// Actual is the spend in the period for the budget's category.
	Actual float64 `json:"actual"`
	// Remaining is AmountIDR - Actual (can be negative when over budget).
	Remaining float64 `json:"remaining"`
	// UsedPct is Actual / AmountIDR * 100 (0 when the budget is 0).
	UsedPct float64 `json:"used_pct"`
	// Over is true when Actual exceeds AmountIDR.
	Over bool `json:"over"`
}

// UpsertBudgetRequest is the write shape for a budget.
type UpsertBudgetRequest struct {
	CategoryID uuid.UUID `json:"category_id" binding:"required"`
	AmountIDR  int64     `json:"amount_idr" binding:"gte=0"`
}

// BudgetRepository persists per-user budgets.
type BudgetRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Budget, error)
	Upsert(ctx context.Context, userID uuid.UUID, req UpsertBudgetRequest) (*Budget, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}
