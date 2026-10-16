// Package payment implements the billing/payment layer: gateway abstraction
// (Midtrans Snap + manual transfer), deterministic invoice numbering, webhook
// verification, proration, tax, and dunning.
//
// Money is always integer Rupiah (int64). No card data is ever handled here
// (PCI-DSS SAQ-A): the gateway returns a redirect/token and the customer pays on
// the gateway's hosted page.
package payment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ─── Enums ────────────────────────────────────────────────────

// InvoiceStatus is the lifecycle state of an invoice.
type InvoiceStatus string

const (
	InvoiceDraft   InvoiceStatus = "draft"
	InvoiceIssued  InvoiceStatus = "issued"
	InvoicePaid    InvoiceStatus = "paid"
	InvoicePastDue InvoiceStatus = "past_due"
	InvoiceVoided  InvoiceStatus = "voided"
	// InvoiceExpired is set by the expiry worker when an unpaid invoice passes
	// its due date (or by a gateway "expire" webhook).
	InvoiceExpired InvoiceStatus = "expired"
)

// IsOpen reports whether an invoice is unpaid and not yet closed. The database
// enforces "at most one open invoice per user" via a partial unique index, so
// this must match that index's predicate exactly.
func (s InvoiceStatus) IsOpen() bool {
	return s == InvoiceIssued || s == InvoicePastDue
}

// PaymentStatus is the lifecycle state of a payment attempt.
type PaymentStatus string

const (
	PaymentPending       PaymentStatus = "pending"
	PaymentPendingReview PaymentStatus = "pending_review"
	PaymentSettled       PaymentStatus = "settled"
	PaymentFailed        PaymentStatus = "failed"
	PaymentExpired       PaymentStatus = "expired"
	PaymentRefunded      PaymentStatus = "refunded"
)

// IsFinal reports whether a payment status is terminal (idempotency boundary).
func (s PaymentStatus) IsFinal() bool {
	switch s {
	case PaymentSettled, PaymentFailed, PaymentExpired, PaymentRefunded:
		return true
	}
	return false
}

// ─── Entities ─────────────────────────────────────────────────

// Invoice is a bill issued to a user.
type Invoice struct {
	ID             uuid.UUID     `json:"id"`
	InvoiceNumber  string        `json:"invoice_number"`
	UserID         uuid.UUID     `json:"user_id"`
	SubscriptionID *uuid.UUID    `json:"subscription_id,omitempty"`
	SubtotalIDR    int64         `json:"subtotal_idr"`
	TaxIDR         int64         `json:"tax_idr"`
	TotalIDR       int64         `json:"total_idr"`
	Status         InvoiceStatus `json:"status"`
	// PlanCode names the plan this invoice buys. It is the single source of
	// truth for activation (never `notes`, which carries free-form metadata
	// such as the renewal idempotency marker).
	PlanCode *string `json:"plan_code,omitempty"`
	Notes    *string `json:"notes,omitempty"`
	IssuedAt       time.Time     `json:"issued_at"`
	DueAt          *time.Time    `json:"due_at,omitempty"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	VoidedAt       *time.Time    `json:"voided_at,omitempty"`
	// ClosedReason explains why a non-paid invoice closed: "cancelled_by_user",
	// "expired", "superseded", or a gateway reason.
	ClosedReason *string    `json:"closed_reason,omitempty"`
	ExpiredAt    *time.Time `json:"expired_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Payment is a payment attempt against an invoice.
type Payment struct {
	ID              uuid.UUID      `json:"id"`
	InvoiceID       uuid.UUID      `json:"invoice_id"`
	UserID          uuid.UUID      `json:"user_id"`
	Gateway         string         `json:"gateway"`
	GatewayRef      *string        `json:"gateway_ref,omitempty"`
	AmountIDR       int64          `json:"amount_idr"`
	Status          PaymentStatus  `json:"status"`
	PaymentMethod   *string        `json:"payment_method,omitempty"`
	ProofURL        *string        `json:"proof_url,omitempty"`
	IdempotencyKey  string         `json:"idempotency_key"`
	PayloadSnapshot map[string]any `json:"payload_snapshot,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// ─── Card Token Entity ─────────────────────────────────────────

// CardToken represents a tokenized payment card (e.g. Midtrans saved_token_id)
// for recurring/one-click billing. Never stores PAN or CVV (PCI-DSS SAQ-A).
type CardToken struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Gateway    string     `json:"gateway"`
	TokenID    string     `json:"-"` // sensitive gateway token, omitted from JSON
	MaskedCard string     `json:"masked_card"`
	CardType   string     `json:"card_type,omitempty"`
	Bank       string     `json:"bank,omitempty"`
	IsDefault  bool       `json:"is_default"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ─── DTOs ─────────────────────────────────────────────────────

// CreateTxRequest initiates a payment at a gateway.
type CreateTxRequest struct {
	InvoiceID     uuid.UUID
	UserID        uuid.UUID
	OrderID       string
	AmountIDR     int64
	CustomerName  string
	CustomerEmail string
	Description   string
	SaveCard      bool
	// ExpiryMinutes bounds how long the payment is valid (0 = gateway default).
	ExpiryMinutes int
}

// CreateTxResponse is the gateway's response to a checkout creation.
type CreateTxResponse struct {
	// GatewayRef is the external order id (used for webhook correlation).
	GatewayRef string
	// SnapToken/RedirectURL are for hosted payment (no card data on our side).
	SnapToken   string
	RedirectURL string
	Status      PaymentStatus
	Raw         map[string]any
}

// WebhookResult is the verified outcome of a gateway callback.
type WebhookResult struct {
	OrderID               string
	GatewayRef            string
	Status                PaymentStatus
	AmountIDR             int64
	PaymentMethod         string
	SavedTokenID          string
	SavedTokenIDExpiredAt string
	MaskedCard            string
	CardType              string
	Bank                  string
	Raw                   map[string]any
}

// StatusResult is the outcome of a server-side status pull.
type StatusResult struct {
	OrderID               string
	Status                PaymentStatus
	AmountIDR             int64
	SavedTokenID          string
	SavedTokenIDExpiredAt string
	MaskedCard            string
	CardType              string
	Bank                  string
	Raw                   map[string]any
}

// ChargeSavedCardRequest initiates a charge using a tokenized card.
type ChargeSavedCardRequest struct {
	UserID    uuid.UUID
	OrderID   string
	AmountIDR int64
	TokenID   string
}

// ChargeCardResponse is the gateway's response to a card charge.
type ChargeCardResponse struct {
	GatewayRef    string
	Status        PaymentStatus
	PaymentMethod string
	MaskedCard    string
	Bank          string
	Raw           map[string]any
}
// ─── Gateway interface ────────────────────────────────────────

// Gateway abstracts a payment provider. Implementations must never store or
// process card data (PCI-DSS SAQ-A).
type Gateway interface {
	// Name returns the gateway identifier ("midtrans" | "manual").
	Name() string
	// CreateTransaction starts a checkout and returns a reference/token.
	CreateTransaction(ctx context.Context, req CreateTxRequest) (*CreateTxResponse, error)
	// VerifyWebhook validates the callback (signature) and returns its result.
	VerifyWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error)
	// CheckStatus pulls the authoritative status from the gateway (anti-spoof).
	CheckStatus(ctx context.Context, gatewayRef string) (*StatusResult, error)
	// CancelTransaction cancels/voids a pending transaction.
	CancelTransaction(ctx context.Context, gatewayRef string) error
}

// CardCharger is an optional interface implemented by gateways supporting
// server-side recurring/auto-charges via tokenized cards (e.g. Midtrans Core API).
type CardCharger interface {
	ChargeSavedCard(ctx context.Context, req ChargeSavedCardRequest) (*ChargeCardResponse, error)
}
// ─── Errors ───────────────────────────────────────────────────

var (
	ErrGatewayNotConfigured = errors.New("payment gateway not configured")
	ErrInvalidSignature     = errors.New("invalid webhook signature")
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrInvoiceNotFound      = errors.New("invoice not found")
	ErrInvalidState         = errors.New("invalid payment state transition")
	ErrDuplicatePayment     = errors.New("duplicate payment (idempotency key)")
	ErrInvalidProof         = errors.New("invalid proof file")
	ErrProofTooLarge        = errors.New("proof file too large")
	// ErrOpenInvoiceExists means the user already has an unpaid invoice; they
	// must pay or cancel it before creating another.
	ErrOpenInvoiceExists = errors.New("an unpaid invoice already exists")
	// ErrInvoiceNotCancellable means the invoice is not in a state a user may
	// cancel (already paid, expired, or previously cancelled).
	ErrInvoiceNotCancellable = errors.New("invoice cannot be cancelled")
	ErrCardTokenNotFound     = errors.New("saved card token not found")
	ErrCardChargeFailed      = errors.New("card charge failed")
)
