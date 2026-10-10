package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

type SPKHandler struct {
	usecase *usecase.SPKUsecase
}

func NewSPKHandler(usecase *usecase.SPKUsecase) *SPKHandler {
	return &SPKHandler{usecase: usecase}
}

func (h *SPKHandler) GetLatest(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	result, err := h.usecase.GetLatest(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Failed to get latest SPK result")
		return
	}
	if result == nil {
		response.NotFound(c, "No SPK result found")
		return
	}
	response.OK(c, "Latest SPK result retrieved successfully", result)
}

func (h *SPKHandler) Calculate(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	result, err := h.usecase.Calculate(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Failed to calculate SPK")
		return
	}
	middleware.InvalidateUserCache(userID)
	response.OK(c, "SPK calculated successfully", result)
}

func (h *SPKHandler) GetHistory(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	results, err := h.usecase.GetHistory(c.Request.Context(), userID, limit)
	if err != nil {
		response.InternalError(c, "Failed to get SPK history")
		return
	}
	if results == nil {
		results = []domain.SPKResult{}
	}
	response.OK(c, "SPK history retrieved successfully", results)
}
