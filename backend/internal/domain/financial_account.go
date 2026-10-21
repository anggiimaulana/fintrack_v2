package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// FinancialAccountType represents the class of an account or wallet.
type FinancialAccountType string

const (
	AccountTypeBank       FinancialAccountType = "bank"
	AccountTypeEwallet    FinancialAccountType = "ewallet"
	AccountTypeCash       FinancialAccountType = "cash"
	AccountTypeInvestment FinancialAccountType = "investment"
)

// Valid reports whether t is a recognized account type.
func (t FinancialAccountType) Valid() bool {
	switch t {
	case AccountTypeBank, AccountTypeEwallet, AccountTypeCash, AccountTypeInvestment:
		return true
	default:
		return false
	}
}

// FinancialAccount is an individual bank account, digital wallet, or cash pool owned by a user.
type FinancialAccount struct {
	ID                uuid.UUID            `json:"id" db:"id"`
	UserID            uuid.UUID            `json:"user_id" db:"user_id"`
	Name              string               `json:"name" db:"name"`
	AccountType       FinancialAccountType `json:"account_type" db:"account_type"`
	InstitutionCode   string               `json:"institution_code" db:"institution_code"`
	AccountNumberMask *string              `json:"account_number_mask,omitempty" db:"account_number_mask"`
	CurrencyCode      string               `json:"currency_code" db:"currency_code"`
	CurrentBalance    float64              `json:"current_balance" db:"current_balance"`
	InitialBalance    float64              `json:"initial_balance" db:"initial_balance"`
	ColorHex          string               `json:"color_hex" db:"color_hex"`
	IconKey           string               `json:"icon_key" db:"icon_key"`
	IsActive          bool                 `json:"is_active" db:"is_active"`
	CreatedAt         time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at" db:"updated_at"`
}

// UpsertFinancialAccountRequest is the input payload for creating or updating an account.
type UpsertFinancialAccountRequest struct {
	Name              string               `json:"name" binding:"required"`
	AccountType       FinancialAccountType `json:"account_type" binding:"required"`
	InstitutionCode   string               `json:"institution_code" binding:"required"`
	AccountNumberMask *string              `json:"account_number_mask"`
	CurrencyCode      string               `json:"currency_code"`
	InitialBalance    float64              `json:"initial_balance"`
	ColorHex          string               `json:"color_hex"`
	IconKey           string               `json:"icon_key"`
	IsActive          *bool                `json:"is_active"`
}

// FinancialAccountSummary holds calculated metrics across user accounts.
type FinancialAccountSummary struct {
	TotalNetWorthIDR  float64                  `json:"total_net_worth_idr"`
	AccountCount      int                      `json:"account_count"`
	Accounts          []FinancialAccount       `json:"accounts"`
	BreakdownByType   map[FinancialAccountType]float64 `json:"breakdown_by_type"`
	BreakdownByInst   map[string]float64       `json:"breakdown_by_institution"`
}

// ImportBatchStatus tracks the lifecycle of an uploaded statement file batch.
type ImportBatchStatus string

const (
	BatchStatusProcessing ImportBatchStatus = "processing"
	BatchStatusCompleted  ImportBatchStatus = "completed"
	BatchStatusFailed      ImportBatchStatus = "failed"
	BatchStatusRolledBack  ImportBatchStatus = "rolled_back"
)

// Valid reports whether s is a recognized batch status.
func (s ImportBatchStatus) Valid() bool {
	switch s {
	case BatchStatusProcessing, BatchStatusCompleted, BatchStatusFailed, BatchStatusRolledBack:
		return true
	default:
		return false
	}
}

// ImportBatch is the audit record of an imported statement file.
type ImportBatch struct {
	ID                uuid.UUID         `json:"id" db:"id"`
	UserID            uuid.UUID         `json:"user_id" db:"user_id"`
	AccountID         uuid.UUID         `json:"account_id" db:"account_id"`
	FileName          string            `json:"file_name" db:"file_name"`
	FileSizeBytes     int64             `json:"file_size_bytes" db:"file_size_bytes"`
	FileHashSHA256    string            `json:"file_hash_sha256" db:"file_hash_sha256"`
	TotalRows         int               `json:"total_rows" db:"total_rows"`
	ImportedRows      int               `json:"imported_rows" db:"imported_rows"`
	SkippedDuplicates int               `json:"skipped_duplicates" db:"skipped_duplicates"`
	Status            ImportBatchStatus `json:"status" db:"status"`
	ErrorMessage      *string           `json:"error_message,omitempty" db:"error_message"`
	Metadata          map[string]any    `json:"metadata" db:"metadata"`
	CreatedAt         time.Time         `json:"created_at" db:"created_at"`
}

// Domain errors for financial accounts and statement imports.
var (
	ErrFinancialAccountNotFound      = errors.New("akun keuangan tidak ditemukan")
	ErrFinancialAccountDuplicateName = errors.New("nama akun keuangan sudah digunakan")
	ErrFinancialAccountInactive      = errors.New("akun keuangan tidak aktif")
	ErrFinancialAccountInvalidType   = errors.New("jenis akun keuangan tidak valid")
	ErrFinancialAccountNameEmpty     = errors.New("nama akun keuangan wajib diisi")
	ErrFinancialAccountInstEmpty     = errors.New("kode institusi keuangan wajib diisi")
	ErrImportBatchNotFound           = errors.New("batch impor tidak ditemukan")
	ErrImportBatchAlreadyRolledBack  = errors.New("batch impor sudah dibatalkan sebelumnya")
	ErrImportBatchNotCompleted       = errors.New("hanya batch selesai yang dapat dibatalkan")
)

// FinancialAccountRepository contract for Postgres persistence.
type FinancialAccountRepository interface {
	Create(ctx context.Context, account *FinancialAccount) error
	GetByID(ctx context.Context, id, userID uuid.UUID) (*FinancialAccount, error)
	GetByIDForUpdate(ctx context.Context, id, userID uuid.UUID) (*FinancialAccount, error)
	ListByUser(ctx context.Context, userID uuid.UUID, onlyActive bool) ([]FinancialAccount, error)
	Update(ctx context.Context, account *FinancialAccount) error
	UpdateBalance(ctx context.Context, id, userID uuid.UUID, newBalance float64) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
}

// ImportBatchRepository contract for statement import batches.
type ImportBatchRepository interface {
	Create(ctx context.Context, batch *ImportBatch) error
	GetByID(ctx context.Context, id, userID uuid.UUID) (*ImportBatch, error)
	ListByUser(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID, limit, offset int) ([]ImportBatch, error)
	UpdateStatus(ctx context.Context, id, userID uuid.UUID, status ImportBatchStatus, errMessage *string) error
	GetLatestByFileHash(ctx context.Context, userID, accountID uuid.UUID, fileHash string) (*ImportBatch, error)
}
