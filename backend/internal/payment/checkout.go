package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CheckoutService orchestrates a checkout: resolve gateway → create invoice →
// create gateway transaction → persist payment. It locks the chosen gateway
// onto the payment row so in-flight payments survive a global gateway switch.
type CheckoutService struct {
	repo     Repository
	settings SettingsRepository
	registry *Registry
	invoices *InvoiceService
	planGW   PlanGatewayResolver
	// tx, when set, runs the open-invoice guard + invoice creation in one
	// transaction so two concurrent checkouts cannot both succeed. The partial
	// unique index on invoices is the final backstop.
	tx TxRunner
}

// NewCheckoutService builds the checkout service.
func NewCheckoutService(repo Repository, settings SettingsRepository, registry *Registry, invoices *InvoiceService, planGW PlanGatewayResolver) *CheckoutService {
	return &CheckoutService{repo: repo, settings: settings, registry: registry, invoices: invoices, planGW: planGW}
}

// WithTx installs the transaction runner used to make checkout race-safe.
func (s *CheckoutService) WithTx(tx TxRunner) *CheckoutService {
	s.tx = tx
	return s
}

// CheckoutRequest starts a payment for a plan.
type CheckoutRequest struct {
	UserID        uuid.UUID
	PlanCode      string
	PlanName      string
	AmountIDR     int64
	CustomerName  string
	CustomerEmail string
	// IdempotencyKey dedupes repeated checkout submissions.
	IdempotencyKey string
}

// CheckoutResult is returned to the client.
type CheckoutResult struct {
	Invoice      *Invoice       `json:"invoice"`
	Payment      *Payment       `json:"payment"`
	Gateway      string         `json:"gateway"`
	SnapToken    string         `json:"snap_token,omitempty"`
	RedirectURL  string         `json:"redirect_url,omitempty"`
	Instructions map[string]any `json:"instructions,omitempty"`
}

// Checkout creates the invoice and payment and initiates the gateway
// transaction.
//
// A user may have at most one open (unpaid) invoice at a time: if one exists,
// ErrOpenInvoiceExists is returned and the caller is expected to surface it so
// the user can pay or cancel the existing invoice instead. The guard runs in a
// transaction with a per-user advisory lock, and the DB's partial unique index
// is the final backstop, so concurrent clicks can never create two invoices.
func (s *CheckoutService) Checkout(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error) {
	gw, err := s.registry.ResolveGateway(ctx, s.settings, req.PlanCode, s.planGW)
	if err != nil {
		return nil, err
	}

	// Create the invoice under the guard.
	var inv *Invoice
	create := func(txCtx context.Context) error {
		// Serialise per user, then check for an existing open invoice.
		if err := s.lockUser(txCtx, req.UserID.String()); err != nil {
			return err
		}
		existing, err := s.repo.GetOpenInvoiceForUserForUpdate(txCtx, req.UserID)
		if err != nil {
			return err
		}
		if existing != nil {
			return fmt.Errorf("%w: %s", ErrOpenInvoiceExists, existing.InvoiceNumber)
		}
		created, err := s.invoices.Create(txCtx, CreateInvoiceDTO{
			UserID:      req.UserID,
			SubtotalIDR: req.AmountIDR,
			PlanCode:    req.PlanCode,
		})
		if err != nil {
			return err
		}
		inv = created
		return nil
	}

	if s.tx != nil {
		if err := s.tx.RunInTx(ctx, create); err != nil {
			return nil, err
		}
	} else {
		if err := create(ctx); err != nil {
			return nil, err
		}
	}

	idemKey := req.IdempotencyKey
	if idemKey == "" {
		idemKey = "checkout:" + inv.ID.String()
	}

	// Initiate the gateway transaction.
	txRes, err := gw.CreateTransaction(ctx, CreateTxRequest{
		InvoiceID:     inv.ID,
		UserID:        req.UserID,
		OrderID:       inv.InvoiceNumber,
		AmountIDR:     inv.TotalIDR,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		Description:   req.PlanName,
		SaveCard:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("gateway create: %w", err)
	}

	ref := txRes.GatewayRef
	pay := &Payment{
		InvoiceID:       inv.ID,
		UserID:          req.UserID,
		Gateway:         gw.Name(), // locked at checkout
		GatewayRef:      &ref,
		AmountIDR:       inv.TotalIDR,
		Status:          txRes.Status,
		IdempotencyKey:  idemKey,
		PayloadSnapshot: txRes.Raw,
	}
	if pay.Status == "" {
		pay.Status = PaymentPending
	}
	if err := s.repo.CreatePayment(ctx, pay); err != nil {
		return nil, err
	}

	return &CheckoutResult{
		Invoice:      inv,
		Payment:      pay,
		Gateway:      gw.Name(),
		SnapToken:    txRes.SnapToken,
		RedirectURL:  txRes.RedirectURL,
		Instructions: rawInstructions(txRes.Raw),
	}, nil
}

// lockUser takes a per-user advisory lock inside the checkout transaction, so
// two concurrent checkouts for the same user serialise instead of both reading
// "no open invoice" and both inserting.
func (s *CheckoutService) lockUser(ctx context.Context, userID string) error {
	if locker, ok := s.tx.(interface {
		AdvisoryXactLock(ctx context.Context, key string) error
	}); ok {
		return locker.AdvisoryXactLock(ctx, "checkout:"+userID)
	}
	return nil
}

func rawInstructions(raw map[string]any) map[string]any {
	if raw == nil {
		return nil
	}
	if ins, ok := raw["instructions"].(map[string]any); ok {
		return ins
	}
	return nil
}
