package handler

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/internal/storage"
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
	// P0/P1 user self-service surface.
	userSvc *payment.UserService
	catalog *payment.CatalogService
	subSvc      *payment.SubscriptionService
	subRepo     billing.SubscriptionRepository
	proofs      *storage.ProofStore
	autoCharger *payment.AutoCharger
	payRepo     payment.Repository
	invoices    *payment.InvoiceService
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

// WithAutoCharge attaches the auto-charger and card token repository.
func (h *PaymentHandler) WithAutoCharge(ac *payment.AutoCharger, repo payment.Repository, inv *payment.InvoiceService) *PaymentHandler {
	h.autoCharger = ac
	h.payRepo = repo
	h.invoices = inv
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
		case errors.Is(err, payment.ErrInvalidSignature):
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid_signature"})
		case errors.Is(err, payment.ErrPaymentNotFound):
			// A verified webhook for an unknown order is acknowledged (200) so the
			// gateway stops retrying; nothing is credited.
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

	// Prorated upgrade: when the user is switching plans mid-cycle, credit the
	// unused portion of the current plan so they only pay the difference for the
	// remaining days. A same-plan renewal (or no active subscription) is charged
	// the full price.
	amount := plan.PriceMonthlyIDR
	if h.subRepo != nil {
		if cur, cerr := h.subRepo.GetByUserID(c.Request.Context(), userID); cerr == nil && cur != nil {
			var curPlan *billing.Plan
			if cur.PlanID != uuid.Nil {
				curPlan, _ = h.planRepo.GetPlanByID(c.Request.Context(), cur.PlanID)
			}
			if curPlan != nil && curPlan.ID != plan.ID {
				pr := billing.CalculateUpgradeProration(cur, curPlan, plan, time.Now().UTC())
				if pr.AmountDueIDR > 0 {
					amount = pr.AmountDueIDR
				}
			}
		}
	}

	res, err := h.checkout.Checkout(c.Request.Context(), payment.CheckoutRequest{
		UserID:        userID,
		PlanCode:      plan.Code,
		PlanName:      plan.Name,
		AmountIDR:     amount,
		CustomerName:  userProfile.FullName,
		CustomerEmail: userProfile.Email,
	})
	if err != nil {
		// A user may hold at most one unpaid invoice; surface that as a conflict
		// with the existing invoice number so the UI can point them at it.
		if errors.Is(err, payment.ErrOpenInvoiceExists) {
			response.Conflict(c, "Anda masih memiliki tagihan yang belum dibayar. Bayar atau batalkan tagihan tersebut terlebih dahulu.")
			return
		}
		response.InternalError(c, "Gagal membuat sesi pembayaran. Silakan coba beberapa saat lagi.")
		return
	}

	response.Created(c, "Sesi pembayaran berhasil dibuat", res)
}
