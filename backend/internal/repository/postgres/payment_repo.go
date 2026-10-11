package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/payment"
)

// PaymentRepo implements payment.Repository over PostgreSQL.
type PaymentRepo struct {
	pool *pgxpool.Pool
}

// NewPaymentRepo builds the repository.
func NewPaymentRepo(pool *pgxpool.Pool) *PaymentRepo {
	return &PaymentRepo{pool: pool}
}

// ─── Invoices ─────────────────────────────────────────────────

const invoiceSelect = `SELECT id, invoice_number, user_id, subscription_id, subtotal_idr, tax_idr, total_idr,
    status, notes, issued_at, due_at, paid_at, voided_at, created_at, updated_at FROM invoices`

func (r *PaymentRepo) scanInvoice(row rowScanner) (*payment.Invoice, error) {
	inv := &payment.Invoice{}
	err := row.Scan(&inv.ID, &inv.InvoiceNumber, &inv.UserID, &inv.SubscriptionID, &inv.SubtotalIDR,
		&inv.TaxIDR, &inv.TotalIDR, &inv.Status, &inv.Notes, &inv.IssuedAt, &inv.DueAt, &inv.PaidAt,
		&inv.VoidedAt, &inv.CreatedAt, &inv.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, payment.ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *PaymentRepo) CreateInvoice(ctx context.Context, inv *payment.Invoice) error {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	err := q(ctx, r.pool).QueryRow(ctx,
		`INSERT INTO invoices (id, invoice_number, user_id, subscription_id, subtotal_idr, tax_idr, total_idr, status, notes, due_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING issued_at, created_at, updated_at`,
		inv.ID, inv.InvoiceNumber, inv.UserID, inv.SubscriptionID, inv.SubtotalIDR, inv.TaxIDR,
		inv.TotalIDR, inv.Status, inv.Notes, inv.DueAt).
		Scan(&inv.IssuedAt, &inv.CreatedAt, &inv.UpdatedAt)
	return err
}

func (r *PaymentRepo) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*payment.Invoice, error) {
	return r.scanInvoice(q(ctx, r.pool).QueryRow(ctx, invoiceSelect+` WHERE id = $1`, id))
}

func (r *PaymentRepo) GetInvoiceByNumber(ctx context.Context, number string) (*payment.Invoice, error) {
	return r.scanInvoice(q(ctx, r.pool).QueryRow(ctx, invoiceSelect+` WHERE invoice_number = $1`, number))
}

func (r *PaymentRepo) UpdateInvoiceStatus(ctx context.Context, id uuid.UUID, status payment.InvoiceStatus, paidAt, voidedAt *string) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE invoices SET status=$2,
            paid_at = COALESCE($3::timestamptz, paid_at),
            voided_at = COALESCE($4::timestamptz, voided_at),
            updated_at = now()
         WHERE id=$1`, id, status, paidAt, voidedAt)
	return err
}

func (r *PaymentRepo) MarkInvoicePaid(ctx context.Context, id uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE invoices SET status='paid', paid_at=now(), updated_at=now() WHERE id=$1`, id)
	return err
}

// NextInvoiceNumber atomically allocates the next number for a day:
// INV/YYYYMMDD/NNNNN. Safe across instances via row lock on invoice_counters.
func (r *PaymentRepo) NextInvoiceNumber(ctx context.Context, day string) (string, error) {
	d, err := time.Parse("20060102", day)
	if err != nil {
		return "", fmt.Errorf("invalid day %q: %w", day, err)
	}
	// Upsert-and-return the incremented counter atomically.
	var seq int64
	err = q(ctx, r.pool).QueryRow(ctx,
		`INSERT INTO invoice_counters (day, seq) VALUES ($1, 1)
         ON CONFLICT (day) DO UPDATE SET seq = invoice_counters.seq + 1
         RETURNING seq`, d).Scan(&seq)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("INV/%s/%05d", day, seq), nil
}

func (r *PaymentRepo) ListInvoicesDueBefore(ctx context.Context, beforeUnix int64) ([]*payment.Invoice, error) {
	before := time.Unix(beforeUnix, 0).UTC()
	rows, err := q(ctx, r.pool).Query(ctx,
		invoiceSelect+` WHERE status IN ('issued','past_due') AND due_at IS NOT NULL AND due_at < $1 ORDER BY due_at ASC LIMIT 500`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*payment.Invoice
	for rows.Next() {
		inv, err := r.scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// ─── Payments ─────────────────────────────────────────────────

const paymentSelect = `SELECT id, invoice_id, user_id, gateway, gateway_ref, amount_idr, status,
    payment_method, proof_url, idempotency_key, payload_snapshot, created_at, updated_at FROM payments`

func (r *PaymentRepo) scanPayment(row rowScanner) (*payment.Payment, error) {
	p := &payment.Payment{}
	err := row.Scan(&p.ID, &p.InvoiceID, &p.UserID, &p.Gateway, &p.GatewayRef, &p.AmountIDR, &p.Status,
		&p.PaymentMethod, &p.ProofURL, &p.IdempotencyKey, &p.PayloadSnapshot, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, payment.ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PaymentRepo) CreatePayment(ctx context.Context, p *payment.Payment) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.PayloadSnapshot == nil {
		p.PayloadSnapshot = map[string]any{}
	}
	err := q(ctx, r.pool).QueryRow(ctx,
		`INSERT INTO payments (id, invoice_id, user_id, gateway, gateway_ref, amount_idr, status, payment_method, proof_url, idempotency_key, payload_snapshot)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING created_at, updated_at`,
		p.ID, p.InvoiceID, p.UserID, p.Gateway, p.GatewayRef, p.AmountIDR, p.Status, p.PaymentMethod,
		p.ProofURL, p.IdempotencyKey, p.PayloadSnapshot).
		Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return payment.ErrDuplicatePayment
		}
		return err
	}
	return nil
}

func (r *PaymentRepo) GetPaymentByID(ctx context.Context, id uuid.UUID) (*payment.Payment, error) {
	return r.scanPayment(q(ctx, r.pool).QueryRow(ctx, paymentSelect+` WHERE id=$1`, id))
}

func (r *PaymentRepo) GetPaymentByGatewayRef(ctx context.Context, ref string) (*payment.Payment, error) {
	return r.scanPayment(q(ctx, r.pool).QueryRow(ctx, paymentSelect+` WHERE gateway_ref=$1`, ref))
}

// GetPaymentByGatewayRefForUpdate locks the payment row for the duration of the
// transaction (webhook idempotency).
func (r *PaymentRepo) GetPaymentByGatewayRefForUpdate(ctx context.Context, ref string) (*payment.Payment, error) {
	return r.scanPayment(q(ctx, r.pool).QueryRow(ctx, paymentSelect+` WHERE gateway_ref=$1 FOR UPDATE`, ref))
}

func (r *PaymentRepo) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status payment.PaymentStatus, method, proofURL *string, snapshot map[string]any) error {
	if snapshot == nil {
		snapshot = map[string]any{}
	}
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE payments SET status=$2,
            payment_method = COALESCE($3, payment_method),
            proof_url = COALESCE($4, proof_url),
            payload_snapshot = $5, updated_at=now()
         WHERE id=$1`, id, status, method, proofURL, snapshot)
	return err
}

func (r *PaymentRepo) SetPaymentProof(ctx context.Context, id uuid.UUID, proofURL string) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE payments SET proof_url=$2, status='pending_review', updated_at=now() WHERE id=$1`, id, proofURL)
	return err
}

func (r *PaymentRepo) ListPaymentsPendingReview(ctx context.Context, limit int) ([]*payment.Payment, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := q(ctx, r.pool).Query(ctx, paymentSelect+` WHERE status='pending_review' ORDER BY created_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*payment.Payment
	for rows.Next() {
		p, err := r.scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// compile-time assertion.
var _ payment.Repository = (*PaymentRepo)(nil)
