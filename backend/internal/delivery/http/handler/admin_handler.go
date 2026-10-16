package handler

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/pagination"
	"github.com/fintrack/backend/pkg/response"
)

type AdminRealtimeEmitter interface {
	PublishUser(ctx context.Context, userID uuid.UUID, ev contracts.Event)
	PublishAdmins(ctx context.Context, ev contracts.Event)
}

type AdminHandler struct {
	userUsecase *usecase.UserUsecase
	realtime    AdminRealtimeEmitter
}

func NewAdminHandler(userUsecase *usecase.UserUsecase) *AdminHandler {
	return &AdminHandler{userUsecase: userUsecase}
}

func (h *AdminHandler) WithRealtime(emitter AdminRealtimeEmitter) *AdminHandler {
	h.realtime = emitter
	return h
}

// ListUsers returns paginated UserStats matching query params (page, per_page, sort, order, q, status).
func (h *AdminHandler) ListUsers(c *gin.Context) {
	pq := pagination.Parse(c, pagination.Options{
		AllowedSort:  []string{"created_at", "full_name", "email", "total_transactions"},
		DefaultSort:  "created_at",
		DefaultOrder: "desc",
	})

	userStats, _, err := h.userUsecase.ListUsers(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to list users")
		return
	}
	if userStats == nil {
		userStats = []domain.UserStats{}
	}

	search := strings.ToLower(strings.TrimSpace(pq.Search))
	statusFilter := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if statusFilter == "" {
		statusFilter = strings.ToLower(strings.TrimSpace(pq.Filters["status"]))
	}

	var filtered []domain.UserStats
	for _, u := range userStats {
		if search != "" {
			nameMatch := strings.Contains(strings.ToLower(u.FullName), search)
			emailMatch := strings.Contains(strings.ToLower(u.Email), search)
			if !nameMatch && !emailMatch {
				continue
			}
		}
		if statusFilter == "active" && !u.IsActive {
			continue
		}
		if (statusFilter == "suspended" || statusFilter == "inactive") && u.IsActive {
			continue
		}
		filtered = append(filtered, u)
	}

	sortCol := pq.Sort
	isAsc := pq.Order == "asc"
	sort.SliceStable(filtered, func(i, j int) bool {
		switch sortCol {
		case "full_name":
			if isAsc {
				return strings.ToLower(filtered[i].FullName) < strings.ToLower(filtered[j].FullName)
			}
			return strings.ToLower(filtered[i].FullName) > strings.ToLower(filtered[j].FullName)
		case "email":
			if isAsc {
				return strings.ToLower(filtered[i].Email) < strings.ToLower(filtered[j].Email)
			}
			return strings.ToLower(filtered[i].Email) > strings.ToLower(filtered[j].Email)
		case "total_transactions":
			if isAsc {
				return filtered[i].TransactionCount < filtered[j].TransactionCount
			}
			return filtered[i].TransactionCount > filtered[j].TransactionCount
		default:
			if isAsc {
				return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
			}
			return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
		}
	})

	total := len(filtered)
	start := pq.Offset()
	if start > total {
		start = total
	}
	end := start + pq.PerPage
	if end > total {
		end = total
	}

	paged := filtered[start:end]
	response.Paginated(c, "Users retrieved successfully", paged, total, pq.Page, pq.PerPage)
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
	case errors.Is(err, domain.ErrNotFound):
		response.NotFound(c, "Pengguna tidak ditemukan")
		return
	case err != nil:
		response.InternalError(c, "Failed to update user status")
		return
	}

	if h.realtime != nil {
		h.realtime.PublishAdmins(c.Request.Context(), contracts.NewEvent(
			contracts.EventUserStatusUpdated,
			time.Now().Unix(),
			gin.H{"id": id.String(), "is_active": *req.IsActive},
		))
		h.realtime.PublishUser(c.Request.Context(), id, contracts.NewEvent(
			contracts.EventUserStatusUpdated,
			time.Now().Unix(),
			gin.H{"id": id.String(), "is_active": *req.IsActive},
		))
		if !*req.IsActive {
			h.realtime.PublishUser(c.Request.Context(), id, contracts.NewEvent(
				"user_suspended",
				time.Now().Unix(),
				gin.H{"user_id": id.String()},
			))
		}
	}

	response.OK(c, "User status updated successfully", gin.H{"id": id, "is_active": *req.IsActive})
}
