package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SavingsCategory struct {
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

type SavingsCategoryRepository interface {
	Create(ctx context.Context, c *SavingsCategory) error
	GetByID(ctx context.Context, id uuid.UUID) (*SavingsCategory, error)
	List(ctx context.Context, includeInactive bool) ([]SavingsCategory, error)
	Update(ctx context.Context, c *SavingsCategory) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
