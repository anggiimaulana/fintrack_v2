package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/pkg/response"
)

// ─── Payments: review, settlement & refund (P06-012) ──────────

type paymentRow struct {
	ID              string  `json:"id"`
	InvoiceID       string  `json:"invoice_id"`
	InvoiceNumber   *string `json:"invoice_number,omitempty"`
	UserID          string  `json:"user_id"`
	UserEmail       string  `json:"user_email"`
	Gateway         string  `json:"gateway"`
	GatewayRef      *string `json:"gateway_ref,omitempty"`
	AmountIDR       int64   `json:"amount_idr"`
	Status          string  `json:"status"`
	Method          *string `json:"payment_method,omitempty"`
	ProofURL        *string `json:"proof_url,omitempty"`
	ReviewReason    *string `json:"review_reason,omitempty"`
	ReviewedBy      *string `json:"reviewed_by,omitempty"`
	ReviewedAt      *string `json:"reviewed_at,omitempty"`
	RefundAmountIDR int64   `json:"refund_amount_idr"`
	RefundReason    *string `json:"refund_reason,omitempty"`
	RefundRef       *string `json:"refund_ref,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

// ListPendingPayments handles GET /admin/payments/review.
func (h *Handler) ListPendingPayments(c *gin.Context) {
	ctx := c.Request.Context()
	page, perPage := paginationParams(c)
	rows, err := h.svc.pool.Query(ctx, `
		SELECT p.id, p.invoice_id, i.invoice_number, p.user_id, u.email, p.gateway, p.gateway_ref,
		       p.amount_idr, p.status, p.payment_method, p.proof_url, p.review_reason,
		       p.reviewed_by, p.reviewed_at, p.refund_amount_idr, p.refund_reason, p.refund_ref,
		       p.created_at
		FROM payments p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN invoices i ON i.id = p.invoice_id
		WHERE p.status = 'pending_review'
		ORDER BY p.created_at ASC
		LIMIT $1 OFFSET $2`, perPage, (page-1)*perPage)
	if err != nil {
		response.InternalError(c, "Gagal memuat antrean pembayaran")
		return
	}
	defer rows.Close()

	out, err := scanPayments(rows)
	if err != nil {
		response.InternalError(c, "Gagal memuat antrean pembayaran")
		return
	}

	var total int64
	_ = h.svc.pool.QueryRow(ctx, `SELECT COUNT(*) FROM payments WHERE status = 'pending_review'`).Scan(&total)
	response.Paginated(c, "Antrean verifikasi berhasil dimuat", out, int(total), page, perPage)
}

// ListPayments handles GET /admin/payments (history + filters).
func (h *Handler) ListPayments(c *gin.Context) {
	ctx := c.Request.Context()
	page, perPage := paginationParams(c)
	status := c.Query("status")
	userID := c.Query("user_id")
	gateway := c.Query("gateway")
	search := strings.TrimSpace(c.Query("q"))

	var userArg any
	if id, err := uuid.Parse(userID); err == nil {
		userArg = id
	}

	const where = `
		WHERE ($1::text = '' OR p.status = $1)
		  AND ($2::uuid IS NULL OR p.user_id = $2)
		  AND ($3::text = '' OR p.gateway = $3)
		  AND ($4::text = '' OR u.email ILIKE '%'||$4||'%' OR i.invoice_number ILIKE '%'||$4||'%')`

	var total int64
	if err := h.svc.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payments p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN invoices i ON i.id = p.invoice_id`+where,
		status, userArg, gateway, search).Scan(&total); err != nil {
		response.InternalError(c, "Gagal memuat pembayaran")
		return
	}

	orderExpr := orderBy(c, map[string]string{
		"created_at": "p.created_at",
		"amount":     "p.amount_idr",
		"status":     "p.status",
		"user":       "u.email",
	}, "p.created_at DESC")

	rows, err := h.svc.pool.Query(ctx, `
		SELECT p.id, p.invoice_id, i.invoice_number, p.user_id, u.email, p.gateway, p.gateway_ref,
		       p.amount_idr, p.status, p.payment_method, p.proof_url, p.review_reason,
		       p.reviewed_by, p.reviewed_at, p.refund_amount_idr, p.refund_reason, p.refund_ref,
		       p.created_at
		FROM payments p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN invoices i ON i.id = p.invoice_id`+where+`
		ORDER BY `+orderExpr+` LIMIT $5 OFFSET $6`,
		status, userArg, gateway, search, perPage, (page-1)*perPage)
	if err != nil {
		response.InternalError(c, "Gagal memuat pembayaran")
		return
	}
	defer rows.Close()

	out, err := scanPayments(rows)
	if err != nil {
		response.InternalError(c, "Gagal memuat pembayaran")
		return
	}
	response.Paginated(c, "Daftar pembayaran berhasil dimuat", out, int(total), page, perPage)
}

// ApprovePayment handles POST /admin/payments/:id/approve.
func (h *Handler) ApprovePayment(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	ctx := c.Request.Context()

	pay, err := h.svc.manual.Approve(ctx, id)
	if err != nil {
		writePaymentError(c, err)
		return
	}
	h.markReviewed(ctx, c, id, "approve", "")
	h.audit(c, "approve", "payment", id.String(), nil,
		map[string]any{"status": pay.Status, "amount_idr": pay.AmountIDR})
	response.OK(c, "Pembayaran disetujui", gin.H{"id": id, "status": pay.Status})
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

// RejectPayment handles POST /admin/payments/:id/reject.
func (h *Handler) RejectPayment(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req rejectRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Reason == "" {
		response.BadRequest(c, "alasan penolakan wajib diisi", "validation")
		return
	}
	ctx := c.Request.Context()

	pay, err := h.svc.manual.Reject(ctx, id, req.Reason)
	if err != nil {
		writePaymentError(c, err)
		return
	}
	h.markReviewed(ctx, c, id, "reject", req.Reason)
	h.audit(c, "reject", "payment", id.String(), nil,
		map[string]any{"status": pay.Status, "reason": req.Reason})
	response.OK(c, "Pembayaran ditolak", gin.H{"id": id, "status": pay.Status})
}

type refundRequest struct {
	AmountIDR int64  `json:"amount_idr"`
	Reason    string `json:"reason"`
	Reference string `json:"reference"`
}

// RefundPayment handles POST /admin/payments/:id/refund.
//
// Partial refunds leave the payment settled; a refund that covers the full
// amount moves it to `refunded`. The original payment row is never deleted.
func (h *Handler) RefundPayment(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req refundRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Reason == "" {
		response.BadRequest(c, "alasan refund wajib diisi", "validation")
		return
	}
	ctx := c.Request.Context()

	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		response.InternalError(c, "Gagal memproses refund")
		return
	}
	defer tx.Rollback(ctx)

	var amount, alreadyRefunded int64
	var status string
	var userID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT amount_idr, refund_amount_idr, status, user_id
		FROM payments WHERE id = $1 FOR UPDATE`, id).Scan(&amount, &alreadyRefunded, &status, &userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFound(c, "Pembayaran tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal memproses refund")
		return
	}
	if status != string(payment.PaymentSettled) && status != string(payment.PaymentRefunded) {
		response.Error(c, 409, "Hanya pembayaran settled yang dapat direfund", "INVALID_PAYMENT_STATE")
		return
	}

	remaining := amount - alreadyRefunded
	if req.AmountIDR <= 0 {
		req.AmountIDR = remaining // default: refund the rest (full refund)
	}
	if req.AmountIDR > remaining {
		response.BadRequest(c, "nominal refund melebihi sisa yang dapat direfund", "validation")
		return
	}

	newRefunded := alreadyRefunded + req.AmountIDR
	newStatus := string(payment.PaymentSettled)
	if newRefunded >= amount {
		newStatus = string(payment.PaymentRefunded)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE payments
		SET refund_amount_idr = $2, refund_reason = $3, refund_ref = $4,
		    refunded_at = now(), status = $5
		WHERE id = $1`,
		id, newRefunded, req.Reason, nullableStr(req.Reference), newStatus); err != nil {
		response.InternalError(c, "Gagal memproses refund")
		return
	}
	if err := insertPaymentEvent(ctx, tx, id, h.svc.actorIDOrNil(c), "refund", req.Reason, req.AmountIDR,
		map[string]any{"reference": req.Reference, "refunded_total": newRefunded}); err != nil {
		response.InternalError(c, "Gagal mencatat event refund")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalError(c, "Gagal memproses refund")
		return
	}

	h.audit(c, "refund", "payment", id.String(),
		map[string]any{"refund_amount_idr": alreadyRefunded},
		map[string]any{"refund_amount_idr": newRefunded, "status": newStatus, "reason": req.Reason})
	response.OK(c, "Refund tercatat", gin.H{
		"id": id, "refund_amount_idr": newRefunded, "status": newStatus,
	})
}

// ─── helpers ──────────────────────────────────────────────────

// markReviewed stamps the reviewer on the payment and appends an event.
func (h *Handler) markReviewed(ctx contextT, c *gin.Context, id uuid.UUID, event, reason string) {
	actor := h.svc.actorIDOrNil(c)
	_, _ = h.svc.pool.Exec(ctx, `
		UPDATE payments SET reviewed_by = $2, reviewed_at = now(), review_reason = NULLIF($3,'')
		WHERE id = $1`, id, actor, reason)
	_ = insertPaymentEvent(ctx, h.svc.pool, id, actor, event, reason, 0, nil)
}

// execer is satisfied by both *pgxpool.Pool and pgx.Tx.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func insertPaymentEvent(ctx contextT, q execer, id uuid.UUID, actor *uuid.UUID, event, reason string, amount int64, meta map[string]any) error {
	_, err := q.Exec(ctx, `
		INSERT INTO payment_events (payment_id, actor_id, event, reason, amount_idr, metadata)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,$6)`,
		id, actor, event, reason, amount, jsonOrEmpty(meta))
	return err
}

// scanPayments converts a payments query into DTO rows.
func scanPayments(rows pgx.Rows) ([]paymentRow, error) {
	out := []paymentRow{}
	for rows.Next() {
		var p paymentRow
		var created time.Time
		var reviewedAt *time.Time
		if err := rows.Scan(&p.ID, &p.InvoiceID, &p.InvoiceNumber, &p.UserID, &p.UserEmail,
			&p.Gateway, &p.GatewayRef, &p.AmountIDR, &p.Status, &p.Method, &p.ProofURL,
			&p.ReviewReason, &p.ReviewedBy, &reviewedAt, &p.RefundAmountIDR, &p.RefundReason,
			&p.RefundRef, &created); err != nil {
			return nil, err
		}
		p.CreatedAt = created.Format(time.RFC3339)
		if reviewedAt != nil {
			s := reviewedAt.Format(time.RFC3339)
			p.ReviewedAt = &s
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// writePaymentError maps payment-domain errors onto HTTP responses.
func writePaymentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, payment.ErrPaymentNotFound):
		response.NotFound(c, "Pembayaran tidak ditemukan")
	case errors.Is(err, payment.ErrInvalidState):
		response.Error(c, 409, "Transaksi ini sudah diproses oleh admin lain.", "INVALID_PAYMENT_STATE")
	default:
		response.InternalError(c, "Gagal memproses pembayaran")
	}
}
