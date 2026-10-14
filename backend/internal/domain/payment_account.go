package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// PaymentAccountKind classifies a manual payment destination.
type PaymentAccountKind string

const (
	PaymentAccountBank    PaymentAccountKind = "bank"
	PaymentAccountEwallet PaymentAccountKind = "ewallet"
	PaymentAccountQRIS    PaymentAccountKind = "qris"
)

// Valid reports whether k is a known kind.
func (k PaymentAccountKind) Valid() bool {
	switch k {
	case PaymentAccountBank, PaymentAccountEwallet, PaymentAccountQRIS:
		return true
	default:
		return false
	}
}

// PaymentAccount is one manual payment destination (bank, e-wallet, or QRIS)
// shown to users at checkout.
type PaymentAccount struct {
	ID            uuid.UUID          `json:"id"`
	Kind          PaymentAccountKind `json:"kind"`
	Name          string             `json:"name"`
	AccountNumber string             `json:"account_number"`
	AccountHolder string             `json:"account_holder"`
	QRISURL       string             `json:"qris_url,omitempty"`
	SortOrder     int                `json:"sort_order"`
	IsActive      bool               `json:"is_active"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

// UpsertPaymentAccountRequest is the write shape for a payment account.
type UpsertPaymentAccountRequest struct {
	Kind          PaymentAccountKind `json:"kind"`
	Name          string             `json:"name"`
	AccountNumber string             `json:"account_number"`
	AccountHolder string             `json:"account_holder"`
	QRISURL       string             `json:"qris_url"`
	SortOrder     int                `json:"sort_order"`
	IsActive      *bool              `json:"is_active"`
}

// Validation errors, exported so the HTTP layer can answer 400 rather than 500.
var (
	ErrPaymentAccountInvalidKind = errors.New("jenis akun pembayaran tidak dikenal")
	ErrPaymentAccountNameEmpty   = errors.New("nama akun pembayaran wajib diisi")
	ErrPaymentAccountNumberEmpty = errors.New("nomor akun wajib diisi untuk bank/e-wallet")
	ErrPaymentAccountQRISURL     = errors.New("tautan QRIS wajib diisi untuk jenis QRIS")
	ErrPaymentAccountNotFound    = errors.New("akun pembayaran tidak ditemukan")
)

// Validate checks a write request before it is persisted. Pure so the admin
// handler and tests share exactly one rule set.
func (r UpsertPaymentAccountRequest) Validate() error {
	if !r.Kind.Valid() {
		return ErrPaymentAccountInvalidKind
	}
	if r.Name == "" {
		return ErrPaymentAccountNameEmpty
	}
	if r.Kind == PaymentAccountQRIS {
		// A QRIS entry is identified by its image, not an account number.
		if r.QRISURL == "" {
			return ErrPaymentAccountQRISURL
		}
		return nil
	}
	if r.AccountNumber == "" {
		return ErrPaymentAccountNumberEmpty
	}
	return nil
}

// PaymentAccountRepository persists manual payment destinations.
type PaymentAccountRepository interface {
	// ListActive returns active accounts in display order (for checkout).
	ListActive(ctx context.Context) ([]PaymentAccount, error)
	// ListAll returns every account, including inactive (for the admin UI).
	ListAll(ctx context.Context) ([]PaymentAccount, error)
	Create(ctx context.Context, req UpsertPaymentAccountRequest) (*PaymentAccount, error)
	Update(ctx context.Context, id uuid.UUID, req UpsertPaymentAccountRequest) (*PaymentAccount, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
