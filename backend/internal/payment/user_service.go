package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ─── User self-service (P0/P1) ────────────────────────────────
//
// These methods back the user-facing billing pages: plan catalog, payment
// history, invoice list, and the manual proof upload that closes the
// checkout → upload → admin-approve loop.

// UserService exposes read operations scoped to a single user.
type UserService struct {
	repo     Repository
	settings SettingsRepository
	registry *Registry
	manual   *ManualService
}

// NewUserService builds the user service. manual may be nil (proof upload then
// reports not-configured instead of panicking).
func NewUserService(repo Repository, settings SettingsRepository, registry *Registry, manual *ManualService) *UserService {
	return &UserService{repo: repo, settings: settings, registry: registry, manual: manual}
}

// ListPayments returns the user's payment attempts (newest first).
func (s *UserService) ListPayments(ctx context.Context, userID uuid.UUID, limit int) ([]*UserPayment, error) {
	return s.repo.ListPaymentsByUser(ctx, userID, limit)
}

// GetPayment returns one payment scoped to its owner (nil when not found).
func (s *UserService) GetPayment(ctx context.Context, userID, paymentID uuid.UUID) (*UserPayment, error) {
	return s.repo.GetPaymentForUser(ctx, userID, paymentID)
}

// GetPaymentForAdmin returns one payment with invoice details regardless of
// owner (used by the super-admin proof viewer).
func (s *UserService) GetPaymentForAdmin(ctx context.Context, paymentID uuid.UUID) (*UserPayment, error) {
	pay, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if pay == nil {
		return nil, nil
	}
	up := &UserPayment{Payment: *pay}
	if inv, ierr := s.repo.GetInvoiceByID(ctx, pay.InvoiceID); ierr == nil && inv != nil {
		up.InvoiceNumber = inv.InvoiceNumber
		up.InvoiceStatus = inv.Status
		up.SubtotalIDR = inv.SubtotalIDR
		up.TaxIDR = inv.TaxIDR
		up.TotalIDR = inv.TotalIDR
		if inv.Notes != nil {
			up.PlanCode = *inv.Notes
		}
	}
	return up, nil
}

// ListInvoices returns the user's invoices (newest first).
func (s *UserService) ListInvoices(ctx context.Context, userID uuid.UUID, limit int) ([]*Invoice, error) {
	return s.repo.ListInvoicesByUser(ctx, userID, limit)
}

// UploadProof validates and stores a proof for the user's own manual payment,
// moving it to pending_review so an admin can approve it. Returns
// ErrPaymentNotFound when the payment is missing or owned by someone else.
func (s *UserService) UploadProof(ctx context.Context, userID, paymentID uuid.UUID, content []byte, filename string) (*Payment, error) {
	if s.manual == nil {
		return nil, fmt.Errorf("%w: manual payments not configured", ErrGatewayNotConfigured)
	}
	return s.manual.UploadProof(ctx, paymentID, userID, content, filename)
}

// PaymentInstructions returns the bank-transfer instructions for a manual
// payment (re-derived from the stored snapshot, so a user can re-open the modal
// later without a fresh checkout).
func (s *UserService) PaymentInstructions(ctx context.Context, userID, paymentID uuid.UUID) (map[string]any, error) {
	pay, err := s.repo.GetPaymentForUser(ctx, userID, paymentID)
	if err != nil {
		return nil, err
	}
	if pay == nil {
		return nil, ErrPaymentNotFound
	}
	if pay.PayloadSnapshot != nil {
		if ins, ok := pay.PayloadSnapshot["instructions"].(map[string]any); ok {
			return ins, nil
		}
	}
	// Fall back to the live gateway config for payments created before the
	// snapshot carried instructions.
	if s.registry != nil {
		if gw, ok := s.registry.Get("manual"); ok {
			res, gerr := gw.CreateTransaction(ctx, CreateTxRequest{
				InvoiceID: pay.InvoiceID,
				OrderID:   pay.InvoiceNumber,
				AmountIDR: pay.AmountIDR,
			})
			if gerr == nil && res.Raw != nil {
				if ins, ok := res.Raw["instructions"].(map[string]any); ok {
					return ins, nil
				}
			}
		}
	}
	return map[string]any{}, nil
}
