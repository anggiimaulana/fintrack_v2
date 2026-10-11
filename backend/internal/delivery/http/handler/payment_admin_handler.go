package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/payment"
)

// ApproveManual approves a manual payment (super_admin only; route-guarded).
func (h *PaymentHandler) ApproveManual(c *gin.Context) {
	if h.admin == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "not_configured"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_id"})
		return
	}
	pay, err := h.admin.Approve(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "approve_failed", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Pembayaran disetujui", "data": pay})
}

// RejectManual rejects a manual payment with a required reason.
func (h *PaymentHandler) RejectManual(c *gin.Context) {
	if h.admin == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "not_configured"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "reason_required"})
		return
	}
	pay, err := h.admin.Reject(c.Request.Context(), id, body.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "reject_failed", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Pembayaran ditolak", "data": pay})
}

// ListPendingManual lists payments awaiting review.
func (h *PaymentHandler) ListPendingManual(c *gin.Context) {
	if h.admin == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "not_configured"})
		return
	}
	list, err := h.admin.ListPending(c.Request.Context(), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "list_failed"})
		return
	}
	if list == nil {
		list = []*payment.Payment{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
}
