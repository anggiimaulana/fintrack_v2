package payment

import (
	"context"
	"fmt"
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
	// PlanCode names the plan this invoice buys. Stored in the dedicated
	// `plan_code` column (NOT `notes`) so activation never has to parse a note.
	PlanCode string
	// Notes carries free-form metadata (e.g. the renewal idempotency marker).
	Notes string
	// DueHours overrides the default due window (0 = settings/invoice_due_hours).
	DueHours int
}

// Create builds and persists an invoice with a deterministic number and an
// integer-Rupiah total (subtotal + tax).
func (s *InvoiceService) Create(ctx context.Context, in CreateInvoiceDTO) (*Invoice, error) {
	// Guard against allocating an invoice number when the user already has an
	// open invoice: prevents sequence gaps on redundant create attempts.
	if in.UserID != uuid.Nil && s.repo != nil {
		if open, err := s.repo.GetOpenInvoiceForUser(ctx, in.UserID); err == nil && open != nil {
			return nil, fmt.Errorf("%w: %s", ErrOpenInvoiceExists, open.InvoiceNumber)
		}
	}

	now := s.now().In(s.location)
	number, err := s.repo.NextInvoiceNumber(ctx, now.Format("20060102"))
	if err != nil {
		return nil, err
	}

	subtotal := in.SubtotalIDR
	if subtotal < 0 {
		subtotal = 0
	}

	dueHours := in.DueHours
	if dueHours <= 0 {
		// Default window, then overridden by settings. Expressed as hours +
		// minutes so an operator can set e.g. 1h 30m; resolved to minutes here.
		dueHours = 24
		if s.settings != nil {
			hours := int64(0)
			if v, err := s.settings.GetInt(ctx, "invoice_expiry_hours"); err == nil && v >= 0 {
				hours = v
			}
			minutes := int64(0)
			if v, err := s.settings.GetInt(ctx, "invoice_expiry_minutes"); err == nil && v >= 0 {
				minutes = v
			}
			// A window of zero (0h 0m) is not a usable invoice: fall back to the
			// legacy invoice_due_hours, then to 24h.
			if hours > 0 || minutes > 0 {
				dueAt := now.Add(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute)
				return s.persistInvoice(ctx, in, number, subtotal, now, dueAt)
			}
			if v, err := s.settings.GetInt(ctx, "invoice_due_hours"); err == nil && v > 0 {
				dueHours = int(v)
			}
		}
	}
	dueAt := now.Add(time.Duration(dueHours) * time.Hour)
	return s.persistInvoice(ctx, in, number, subtotal, now, dueAt)
}

// persistInvoice writes the invoice row with a pre-computed due date. Split out
// so the hours+minutes and legacy-hours paths share one insert.
func (s *InvoiceService) persistInvoice(ctx context.Context, in CreateInvoiceDTO, number string, subtotal int64, now, dueAt time.Time) (*Invoice, error) {
	taxIDR := int64(0)
	if s.tax != nil {
		taxIDR = s.tax.TaxIDR(ctx, subtotal)
	}

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
	if in.PlanCode != "" {
		planCode := in.PlanCode
		inv.PlanCode = &planCode
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
