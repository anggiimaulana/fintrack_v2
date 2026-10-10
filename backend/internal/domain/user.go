package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrEmailTaken is returned when registering an email that already exists.
var ErrEmailTaken = errors.New("email already registered")

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

// Profile is the application user record (unified users table).
type Profile struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	FullName  string    `json:"full_name" db:"full_name"`
	Role      UserRole  `json:"role" db:"role"`
	AvatarURL *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ProfileRepository persists users and reads aggregated admin statistics.
type ProfileRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Profile, error)
	GetByEmail(ctx context.Context, email string) (*Profile, string, error)
	Create(ctx context.Context, p *Profile, passwordHash string) (*Profile, error)
	Update(ctx context.Context, p *Profile) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
	List(ctx context.Context) ([]Profile, error)
	ListWithStats(ctx context.Context) ([]UserStats, *AdminStats, error)
}

type UserStats struct {
	ID                uuid.UUID `json:"id"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	Role              UserRole  `json:"role"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	TransactionCount  int       `json:"transaction_count"`
	TotalIncome       float64   `json:"total_income"`
	TotalExpense      float64   `json:"total_expense"`
	LastTransactionAt *time.Time `json:"last_transaction_at,omitempty"`
}

type AdminStats struct {
	TotalUsers        int     `json:"total_users"`
	ActiveUsers       int     `json:"active_users"`
	TotalTransactions int     `json:"total_transactions"`
	TotalIncome       float64 `json:"total_income"`
	TotalExpense      float64 `json:"total_expense"`
}
