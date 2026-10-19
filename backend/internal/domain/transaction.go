package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrConflict signals an optimistic-locking failure (stale updated_at).
var ErrConflict = errors.New("record was modified by another request")

// TxRunner executes a function within a single database transaction, exposing a
// transaction-scoped context and per-key advisory locks.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
	AdvisoryXactLock(ctx context.Context, key string) error
}

type TransactionType string

const (
	TransactionIncome  TransactionType = "income"
	TransactionExpense TransactionType = "expense"
	TransactionSavings TransactionType = "savings"
)

type RecurringInterval string

const (
	RecurringDaily   RecurringInterval = "daily"
	RecurringWeekly  RecurringInterval = "weekly"
	RecurringMonthly RecurringInterval = "monthly"
)

type TransactionSource string

const (
	SourceManual  TransactionSource = "manual"
	SourceAI      TransactionSource = "ai"
	SourceVoice   TransactionSource = "voice"
	SourceReceipt TransactionSource = "receipt"
)

type Transaction struct {
	ID                uuid.UUID          `json:"id" db:"id"`
	UserID            uuid.UUID          `json:"user_id" db:"user_id"`
	Type              TransactionType    `json:"type" db:"type"`
	Amount            float64            `json:"amount" db:"amount"`
	// CurrencyCode is the currency of Amount (the original user input).
	CurrencyCode string `json:"currency_code" db:"currency_code"`
	// ExchangeRate is the snapshot: 1 unit CurrencyCode = ExchangeRate IDR.
	ExchangeRate float64 `json:"exchange_rate" db:"exchange_rate"`
	// AmountBase is the IDR value; DB-generated (round(amount*exchange_rate,2)).
	AmountBase float64 `json:"amount_base" db:"amount_base"`
	// RateAt is when the snapshot rate was effective (nil for IDR).
	RateAt            *time.Time         `json:"rate_at,omitempty" db:"rate_at"`
	IncomeCategoryID  *uuid.UUID         `json:"income_category_id,omitempty" db:"income_category_id"`
	ExpenseCategoryID *uuid.UUID         `json:"expense_category_id,omitempty" db:"expense_category_id"`
	SavingsCategoryID *uuid.UUID         `json:"savings_category_id,omitempty" db:"savings_category_id"`
	SubcategoryID     *uuid.UUID         `json:"subcategory_id,omitempty" db:"subcategory_id"`
	Description       *string            `json:"description,omitempty" db:"description"`
	Notes             *string            `json:"notes,omitempty" db:"notes"`
	Tags              []string           `json:"tags,omitempty" db:"tags"`
	TransactionDate   time.Time          `json:"transaction_date" db:"transaction_date"`
	IsRecurring       bool               `json:"is_recurring" db:"is_recurring"`
	RecurringInterval *RecurringInterval `json:"recurring_interval,omitempty" db:"recurring_interval"`
	Source            TransactionSource  `json:"source" db:"source"`
	IsDeleted         bool               `json:"is_deleted" db:"is_deleted"`
	DeletedAt         *time.Time         `json:"deleted_at,omitempty" db:"deleted_at"`
	CreatedAt         time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" db:"updated_at"`
}

type TimeSeriesPoint struct {
	Period  string  `json:"period"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Savings float64 `json:"savings"`
}

type CategoryExpenseSummary struct {
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
	Count        int     `json:"count"`
}

type CategoryTimeSeriesRow struct {
	Period       string  `json:"period"`
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *Transaction) error
	GetByID(ctx context.Context, id, userID uuid.UUID) (*Transaction, error)
	List(ctx context.Context, userID uuid.UUID, filter TransactionFilter) ([]Transaction, int, error)
	Update(ctx context.Context, tx *Transaction, expectedUpdatedAt *time.Time) error
	SoftDelete(ctx context.Context, id, userID uuid.UUID) error
	GetStats(ctx context.Context, userID uuid.UUID, period StatsPeriod) (*TransactionStats, error)
	GetMonthlyExpense(ctx context.Context, userID uuid.UUID, year, month int) (float64, error)
	GetCategoryExpensesByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]CategoryExpenseSummary, error)
	GetCategoryIncomesByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]CategoryExpenseSummary, error)
	GetCategorySavingsByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]CategoryExpenseSummary, error)
	GetDailyTransactionCount(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error)
	GetRecentByUser(ctx context.Context, userID uuid.UUID, limit int) ([]Transaction, error)
	GetTimeSeries(ctx context.Context, userID uuid.UUID, start, end time.Time, granularity string) ([]TimeSeriesPoint, error)
	GetCategoryTimeSeries(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]CategoryTimeSeriesRow, error)
	HasTransactions(ctx context.Context, userID uuid.UUID) (bool, error)
}

type TransactionFilter struct {
	Page       int
	PerPage    int
	Type       *TransactionType
	CategoryID *uuid.UUID
	DateFrom   *time.Time
	DateTo     *time.Time
	Search     string
	SortBy     string
	SortOrder  string
}

type StatsPeriod struct {
	StartDate time.Time
	EndDate   time.Time
}

type TransactionStats struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	TotalSavings float64 `json:"total_savings"`
	NetBalance   float64 `json:"net_balance"`
	IncomeCount  int     `json:"income_count"`
	ExpenseCount int     `json:"expense_count"`
	SavingsCount int     `json:"savings_count"`
}

type CreateTransactionRequest struct {
	Type              TransactionType    `json:"type" binding:"required"`
	Amount            float64            `json:"amount" binding:"required,gt=0"`
	// CurrencyCode is optional; server defaults to the user's display currency.
	CurrencyCode      string             `json:"currency_code"`
	IncomeCategoryID  *uuid.UUID         `json:"income_category_id"`
	ExpenseCategoryID *uuid.UUID         `json:"expense_category_id"`
	SavingsCategoryID *uuid.UUID         `json:"savings_category_id"`
	SubcategoryID     *uuid.UUID         `json:"subcategory_id"`
	Description       *string            `json:"description"`
	Notes             *string            `json:"notes"`
	Tags              []string           `json:"tags"`
	TransactionDate   *time.Time         `json:"transaction_date"`
	IsRecurring       bool               `json:"is_recurring"`
	RecurringInterval *RecurringInterval `json:"recurring_interval"`
	Source            TransactionSource  `json:"source"`
}

type UpdateTransactionRequest struct {
	Amount            *float64           `json:"amount"`
	CurrencyCode      *string            `json:"currency_code"`
	IncomeCategoryID  *uuid.UUID         `json:"income_category_id"`
	ExpenseCategoryID *uuid.UUID         `json:"expense_category_id"`
	SavingsCategoryID *uuid.UUID         `json:"savings_category_id"`
	SubcategoryID     *uuid.UUID         `json:"subcategory_id"`
	Description       *string            `json:"description"`
	Notes             *string            `json:"notes"`
	Tags              []string           `json:"tags"`
	TransactionDate   *time.Time         `json:"transaction_date"`
	IsRecurring       *bool              `json:"is_recurring"`
	RecurringInterval *RecurringInterval `json:"recurring_interval"`
	Source            *TransactionSource `json:"source"`
	// UpdatedAt enables optimistic locking: when set, the update only applies
	// if the row's updated_at matches, otherwise ErrConflict is returned.
	UpdatedAt *time.Time `json:"updated_at"`
}
