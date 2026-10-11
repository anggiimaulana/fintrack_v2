package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/internal/payment"
)

// PaymentHandler exposes payment endpoints (webhook + manual + admin review).
type PaymentHandler struct {
	processor *payment.WebhookProcessor
	manual    *payment.ManualService
	admin     *payment.AdminService
}

// NewPaymentHandler builds the handler.
func NewPaymentHandler(processor *payment.WebhookProcessor) *PaymentHandler {
	return &PaymentHandler{processor: processor}
}

// WithManual attaches the manual payment service.
func (h *PaymentHandler) WithManual(m *payment.ManualService) *PaymentHandler { h.manual = m; return h }

// WithAdmin attaches the admin review service.
func (h *PaymentHandler) WithAdmin(a *payment.AdminService) *PaymentHandler { h.admin = a; return h }

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
