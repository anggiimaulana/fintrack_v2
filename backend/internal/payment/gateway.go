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
)

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
	Notes          *string       `json:"notes,omitempty"`
	IssuedAt       time.Time     `json:"issued_at"`
	DueAt          *time.Time    `json:"due_at,omitempty"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	VoidedAt       *time.Time    `json:"voided_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
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

// ─── DTOs ─────────────────────────────────────────────────────

// CreateTxRequest initiates a payment at a gateway.
type CreateTxRequest struct {
	InvoiceID     uuid.UUID
	OrderID       string
	AmountIDR     int64
	CustomerName  string
	CustomerEmail string
	Description   string
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
	OrderID       string
	GatewayRef    string
	Status        PaymentStatus
	AmountIDR     int64
	PaymentMethod string
	Raw           map[string]any
}

// StatusResult is the outcome of a server-side status pull.
type StatusResult struct {
	OrderID   string
	Status    PaymentStatus
	AmountIDR int64
	Raw       map[string]any
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
)
