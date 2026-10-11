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
}

// NewCheckoutService builds the checkout service.
func NewCheckoutService(repo Repository, settings SettingsRepository, registry *Registry, invoices *InvoiceService, planGW PlanGatewayResolver) *CheckoutService {
	return &CheckoutService{repo: repo, settings: settings, registry: registry, invoices: invoices, planGW: planGW}
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
func (s *CheckoutService) Checkout(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error) {
	gw, err := s.registry.ResolveGateway(ctx, s.settings, req.PlanCode, s.planGW)
	if err != nil {
		return nil, err
	}

	// Invoice (records the plan code in notes for later activation).
	inv, err := s.invoices.Create(ctx, CreateInvoiceDTO{
		UserID:      req.UserID,
		SubtotalIDR: req.AmountIDR,
		Notes:       req.PlanCode,
	})
	if err != nil {
		return nil, err
	}

	idemKey := req.IdempotencyKey
	if idemKey == "" {
		idemKey = "checkout:" + inv.ID.String()
	}

	// Initiate the gateway transaction.
	txRes, err := gw.CreateTransaction(ctx, CreateTxRequest{
		InvoiceID:     inv.ID,
		OrderID:       inv.InvoiceNumber,
		AmountIDR:     inv.TotalIDR,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		Description:   req.PlanName,
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

func rawInstructions(raw map[string]any) map[string]any {
	if raw == nil {
		return nil
	}
	if ins, ok := raw["instructions"].(map[string]any); ok {
		return ins
	}
	return nil
}
