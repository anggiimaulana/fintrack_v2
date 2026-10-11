package payment

import (
	"context"

	"github.com/google/uuid"
)

// Repository persists invoices and payments.
type Repository interface {
	// Invoices
	CreateInvoice(ctx context.Context, inv *Invoice) error
	GetInvoiceByID(ctx context.Context, id uuid.UUID) (*Invoice, error)
	GetInvoiceByNumber(ctx context.Context, number string) (*Invoice, error)
	UpdateInvoiceStatus(ctx context.Context, id uuid.UUID, status InvoiceStatus, paidAt, voidedAt *string) error
	MarkInvoicePaid(ctx context.Context, id uuid.UUID) error
	NextInvoiceNumber(ctx context.Context, day string) (string, error)

	// Payments
	CreatePayment(ctx context.Context, p *Payment) error
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	GetPaymentByGatewayRef(ctx context.Context, ref string) (*Payment, error)
	GetPaymentByGatewayRefForUpdate(ctx context.Context, ref string) (*Payment, error)
	UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status PaymentStatus, method, proofURL *string, snapshot map[string]any) error
	SetPaymentProof(ctx context.Context, id uuid.UUID, proofURL string) error
	ListPaymentsPendingReview(ctx context.Context, limit int) ([]*Payment, error)
	ListInvoicesDueBefore(ctx context.Context, beforeUnix int64) ([]*Invoice, error)
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
