package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/pkg/response"
)

// ApproveManual approves a manual payment (super_admin only; route-guarded).
func (h *PaymentHandler) ApproveManual(c *gin.Context) {
	if h.admin == nil {
		response.Error(c, http.StatusServiceUnavailable, "Layanan verifikasi pembayaran belum dikonfigurasi", "not_configured")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID pembayaran tidak valid", "invalid_id")
		return
	}
	pay, err := h.admin.Approve(c.Request.Context(), id)
	if err != nil {
		writeManualReviewError(c, err)
		return
	}
	response.OK(c, "Pembayaran disetujui", pay)
}

// RejectManual rejects a manual payment with a required reason.
func (h *PaymentHandler) RejectManual(c *gin.Context) {
	if h.admin == nil {
		response.Error(c, http.StatusServiceUnavailable, "Layanan verifikasi pembayaran belum dikonfigurasi", "not_configured")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID pembayaran tidak valid", "invalid_id")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Reason == "" {
		response.BadRequest(c, "Alasan penolakan wajib diisi", "reason_required")
		return
	}
	pay, err := h.admin.Reject(c.Request.Context(), id, body.Reason)
	if err != nil {
		writeManualReviewError(c, err)
		return
	}
	response.OK(c, "Pembayaran ditolak", pay)
}

// ListPendingManual lists payments awaiting review.
func (h *PaymentHandler) ListPendingManual(c *gin.Context) {
	if h.admin == nil {
		response.Error(c, http.StatusServiceUnavailable, "Layanan verifikasi pembayaran belum dikonfigurasi", "not_configured")
		return
	}
	list, err := h.admin.ListPending(c.Request.Context(), 50)
	if err != nil {
		response.InternalError(c, "Gagal memuat antrean pembayaran")
		return
	}
	if list == nil {
		list = []*payment.Payment{}
	}
	response.OK(c, "Antrean verifikasi berhasil dimuat", list)
}

// writeManualReviewError maps manual-review domain errors onto the correct HTTP
// status: a missing payment is 404, an already-processed one is 409, and
// anything else is a genuine server error (500).
func writeManualReviewError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, payment.ErrPaymentNotFound):
		response.NotFound(c, "Pembayaran tidak ditemukan")
	case errors.Is(err, payment.ErrInvalidState):
		response.Conflict(c, "Pembayaran ini sudah diproses sebelumnya")
	default:
		response.InternalError(c, "Gagal memproses pembayaran")
	}
}
