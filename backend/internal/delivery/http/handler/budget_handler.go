package handler

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

// BudgetHandler exposes per-user budgets and their actual-vs-budget status.
type BudgetHandler struct {
	usecase *usecase.BudgetUsecase
}

func NewBudgetHandler(u *usecase.BudgetUsecase) *BudgetHandler {
	return &BudgetHandler{usecase: u}
}

// List handles GET /budgets.
func (h *BudgetHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	budgets, err := h.usecase.List(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Gagal memuat anggaran")
		return
	}
	response.OK(c, "Anggaran berhasil dimuat", budgets)
}

// Status handles GET /budgets/status?start_date&end_date — each budget with its
// actual spend for the period.
func (h *BudgetHandler) Status(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	start, end, ok := parsePeriodQuery(c)
	if !ok {
		response.BadRequest(c, "start_date dan end_date wajib diisi (YYYY-MM-DD)", "validation")
		return
	}

	statuses, err := h.usecase.Status(c.Request.Context(), userID, start, end)
	if err != nil {
		response.InternalError(c, "Gagal memuat status anggaran")
		return
	}
	response.OK(c, "Status anggaran berhasil dimuat", statuses)
}

// Upsert handles POST /budgets (create or replace a category budget).
func (h *BudgetHandler) Upsert(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	var req domain.UpsertBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	budget, err := h.usecase.Upsert(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Gagal menyimpan anggaran")
		return
	}
	// GET /budgets and /budgets/status are cached for 30s; drop this user's
	// entries so the new budget shows immediately.
	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.OK(c, "Anggaran disimpan", budget)
}

// Delete handles DELETE /budgets/:id.
func (h *BudgetHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID anggaran tidak valid", "invalid_id")
		return
	}
	if err := h.usecase.Delete(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Anggaran tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal menghapus anggaran")
		return
	}
	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.OK(c, "Anggaran dihapus", nil)
}

// parsePeriodQuery reads start_date/end_date, falling back to the current month
// when they are absent so the endpoint is usable without params.
func parsePeriodQuery(c *gin.Context) (time.Time, time.Time, bool) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")
	if startStr == "" || endStr == "" {
		now := time.Now()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, now, true
	}
	start, err1 := time.Parse("2006-01-02", startStr)
	end, err2 := time.Parse("2006-01-02", endStr)
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}
