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

type GeminiKeyHandler struct {
	usecase *usecase.GeminiKeyUsecase
}

func NewGeminiKeyHandler(usecase *usecase.GeminiKeyUsecase) *GeminiKeyHandler {
	return &GeminiKeyHandler{usecase: usecase}
}

func (h *GeminiKeyHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	keys, err := h.usecase.List(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Failed to list API keys")
		return
	}
	resp := make([]domain.GeminiKeyResponse, len(keys))
	for i, k := range keys {
		resp[i] = k.ToResponse()
	}
	if len(keys) == 0 {
		resp = []domain.GeminiKeyResponse{}
	}
	response.OK(c, "API keys retrieved successfully", resp)
}

func (h *GeminiKeyHandler) Create(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	var req domain.CreateGeminiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	key, err := h.usecase.Create(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateAlias) {
			response.Conflict(c, "Alias sudah digunakan, gunakan nama lain.")
			return
		}
		response.InternalError(c, "Failed to create API key")
		return
	}
	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.Created(c, "API key created successfully", key.ToResponse())
}

func (h *GeminiKeyHandler) Update(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid key ID", err.Error())
		return
	}
	var req domain.UpdateGeminiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	key, err := h.usecase.Update(c.Request.Context(), userID, id, req)
	if err != nil {
		response.InternalError(c, "Failed to update API key")
		return
	}
	if key == nil {
		response.NotFound(c, "API key not found")
		return
	}
	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.OK(c, "API key updated successfully", key.ToResponse())
}

func (h *GeminiKeyHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid key ID", err.Error())
		return
	}
	if err := h.usecase.Delete(c.Request.Context(), userID, id); err != nil {
		response.InternalError(c, "Failed to delete API key")
		return
	}
	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.OK(c, "API key deleted successfully", nil)
}

func (h *GeminiKeyHandler) Test(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid key ID", err.Error())
		return
	}
	result, err := h.usecase.Test(c.Request.Context(), userID, id)
	if err != nil {
		response.InternalError(c, "Failed to test API key: "+err.Error())
		return
	}
	response.OK(c, "API key test completed", result)
}
