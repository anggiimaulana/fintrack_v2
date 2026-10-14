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

// PaymentAccountSource lists the active manual payment destinations. Declared
// as a narrow interface (implemented by the payment-account repository via a
// small adapter) so the payment package does not depend on the domain package.
type PaymentAccountSource interface {
	ListActive(ctx context.Context) ([]ManualAccount, error)
}

// ManualAccount is one active manual payment destination, as the payment
// package needs it (a copy of the domain type, to keep the packages decoupled).
type ManualAccount struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	AccountNumber string `json:"account_number"`
	AccountHolder string `json:"account_holder"`
	QRISURL       string `json:"qris_url,omitempty"`
}

// UserService exposes read operations scoped to a single user.
type UserService struct {
	repo     Repository
	settings SettingsRepository
	registry *Registry
	manual   *ManualService
	// accounts supplies every active payment destination for the checkout
	// instructions. Nil falls back to the single gateway-configured account.
	accounts PaymentAccountSource
}

// NewUserService builds the user service. manual may be nil (proof upload then
// reports not-configured instead of panicking).
func NewUserService(repo Repository, settings SettingsRepository, registry *Registry, manual *ManualService) *UserService {
	return &UserService{repo: repo, settings: settings, registry: registry, manual: manual}
}

// WithPaymentAccounts attaches the multi-account source used to enrich manual
// payment instructions with every destination the operator has configured.
func (s *UserService) WithPaymentAccounts(src PaymentAccountSource) *UserService {
	s.accounts = src
	return s
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

// CancelInvoice lets a user cancel their own unpaid invoice. It closes the
// invoice (voided, reason "cancelled_by_user") and expires any payment still
// pending against it, freeing the user to start a fresh checkout.
//
// Only issued/past_due invoices are cancellable: a paid invoice is terminal, and
// an already-closed one is a no-op error. The close is a conditional UPDATE, so
// a payment settling concurrently wins and the user gets ErrInvoiceNotCancellable
// rather than a cancelled-but-paid invoice.
func (s *UserService) CancelInvoice(ctx context.Context, userID, invoiceID uuid.UUID) (*Invoice, error) {
	inv, err := s.repo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	// Scope to the owner: never reveal or mutate another user's invoice.
	if inv.UserID != userID {
		return nil, ErrInvoiceNotFound
	}
	if !inv.Status.IsOpen() {
		return nil, fmt.Errorf("%w: status %s", ErrInvoiceNotCancellable, inv.Status)
	}

	changed, err := s.repo.CloseInvoice(ctx, inv.ID, InvoiceVoided, "cancelled_by_user")
	if err != nil {
		return nil, err
	}
	if !changed {
		// Lost the race: someone paid or expired it between our read and write.
		return nil, ErrInvoiceNotCancellable
	}
	if err := s.repo.ExpirePaymentsForInvoice(ctx, inv.ID); err != nil {
		return nil, err
	}
	// Re-read so the caller sees the closed state.
	return s.repo.GetInvoiceByID(ctx, inv.ID)
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

// PaymentInstructions returns the manual payment instructions for a payment:
// the amount, the order id, and *every* active destination the operator has
// configured (bank/e-wallet/QRIS), so the UI can show all of them without an
// extra round trip. Instructions are re-derived on each call, so a user can
// re-open the modal later without a fresh checkout.
func (s *UserService) PaymentInstructions(ctx context.Context, userID, paymentID uuid.UUID) (map[string]any, error) {
	pay, err := s.repo.GetPaymentForUser(ctx, userID, paymentID)
	if err != nil {
		return nil, err
	}
	if pay == nil {
		return nil, ErrPaymentNotFound
	}

	// Start from the stored snapshot (amount + order id), then overlay the live
	// account list so newly added destinations appear on old pending payments.
	ins := map[string]any{}
	if pay.PayloadSnapshot != nil {
		if snap, ok := pay.PayloadSnapshot["instructions"].(map[string]any); ok {
			for k, v := range snap {
				ins[k] = v
			}
		}
	}
	// A payment created before the snapshot carried instructions has neither
	// amount nor order id; derive them from the row itself.
	if _, ok := ins["amount_idr"]; !ok {
		ins["amount_idr"] = pay.AmountIDR
	}
	if _, ok := ins["order_id"]; !ok {
		ins["order_id"] = pay.InvoiceNumber
	}

	// Attach every active account (the modern, multi-account path).
	if s.accounts != nil {
		if accounts, aerr := s.accounts.ListActive(ctx); aerr == nil && len(accounts) > 0 {
			ins["accounts"] = accounts
			return ins, nil
		}
	}

	// Fallback: the legacy single account from the gateway config, so a
	// deployment with no rows in payment_accounts still shows something.
	if _, ok := ins["bank_name"]; !ok && s.registry != nil {
		if gw, gok := s.registry.Get("manual"); gok {
			res, gerr := gw.CreateTransaction(ctx, CreateTxRequest{
				InvoiceID: pay.InvoiceID,
				OrderID:   pay.InvoiceNumber,
				AmountIDR: pay.AmountIDR,
			})
			if gerr == nil && res.Raw != nil {
				if legacy, ok := res.Raw["instructions"].(map[string]any); ok {
					for k, v := range legacy {
						if _, exists := ins[k]; !exists {
							ins[k] = v
						}
					}
				}
			}
		}
	}
	return ins, nil
}
