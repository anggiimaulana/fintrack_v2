package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IncomeCategory struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Icon        string     `json:"icon" db:"icon"`
	Color       string     `json:"color" db:"color"`
	Description *string    `json:"description,omitempty" db:"description"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	SortOrder   int        `json:"sort_order" db:"sort_order"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type ExpenseCategory struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Icon        string     `json:"icon" db:"icon"`
	Color       string     `json:"color" db:"color"`
	Description *string    `json:"description,omitempty" db:"description"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	SortOrder   int        `json:"sort_order" db:"sort_order"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type ExpenseSubcategory struct {
	ID         uuid.UUID `json:"id" db:"id"`
	CategoryID uuid.UUID `json:"category_id" db:"category_id"`
	Name       string    `json:"name" db:"name"`
	Icon       string    `json:"icon" db:"icon"`
	Color      string    `json:"color" db:"color"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	SortOrder  int       `json:"sort_order" db:"sort_order"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type IncomeCategoryRepository interface {
	Create(ctx context.Context, c *IncomeCategory) error
	GetByID(ctx context.Context, id uuid.UUID) (*IncomeCategory, error)
	List(ctx context.Context, includeInactive bool) ([]IncomeCategory, error)
	Update(ctx context.Context, c *IncomeCategory) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type ExpenseCategoryRepository interface {
	Create(ctx context.Context, c *ExpenseCategory) error
	GetByID(ctx context.Context, id uuid.UUID) (*ExpenseCategory, error)
	List(ctx context.Context, includeInactive bool) ([]ExpenseCategory, error)
	Update(ctx context.Context, c *ExpenseCategory) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type ExpenseSubcategoryRepository interface {
	Create(ctx context.Context, s *ExpenseSubcategory) error
	GetByID(ctx context.Context, id uuid.UUID) (*ExpenseSubcategory, error)
	ListByCategory(ctx context.Context, categoryID uuid.UUID, includeInactive bool) ([]ExpenseSubcategory, error)
	Update(ctx context.Context, s *ExpenseSubcategory) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type CreateCategoryRequest struct {
	Name        string  `json:"name" binding:"required"`
	Icon        string  `json:"icon"`
	Color       string  `json:"color"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Icon        *string `json:"icon"`
	Color       *string `json:"color"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
	IsActive    *bool   `json:"is_active"`
}

type CreateSubcategoryRequest struct {
	CategoryID uuid.UUID `json:"category_id" binding:"required"`
	Name       string    `json:"name" binding:"required"`
	Icon       string    `json:"icon"`
	Color      string    `json:"color"`
	SortOrder  int       `json:"sort_order"`
}

type UpdateSubcategoryRequest struct {
	Name      *string `json:"name"`
	Icon      *string `json:"icon"`
	Color     *string `json:"color"`
	SortOrder *int    `json:"sort_order"`
	IsActive  *bool   `json:"is_active"`
}
