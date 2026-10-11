package payment

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// InvoiceService creates deterministic invoices and computes due dates.
type InvoiceService struct {
	repo     Repository
	settings SettingsRepository
	tax      *TaxCalculator
	now      func() time.Time
	// location is the business timezone used for the invoice day + due date.
	location *time.Location
}

// NewInvoiceService builds the invoice service (Asia/Jakarta by default).
func NewInvoiceService(repo Repository, settings SettingsRepository, tax *TaxCalculator) *InvoiceService {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return &InvoiceService{repo: repo, settings: settings, tax: tax, now: func() time.Time { return time.Now().UTC() }, location: loc}
}

// CreateInvoiceDTO is the input to CreateInvoice.
type CreateInvoiceDTO struct {
	UserID         uuid.UUID
	SubscriptionID *uuid.UUID
	SubtotalIDR    int64
	Notes          string
	// DueHours overrides the default due window (0 = settings/invoice_due_hours).
	DueHours int
}

// Create builds and persists an invoice with a deterministic number and an
// integer-Rupiah total (subtotal + tax).
func (s *InvoiceService) Create(ctx context.Context, in CreateInvoiceDTO) (*Invoice, error) {
	now := s.now().In(s.location)
	number, err := s.repo.NextInvoiceNumber(ctx, now.Format("20060102"))
	if err != nil {
		return nil, err
	}

	subtotal := in.SubtotalIDR
	if subtotal < 0 {
		subtotal = 0
	}
	taxIDR := int64(0)
	if s.tax != nil {
		taxIDR = s.tax.TaxIDR(ctx, subtotal)
	}

	dueHours := in.DueHours
	if dueHours <= 0 {
		dueHours = 24
		if s.settings != nil {
			if v, err := s.settings.GetInt(ctx, "invoice_due_hours"); err == nil && v > 0 {
				dueHours = int(v)
			}
		}
	}
	dueAt := now.Add(time.Duration(dueHours) * time.Hour)

	inv := &Invoice{
		UserID:         in.UserID,
		SubscriptionID: in.SubscriptionID,
		InvoiceNumber:  number,
		SubtotalIDR:    subtotal,
		TaxIDR:         taxIDR,
		TotalIDR:       subtotal + taxIDR,
		Status:         InvoiceIssued,
		DueAt:          &dueAt,
	}
	if in.Notes != "" {
		notes := in.Notes
		inv.Notes = &notes
	}
	if err := s.repo.CreateInvoice(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// Get returns an invoice by id.
func (s *InvoiceService) Get(ctx context.Context, id uuid.UUID) (*Invoice, error) {
	return s.repo.GetInvoiceByID(ctx, id)
}

// Void marks an invoice as voided.
func (s *InvoiceService) Void(ctx context.Context, id uuid.UUID) error {
	now := s.now().UTC().Format(time.RFC3339)
	return s.repo.UpdateInvoiceStatus(ctx, id, InvoiceVoided, nil, &now)
}
