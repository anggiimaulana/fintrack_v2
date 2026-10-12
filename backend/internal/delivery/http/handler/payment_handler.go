package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/pkg/response"
)

// PaymentHandler exposes payment endpoints (webhook + manual + admin review).
type PaymentHandler struct {
	processor *payment.WebhookProcessor
	manual    *payment.ManualService
	admin     *payment.AdminService
	checkout  *payment.CheckoutService
	planRepo  billing.PlanRepository
	userRepo  domain.ProfileRepository
}
// NewPaymentHandler builds the handler.
func NewPaymentHandler(processor *payment.WebhookProcessor) *PaymentHandler {
	return &PaymentHandler{processor: processor}
}

// WithManual attaches the manual payment service.
func (h *PaymentHandler) WithManual(m *payment.ManualService) *PaymentHandler { h.manual = m; return h }

// WithAdmin attaches the admin review service.
func (h *PaymentHandler) WithAdmin(a *payment.AdminService) *PaymentHandler { h.admin = a; return h }

// WithCheckout attaches the checkout service and required repositories.
func (h *PaymentHandler) WithCheckout(c *payment.CheckoutService, plans billing.PlanRepository, users domain.ProfileRepository) *PaymentHandler {
	h.checkout = c
	h.planRepo = plans
	h.userRepo = users
	return h
}

// MidtransWebhook handles POST /payments/webhook/midtrans (public, no auth).
// The raw body is capped at 1 MB; the gateway verifies the signature.
func (h *PaymentHandler) MidtransWebhook(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_body", "message": "Body tidak valid atau terlalu besar."})
		return
	}

	headers := map[string]string{"content-type": c.GetHeader("Content-Type")}
	res, err := h.processor.Process(c.Request.Context(), "midtrans", payload, headers)
	if err != nil {
		switch {
		case err == payment.ErrInvalidSignature:
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid_signature"})
		case err == payment.ErrPaymentNotFound:
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "ignored"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "processing_failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"status":            string(res.Status),
		"already_processed": res.AlreadyProcessed,
	})
}

// Checkout handles POST /api/v1/payments/checkout (authenticated user).
func (h *PaymentHandler) Checkout(c *gin.Context) {
	if h.checkout == nil {
		response.InternalError(c, "Layanan checkout belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		PlanCode string `json:"plan_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Kode paket wajib diisi", err.Error())
		return
	}

	plan, err := h.planRepo.GetPlanByCode(c.Request.Context(), req.PlanCode)
	if err != nil || plan == nil || !plan.IsActive {
		response.NotFound(c, "Paket langganan tidak ditemukan atau tidak aktif")
		return
	}

	userProfile, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil || userProfile == nil {
		response.InternalError(c, "Gagal memuat profil pengguna")
		return
	}

	res, err := h.checkout.Checkout(c.Request.Context(), payment.CheckoutRequest{
		UserID:        userID,
		PlanCode:      plan.Code,
		PlanName:      plan.Name,
		AmountIDR:     plan.PriceMonthlyIDR,
		CustomerName:  userProfile.FullName,
		CustomerEmail: userProfile.Email,
	})
	if err != nil {
		response.InternalError(c, "Gagal membuat sesi pembayaran. Silakan coba beberapa saat lagi.")
		return
	}

	response.Created(c, "Sesi pembayaran berhasil dibuat", res)
}
