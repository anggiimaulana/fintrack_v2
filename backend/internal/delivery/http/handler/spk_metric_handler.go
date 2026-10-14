package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

// SPKMetricHandler exposes the SPK metric *configuration* endpoints: the
// super-admin default model and the signed-in user's personal override.
type SPKMetricHandler struct {
	usecase *usecase.SPKMetricUsecase
}

func NewSPKMetricHandler(u *usecase.SPKMetricUsecase) *SPKMetricHandler {
	return &SPKMetricHandler{usecase: u}
}

// metricModelRequest is the wire shape accepted from both admin and user. It
// mirrors domain.SPKMetricModel so the JSON is symmetric on read and write.
type metricModelRequest struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Indicators  []domain.SPKMetricIndicator    `json:"indicators"`
	Thresholds  []domain.SPKStatusThreshold    `json:"status_thresholds"`
}

func (r metricModelRequest) model() domain.SPKMetricModel {
	return domain.SPKMetricModel{Indicators: r.Indicators, Thresholds: r.Thresholds}
}

// metricValidationError maps a domain validation error to an HTTP 400 with the
// specific reason, so the UI can point at the offending field.
func metricValidationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrSPKNoIndicators),
		errors.Is(err, domain.ErrSPKWeightSum),
		errors.Is(err, domain.ErrSPKDuplicateKey),
		errors.Is(err, domain.ErrSPKInvalidType),
		errors.Is(err, domain.ErrSPKInvalidWeight),
		errors.Is(err, domain.ErrSPKInvalidTarget),
		errors.Is(err, domain.ErrSPKMissingLabel),
		errors.Is(err, domain.ErrSPKNoThresholds),
		errors.Is(err, domain.ErrSPKThresholdOrder),
		errors.Is(err, domain.ErrSPKThresholdInvalid):
		response.BadRequest(c, err.Error(), "invalid_metric_model")
	default:
		response.InternalError(c, "Gagal menyimpan model SPK")
	}
}

// ─── User-facing: personal override ───────────────────────────

// GetMyModel handles GET /spk/metrics — the model that applies to the user,
// plus whether it is their own override.
func (h *SPKMetricHandler) GetMyModel(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	model, isOverride, err := h.usecase.GetUserModel(c.Request.Context(), userID.String())
	if err != nil {
		response.InternalError(c, "Gagal memuat model SPK")
		return
	}
	response.OK(c, "Model SPK berhasil dimuat", gin.H{
		"model":       model,
		"is_override": isOverride,
	})
}

// PutMyModel handles PUT /spk/metrics — save the user's personal override.
func (h *SPKMetricHandler) PutMyModel(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	var req metricModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	model, err := h.usecase.SetUserOverride(c.Request.Context(), userID.String(), req.model())
	if err != nil {
		metricValidationError(c, err)
		return
	}
	// The cached GET /spk/metrics (30s) would otherwise keep serving the old
	// model; bump this user's cache version so the change is visible at once.
	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.OK(c, "Model SPK Anda disimpan", gin.H{"model": model, "is_override": true})
}

// DeleteMyModel handles DELETE /spk/metrics — revert to the super-admin default.
func (h *SPKMetricHandler) DeleteMyModel(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.usecase.DeleteUserOverride(c.Request.Context(), userID.String()); err != nil {
		response.InternalError(c, "Gagal mengembalikan model SPK")
		return
	}
	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.OK(c, "Model SPK dikembalikan ke default", nil)
}

// ─── Super-admin: default model ───────────────────────────────

// GetDefault handles GET /admin/spk-metrics.
func (h *SPKMetricHandler) GetDefault(c *gin.Context) {
	model, err := h.usecase.GetDefault(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Gagal memuat model SPK default")
		return
	}
	response.OK(c, "Model SPK default berhasil dimuat", gin.H{"model": model})
}

// PutDefault handles PUT /admin/spk-metrics.
func (h *SPKMetricHandler) PutDefault(c *gin.Context) {
	var req metricModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	model, err := h.usecase.SetDefault(c.Request.Context(), req.Name, req.Description, req.model(), actorIDString(c))
	if err != nil {
		metricValidationError(c, err)
		return
	}
	// The default applies to every user without an override, so their cached
	// /spk/metrics responses must be dropped globally, not just for one user.
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Model SPK default disimpan", gin.H{"model": model})
}

// actorIDString returns the acting user's id as *string for the audit column.
func actorIDString(c *gin.Context) *string {
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uuid.UUID); ok {
			s := id.String()
			return &s
		}
	}
	return nil
}
