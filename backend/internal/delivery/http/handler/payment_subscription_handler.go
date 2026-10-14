package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/pkg/response"
)

// MySubscription handles GET /api/v1/subscriptions/me. It returns the user's
// subscription (plan joined) plus any scheduled plan change, so the UI can
// show "aktif sampai ...", "downgrade ke Free pada ...", etc.
func (h *PaymentHandler) MySubscription(c *gin.Context) {
	if h.subRepo == nil {
		response.OK(c, "Belum ada langganan", gin.H{"subscription": nil, "scheduled_plan_code": ""})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	sub, err := h.subRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, billing.ErrSubscriptionNotFound) {
			response.OK(c, "Belum ada langganan", gin.H{"subscription": nil, "scheduled_plan_code": ""})
			return
		}
		response.InternalError(c, "Gagal memuat langganan")
		return
	}
	if sub == nil {
		response.OK(c, "Belum ada langganan", gin.H{"subscription": nil, "scheduled_plan_code": ""})
		return
	}

	scheduled := ""
	if sub.Metadata != nil {
		if v, ok := sub.Metadata["scheduled_plan_code"].(string); ok {
			scheduled = v
		}
	}
	// Attach the plan for display.
	if sub.PlanID != uuid.Nil && h.planRepo != nil {
		if plan, perr := h.planRepo.GetPlanByID(c.Request.Context(), sub.PlanID); perr == nil && plan != nil {
			sub.Plan = plan
		}
	}
	response.OK(c, "Langganan berhasil dimuat", gin.H{
		"subscription":        sub,
		"scheduled_plan_code": scheduled,
	})
}

// ScheduleDowngrade handles POST /api/v1/subscriptions/me/downgrade.
// The change takes effect at the end of the current cycle (soft-limit).
func (h *PaymentHandler) ScheduleDowngrade(c *gin.Context) {
	if h.subSvc == nil {
		response.InternalError(c, "Layanan langganan belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	var req struct {
		PlanCode string `json:"plan_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Kode paket wajib diisi", "validation")
		return
	}
	if err := h.subSvc.ScheduleDowngrade(c.Request.Context(), userID, req.PlanCode); err != nil {
		switch {
		case errors.Is(err, billing.ErrSubscriptionNotFound):
			response.NotFound(c, "Langganan tidak ditemukan")
		case errors.Is(err, billing.ErrPlanNotFound):
			response.NotFound(c, "Paket tidak ditemukan")
		case errors.Is(err, billing.ErrInvalidStateTransition):
			response.BadRequest(c, "Paket tujuan tidak valid", "invalid_plan")
		default:
			response.InternalError(c, "Gagal menjadwalkan penurunan paket")
		}
		return
	}
	response.OK(c, "Penurunan paket dijadwalkan pada akhir periode", nil)
}

// CancelDowngrade handles DELETE /api/v1/subscriptions/me/downgrade.
func (h *PaymentHandler) CancelDowngrade(c *gin.Context) {
	if h.subSvc == nil {
		response.InternalError(c, "Layanan langganan belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.subSvc.CancelDowngrade(c.Request.Context(), userID); err != nil {
		if errors.Is(err, billing.ErrSubscriptionNotFound) {
			response.NotFound(c, "Langganan tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal membatalkan penurunan paket")
		return
	}
	response.OK(c, "Penurunan paket dibatalkan", nil)
}

// CancelSubscription handles POST /api/v1/subscriptions/me/cancel. Access
// continues until the end of the current period.
func (h *PaymentHandler) CancelSubscription(c *gin.Context) {
	if h.subSvc == nil {
		response.InternalError(c, "Layanan langganan belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.subSvc.CancelSubscription(c.Request.Context(), userID); err != nil {
		if errors.Is(err, billing.ErrSubscriptionNotFound) {
			response.NotFound(c, "Langganan tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal membatalkan langganan")
		return
	}
	response.OK(c, "Langganan akan berakhir pada akhir periode", nil)
}
