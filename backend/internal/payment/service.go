package payment

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/pagination"
)

// Repository persists invoices and payments.
type Repository interface {
	// Invoices
	CreateInvoice(ctx context.Context, inv *Invoice) error
	GetInvoiceByID(ctx context.Context, id uuid.UUID) (*Invoice, error)
	GetInvoiceByNumber(ctx context.Context, number string) (*Invoice, error)
	// GetOpenInvoiceForUser returns the user's single open (unpaid) invoice, or
	// (nil, nil) when none. GetOpenInvoiceForUserForUpdate is the locking variant
	// used inside a checkout transaction to serialise concurrent checkouts.
	GetOpenInvoiceForUser(ctx context.Context, userID uuid.UUID) (*Invoice, error)
	GetOpenInvoiceForUserForUpdate(ctx context.Context, userID uuid.UUID) (*Invoice, error)
	// CloseInvoice transitions an open invoice to voided/expired with a reason,
	// reporting whether a row actually changed (false = lost a race).
	CloseInvoice(ctx context.Context, id uuid.UUID, status InvoiceStatus, reason string) (bool, error)
	// ExpirePaymentsForInvoice expires an invoice's still-open payments.
	ExpirePaymentsForInvoice(ctx context.Context, invoiceID uuid.UUID) error
	UpdateInvoiceStatus(ctx context.Context, id uuid.UUID, status InvoiceStatus, paidAt, voidedAt *string) error
	MarkInvoicePaid(ctx context.Context, id uuid.UUID) error
	NextInvoiceNumber(ctx context.Context, day string) (string, error)

	// Payments
	CreatePayment(ctx context.Context, p *Payment) error
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	// GetPaymentByIDForUpdate locks the payment row for the duration of the
	// transaction (manual review idempotency: one admin wins, the rest see the
	// committed state and fail with ErrInvalidState).
	GetPaymentByIDForUpdate(ctx context.Context, id uuid.UUID) (*Payment, error)
	GetPaymentByGatewayRef(ctx context.Context, ref string) (*Payment, error)
	GetPaymentByGatewayRefForUpdate(ctx context.Context, ref string) (*Payment, error)
	UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status PaymentStatus, method, proofURL *string, snapshot map[string]any) error
	SetPaymentProof(ctx context.Context, id uuid.UUID, proofURL string) error
	ListPaymentsPendingReview(ctx context.Context, limit int) ([]*Payment, error)
	ListInvoicesDueBefore(ctx context.Context, beforeUnix int64) ([]*Invoice, error)
	// InvoiceExistsByMarker reports whether an invoice already carries the given
	// idempotency marker in its notes (renewal-per-cycle dedupe).
	InvoiceExistsByMarker(ctx context.Context, marker string) (bool, error)

	// Renewal billing: subscriptions whose cycle just ended and that should be
	// invoiced for the next one.
	ListRenewalCandidates(ctx context.Context, now time.Time) ([]RenewalCandidate, error)

	// Dunning: still-entitled subscriptions whose paid period ends before the
	// given Unix time. Ended/expired/canceled subscriptions are excluded.
	ListSubscriptionsEndingBefore(ctx context.Context, beforeUnix int64) ([]SubscriptionExpiry, error)

	// User-scoped reads (self-service history). Payments are joined with their
	// invoice so the UI can render one row per attempt without extra queries.
	ListPaymentsByUser(ctx context.Context, userID uuid.UUID, q *pagination.Query) ([]*UserPayment, int, error)
	// GetPaymentForUser returns one payment with invoice details, scoped to the
	// owner so a user can never read another user's payment.
	GetPaymentForUser(ctx context.Context, userID, paymentID uuid.UUID) (*UserPayment, error)
	ListInvoicesByUser(ctx context.Context, userID uuid.UUID, q *pagination.Query) ([]*Invoice, int, error)

	// Card tokens (for recurring and 1-click tokenized payments)
	SaveCardToken(ctx context.Context, token *CardToken) error
	GetDefaultCardToken(ctx context.Context, userID uuid.UUID) (*CardToken, error)
	ListCardTokensByUser(ctx context.Context, userID uuid.UUID) ([]*CardToken, error)
	GetCardTokenByID(ctx context.Context, id uuid.UUID) (*CardToken, error)
	SetDefaultCardToken(ctx context.Context, userID, tokenID uuid.UUID) error
	DeleteCardToken(ctx context.Context, userID, tokenID uuid.UUID) error
}

// UserPayment is a payment attempt joined with its invoice, for the user-facing
// billing history. Money fields stay integer Rupiah (int64).
type UserPayment struct {
	Payment
	InvoiceNumber string        `json:"invoice_number"`
	InvoiceStatus InvoiceStatus `json:"invoice_status"`
	PlanCode      string        `json:"plan_code,omitempty"`
	PlanName      string        `json:"plan_name,omitempty"`
	SubtotalIDR   int64         `json:"subtotal_idr"`
	TaxIDR        int64         `json:"tax_idr"`
	TotalIDR      int64         `json:"total_idr"`
	InvoiceDueAt  *string       `json:"invoice_due_at,omitempty"`
}

// SettingsRepository reads/writes the global settings store.
type SettingsRepository interface {
	GetString(ctx context.Context, key string) (string, error)
	GetInt(ctx context.Context, key string) (int64, error)
	Set(ctx context.Context, key string, value any) error
}

// SubscriptionActivator applies a successful payment to the subscription
// (activate or renew). Implemented by bridging to the billing module.
type SubscriptionActivator interface {
	// ActivateForUser ensures the user has an active subscription on the plan
	// referenced by the invoice, renewing the current period.
	ActivateForUser(ctx context.Context, userID uuid.UUID, planCode string) error
}

// Service coordinates invoice creation, gateway checkout, and webhook
// processing with idempotency.
type Service struct {
	repo     Repository
	settings SettingsRepository
	registry *Registry
}

// NewService builds the payment service.
func NewService(repo Repository, settings SettingsRepository, registry *Registry) *Service {
	return &Service{repo: repo, settings: settings, registry: registry}
}
