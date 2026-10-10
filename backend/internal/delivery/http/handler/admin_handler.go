package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

type AdminHandler struct {
	userUsecase *usecase.UserUsecase
}

func NewAdminHandler(userUsecase *usecase.UserUsecase) *AdminHandler {
	return &AdminHandler{userUsecase: userUsecase}
}

// ListUsers returns `data` as an array of UserStats (contract expected by FE).
func (h *AdminHandler) ListUsers(c *gin.Context) {
	userStats, _, err := h.userUsecase.ListUsers(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to list users")
		return
	}
	if userStats == nil {
		userStats = []domain.UserStats{}
	}
	response.OK(c, "Users retrieved successfully", userStats)
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	_, adminStats, err := h.userUsecase.ListUsers(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to get admin stats")
		return
	}
	response.OK(c, "Admin stats retrieved successfully", adminStats)
}

// SetUserStatus toggles a user's active flag. Body: {"is_active": bool}.
func (h *AdminHandler) SetUserStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user id", "invalid_id")
		return
	}

	var req struct {
		IsActive *bool `json:"is_active" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(uuid.UUID)

	err = h.userUsecase.SetUserActive(c.Request.Context(), id, *req.IsActive, actorID)
	switch {
	case errors.Is(err, usecase.ErrSelfDeactivate):
		response.BadRequest(c, "Tidak bisa menonaktifkan akun sendiri", "self_deactivate")
		return
	case err != nil:
		response.InternalError(c, "Failed to update user status")
		return
	}

	response.OK(c, "User status updated successfully", gin.H{"id": id, "is_active": *req.IsActive})
}
