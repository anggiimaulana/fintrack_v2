package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

type ProviderHandler struct {
	usecase *usecase.ProviderUsecase
}

func NewProviderHandler(usecase *usecase.ProviderUsecase) *ProviderHandler {
	return &ProviderHandler{usecase: usecase}
}

func (h *ProviderHandler) List(c *gin.Context) {
	providers, err := h.usecase.List(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to list providers")
		return
	}
	response.OK(c, "Providers retrieved successfully", providers)
}

func (h *ProviderHandler) Create(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)
	var req domain.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	provider, err := h.usecase.Create(c.Request.Context(), adminID, req)
	if err != nil {
		response.InternalError(c, "Failed to create provider: "+err.Error())
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.Created(c, "Provider created successfully", provider)
}

func (h *ProviderHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid provider ID", err.Error())
		return
	}
	var req domain.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	provider, err := h.usecase.Update(c.Request.Context(), id, req)
	if err != nil {
		response.InternalError(c, "Failed to update provider: "+err.Error())
		return
	}
	if provider == nil {
		response.NotFound(c, "Provider not found")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Provider updated successfully", provider)
}
