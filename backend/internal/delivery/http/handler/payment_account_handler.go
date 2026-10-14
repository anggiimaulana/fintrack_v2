package handler

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/audit"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

// AuditWriter records administrative actions (satisfied by *audit.Service).
type AuditWriter interface {
	Log(ctx context.Context, e audit.Event)
}

// PaymentAccountHandler exposes manual payment destinations: the public list
// users pay to, and the super-admin CRUD.
type PaymentAccountHandler struct {
	usecase *usecase.PaymentAccountUsecase
	// audit records mutations. Nil disables auditing (used by tests that do not
	// exercise the audit trail).
	audit AuditWriter
}

func NewPaymentAccountHandler(u *usecase.PaymentAccountUsecase) *PaymentAccountHandler {
	return &PaymentAccountHandler{usecase: u}
}

// WithAudit attaches the audit writer so account changes are recorded
// (matching every other administrative mutation in this codebase).
func (h *PaymentAccountHandler) WithAudit(w AuditWriter) *PaymentAccountHandler {
	h.audit = w
	return h
}

// auditAccount records one account mutation. A nil writer is a no-op.
func (h *PaymentAccountHandler) auditAccount(c *gin.Context, action string, acc *domain.PaymentAccount) {
	if h.audit == nil {
		return
	}
	var actorID *uuid.UUID
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uuid.UUID); ok {
			actorID = &id
		}
	}
	entityID := ""
	var after map[string]any
	if acc != nil {
		entityID = acc.ID.String()
		after = map[string]any{
			"kind":           string(acc.Kind),
			"name":           acc.Name,
			"account_number": acc.AccountNumber,
			"account_holder": acc.AccountHolder,
			"is_active":      acc.IsActive,
		}
	}
	h.audit.Log(c.Request.Context(), audit.Event{
		ActorID:   actorID,
		Action:    action,
		Entity:    "payment_account",
		EntityID:  entityID,
		After:     after,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
}

// paymentAccountValidationError maps a domain validation error to a 400.
func paymentAccountValidationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrPaymentAccountInvalidKind),
		errors.Is(err, domain.ErrPaymentAccountNameEmpty),
		errors.Is(err, domain.ErrPaymentAccountNumberEmpty),
		errors.Is(err, domain.ErrPaymentAccountQRISURL):
		response.BadRequest(c, err.Error(), "invalid_payment_account")
	case errors.Is(err, domain.ErrPaymentAccountNotFound):
		response.NotFound(c, err.Error())
	default:
		response.InternalError(c, "Gagal menyimpan akun pembayaran")
	}
}

// ListActive handles GET /payment-accounts — the active destinations shown to a
// user at checkout (any authenticated user).
func (h *PaymentAccountHandler) ListActive(c *gin.Context) {
	accounts, err := h.usecase.ListActive(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Gagal memuat akun pembayaran")
		return
	}
	response.OK(c, "Akun pembayaran berhasil dimuat", accounts)
}

// ListAll handles GET /admin/payment-accounts — every account (super admin).
func (h *PaymentAccountHandler) ListAll(c *gin.Context) {
	accounts, err := h.usecase.ListAll(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Gagal memuat akun pembayaran")
		return
	}
	response.OK(c, "Akun pembayaran berhasil dimuat", accounts)
}

// Create handles POST /admin/payment-accounts.
func (h *PaymentAccountHandler) Create(c *gin.Context) {
	var req domain.UpsertPaymentAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	acc, err := h.usecase.Create(c.Request.Context(), req)
	if err != nil {
		paymentAccountValidationError(c, err)
		return
	}
	h.auditAccount(c, "create", acc)
	response.Created(c, "Akun pembayaran dibuat", acc)
}

// Update handles PUT /admin/payment-accounts/:id.
func (h *PaymentAccountHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID akun tidak valid", "invalid_id")
		return
	}
	var req domain.UpsertPaymentAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	acc, err := h.usecase.Update(c.Request.Context(), id, req)
	if err != nil {
		paymentAccountValidationError(c, err)
		return
	}
	h.auditAccount(c, "update", acc)
	response.OK(c, "Akun pembayaran diperbarui", acc)
}

// Delete handles DELETE /admin/payment-accounts/:id.
func (h *PaymentAccountHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID akun tidak valid", "invalid_id")
		return
	}
	if err := h.usecase.Delete(c.Request.Context(), id); err != nil {
		paymentAccountValidationError(c, err)
		return
	}
	h.auditAccount(c, "delete", nil)
	response.OK(c, "Akun pembayaran dihapus", nil)
}
