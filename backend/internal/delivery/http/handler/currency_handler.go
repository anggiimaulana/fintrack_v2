package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/currency"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

// CurrencyHandler serves the dynamic currency catalog, rate history, and the
// Super Admin catalog mutations.
type CurrencyHandler struct {
	uc *usecase.CurrencyUsecase
}

func NewCurrencyHandler(uc *usecase.CurrencyUsecase) *CurrencyHandler {
	return &CurrencyHandler{uc: uc}
}

// currencyView augments catalog metadata with the rate applicable now, so the
// frontend can render and convert without a second round-trip.
type currencyView struct {
	currency.Meta
	RateToBase *float64   `json:"rate_to_base,omitempty"`
	RateAt     *time.Time `json:"rate_at,omitempty"`
}

func (h *CurrencyHandler) view(ctx *gin.Context, metas []currency.Meta) []currencyView {
	out := make([]currencyView, 0, len(metas))
	now := time.Now()
	for _, m := range metas {
		v := currencyView{Meta: m}
		if r, err := h.uc.RateAsOf(ctx.Request.Context(), m.Code, now); err == nil && r != nil {
			rate := r.RateToBase
			at := r.EffectiveAt
			v.RateToBase = &rate
			v.RateAt = &at
		}
		out = append(out, v)
	}
	return out
}

// List handles GET /currencies - active currencies for the signed-in user.
func (h *CurrencyHandler) List(c *gin.Context) {
	metas, err := h.uc.Catalog(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Gagal memuat daftar mata uang")
		return
	}
	response.OK(c, "Daftar mata uang berhasil dimuat", h.view(c, metas))
}

// ListAll handles GET /admin/currencies - full catalog (Super Admin).
func (h *CurrencyHandler) ListAll(c *gin.Context) {
	metas, err := h.uc.FullCatalog(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Gagal memuat katalog mata uang")
		return
	}
	response.OK(c, "Katalog mata uang berhasil dimuat", h.view(c, metas))
}

type upsertCurrencyRequest struct {
	Code          string  `json:"code" binding:"required"`
	Name          string  `json:"name" binding:"required"`
	Symbol        *string `json:"symbol"`
	Locale        *string `json:"locale"`
	DecimalDigits *int    `json:"decimal_digits"`
	IsActive      *bool   `json:"is_active"`
	SortOrder     *int    `json:"sort_order"`
	RateSource    *string `json:"rate_source"`
}

// Upsert handles POST /admin/currencies (Super Admin).
func (h *CurrencyHandler) Upsert(c *gin.Context) {
	var req upsertCurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Body tidak valid", err.Error())
		return
	}
	in := usecase.CurrencyInput{
		Code:          req.Code,
		Name:          req.Name,
		DecimalDigits: req.DecimalDigits,
		IsActive:      req.IsActive,
		SortOrder:     req.SortOrder,
	}
	// A blank optional string is treated as "not provided" (the field is NOT
	// NULL in the catalog, so it cannot be cleared).
	if req.Symbol != nil && *req.Symbol != "" {
		in.Symbol = req.Symbol
	}
	if req.Locale != nil && *req.Locale != "" {
		in.Locale = req.Locale
	}
	if req.RateSource != nil && *req.RateSource != "" {
		in.RateSource = req.RateSource
	}
	saved, err := h.uc.UpsertCurrency(c.Request.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrCurrencyActivation):
			response.Error(c, http.StatusUnprocessableEntity, "Isi kurs mata uang terlebih dahulu", "currency_no_rate")
		case errors.Is(err, currency.ErrUnknown):
			response.Error(c, http.StatusNotFound, "Mata uang tidak ditemukan", "currency_unknown")
		default:
			response.Error(c, http.StatusUnprocessableEntity, err.Error(), "currency_invalid")
		}
		return
	}
	response.OK(c, "Mata uang berhasil disimpan", saved)
}

type addRateRequest struct {
	CurrencyCode string  `json:"currency_code" binding:"required"`
	RateToBase   float64 `json:"rate_to_base" binding:"required"`
	EffectiveAt  string  `json:"effective_at"`
	Note         string  `json:"note"`
}

// ListRates handles GET /admin/currency-rates?code=JPY (Super Admin).
func (h *CurrencyHandler) ListRates(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.BadRequest(c, "Parameter code wajib diisi", "validation")
		return
	}
	rates, err := h.uc.Rates(c.Request.Context(), code, 200)
	if err != nil {
		response.InternalError(c, "Gagal memuat riwayat kurs")
		return
	}
	response.OK(c, "Riwayat kurs berhasil dimuat", rates)
}

// AddRate handles POST /admin/currency-rates (Super Admin). Append-only.
func (h *CurrencyHandler) AddRate(c *gin.Context) {
	var req addRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Body tidak valid", err.Error())
		return
	}
	rt := currency.Rate{CurrencyCode: req.CurrencyCode, RateToBase: req.RateToBase}
	if req.EffectiveAt != "" {
		t, err := parseTimeFlexible(req.EffectiveAt)
		if err != nil {
			response.BadRequest(c, "Format tanggal kurs tidak valid", "validation")
			return
		}
		rt.EffectiveAt = t
	}
	if req.Note != "" {
		rt.Note = &req.Note
	}
	var actorID *uuid.UUID
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uuid.UUID); ok {
			actorID = &id
		}
	}
	if err := h.uc.AddRate(c.Request.Context(), rt, actorID); err != nil {
		switch {
		case errors.Is(err, currency.ErrRateDuplicate):
			response.Error(c, http.StatusConflict, "Sudah ada kurs pada tanggal tersebut", "currency_rate_duplicate")
		case errors.Is(err, currency.ErrUnknown):
			response.Error(c, http.StatusNotFound, "Mata uang tidak ditemukan", "currency_unknown")
		default:
			response.Error(c, http.StatusUnprocessableEntity, err.Error(), "currency_rate_invalid")
		}
		return
	}
	response.OK(c, "Kurs berhasil ditambahkan", nil)
}

// parseTimeFlexible accepts RFC3339 or a plain date (YYYY-MM-DD).
func parseTimeFlexible(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}
