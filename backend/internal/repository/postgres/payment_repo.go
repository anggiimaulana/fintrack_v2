package postgres

import (
	"context"
	"encoding/json"
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
    status, notes, issued_at, due_at, paid_at, voided_at, closed_reason, expired_at, created_at, updated_at FROM invoices`

func (r *PaymentRepo) scanInvoice(row rowScanner) (*payment.Invoice, error) {
	inv := &payment.Invoice{}
	err := row.Scan(&inv.ID, &inv.InvoiceNumber, &inv.UserID, &inv.SubscriptionID, &inv.SubtotalIDR,
		&inv.TaxIDR, &inv.TotalIDR, &inv.Status, &inv.Notes, &inv.IssuedAt, &inv.DueAt, &inv.PaidAt,
		&inv.VoidedAt, &inv.ClosedReason, &inv.ExpiredAt, &inv.CreatedAt, &inv.UpdatedAt)
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

// GetOpenInvoiceForUser returns the user's single open (unpaid, unclosed)
// invoice, or (nil, nil) when they have none. Used to block a second checkout.
func (r *PaymentRepo) GetOpenInvoiceForUser(ctx context.Context, userID uuid.UUID) (*payment.Invoice, error) {
	inv, err := r.scanInvoice(q(ctx, r.pool).QueryRow(ctx,
		invoiceSelect+` WHERE user_id = $1 AND status IN ('issued','past_due') LIMIT 1`, userID))
	if errors.Is(err, payment.ErrInvoiceNotFound) {
		return nil, nil
	}
	return inv, err
}

// GetOpenInvoiceForUserForUpdate is the locking variant used inside a checkout
// transaction: it serialises concurrent checkouts for the same user so exactly
// one invoice is ever open (the partial unique index is the final backstop).
func (r *PaymentRepo) GetOpenInvoiceForUserForUpdate(ctx context.Context, userID uuid.UUID) (*payment.Invoice, error) {
	inv, err := r.scanInvoice(q(ctx, r.pool).QueryRow(ctx,
		invoiceSelect+` WHERE user_id = $1 AND status IN ('issued','past_due') FOR UPDATE LIMIT 1`, userID))
	if errors.Is(err, payment.ErrInvoiceNotFound) {
		return nil, nil
	}
	return inv, err
}

// CloseInvoice transitions an open invoice to a closed state (voided/expired)
// with a reason, and expires its non-final payments in the same statement pair.
// Returns true when a row actually changed, so callers can detect a lost race.
//
// $2 is cast explicitly: it is used both as the new status (varchar) and inside
// a CASE comparison, and Postgres cannot otherwise deduce a single type for it.
func (r *PaymentRepo) CloseInvoice(ctx context.Context, id uuid.UUID, status payment.InvoiceStatus, reason string) (bool, error) {
	tag, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE invoices
		 SET status = $2::text,
		     closed_reason = $3,
		     voided_at = CASE WHEN $2::text = 'voided' THEN now() ELSE voided_at END,
		     expired_at = CASE WHEN $2::text = 'expired' THEN now() ELSE expired_at END,
		     updated_at = now()
		 WHERE id = $1 AND status IN ('issued','past_due')`, id, string(status), reason)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ExpirePaymentsForInvoice marks a closed invoice's still-open payments as
// expired, so the user's history is consistent (no payment lingers "pending"
// against an invoice that can never be paid).
func (r *PaymentRepo) ExpirePaymentsForInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE payments SET status = 'expired', updated_at = now()
		 WHERE invoice_id = $1 AND status IN ('pending','pending_review')`, invoiceID)
	return err
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

// GetPaymentByIDForUpdate locks the payment row for the duration of the
// transaction (manual review idempotency).
func (r *PaymentRepo) GetPaymentByIDForUpdate(ctx context.Context, id uuid.UUID) (*payment.Payment, error) {
	return r.scanPayment(q(ctx, r.pool).QueryRow(ctx, paymentSelect+` WHERE id=$1 FOR UPDATE`, id))
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

// ─── User-scoped history ──────────────────────────────────────

// userPaymentSelect joins payments with their invoice (and the invoice's plan,
// resolved from notes) for the self-service history view.
const userPaymentSelect = `SELECT p.id, p.invoice_id, p.user_id, p.gateway, p.gateway_ref,
    p.amount_idr, p.status, p.payment_method, p.proof_url, p.idempotency_key,
    p.payload_snapshot, p.created_at, p.updated_at,
    COALESCE(i.invoice_number,''), COALESCE(i.status::text,''),
    COALESCE(pl.code,''), COALESCE(pl.name,''),
    COALESCE(i.subtotal_idr,0), COALESCE(i.tax_idr,0), COALESCE(i.total_idr,0),
    i.due_at
    FROM payments p
    LEFT JOIN invoices i ON i.id = p.invoice_id
    LEFT JOIN plans pl ON pl.code = i.notes`

func scanUserPayment(row rowScanner) (*payment.UserPayment, error) {
	var (
		up            payment.UserPayment
		payload       []byte
		invoiceStatus string
		dueAt         *time.Time
	)
	if err := row.Scan(
		&up.ID, &up.InvoiceID, &up.UserID, &up.Gateway, &up.GatewayRef,
		&up.AmountIDR, &up.Status, &up.PaymentMethod, &up.ProofURL, &up.IdempotencyKey,
		&payload, &up.CreatedAt, &up.UpdatedAt,
		&up.InvoiceNumber, &invoiceStatus,
		&up.PlanCode, &up.PlanName,
		&up.SubtotalIDR, &up.TaxIDR, &up.TotalIDR,
		&dueAt,
	); err != nil {
		return nil, err
	}
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &up.PayloadSnapshot)
	}
	up.InvoiceStatus = payment.InvoiceStatus(invoiceStatus)
	if dueAt != nil {
		s := dueAt.Format(time.RFC3339)
		up.InvoiceDueAt = &s
	}
	return &up, nil
}

// ListPaymentsByUser returns the user's payment attempts, newest first.
func (r *PaymentRepo) ListPaymentsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*payment.UserPayment, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := q(ctx, r.pool).Query(ctx,
		userPaymentSelect+` WHERE p.user_id = $1 ORDER BY p.created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*payment.UserPayment
	for rows.Next() {
		up, err := scanUserPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, up)
	}
	return out, rows.Err()
}

// GetPaymentForUser returns one payment (with invoice details) scoped to its
// owner. Returns (nil, nil) when the payment does not exist or belongs to
// someone else, so callers can answer 404 without leaking existence.
func (r *PaymentRepo) GetPaymentForUser(ctx context.Context, userID, paymentID uuid.UUID) (*payment.UserPayment, error) {
	up, err := scanUserPayment(q(ctx, r.pool).QueryRow(ctx,
		userPaymentSelect+` WHERE p.id = $1 AND p.user_id = $2`, paymentID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return up, nil
}

// ListInvoicesByUser returns the user's invoices, newest first.
func (r *PaymentRepo) ListInvoicesByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*payment.Invoice, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := q(ctx, r.pool).Query(ctx,
		invoiceSelect+` WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
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

// compile-time assertion.
var _ payment.Repository = (*PaymentRepo)(nil)
