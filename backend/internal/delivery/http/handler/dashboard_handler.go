package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

type DashboardHandler struct {
	usecase *usecase.DashboardUsecase
}

func NewDashboardHandler(usecase *usecase.DashboardUsecase) *DashboardHandler {
	return &DashboardHandler{usecase: usecase}
}

func (h *DashboardHandler) GetOverview(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	req := domain.OverviewRequest{
		StartDate:    c.Query("start_date"),
		EndDate:      c.Query("end_date"),
		CompareStart: c.Query("compare_start"),
		CompareEnd:   c.Query("compare_end"),
	}
	overview, err := h.usecase.GetOverview(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Failed to get dashboard overview")
		return
	}
	response.OK(c, "Dashboard overview retrieved successfully", overview)
}

func (h *DashboardHandler) GetCategoryChart(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	req := domain.ChartRequest{
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
	}
	chartData, err := h.usecase.GetCategoryChartData(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Failed to get category chart data")
		return
	}
	response.OK(c, "Category chart data retrieved successfully", chartData)
}

func (h *DashboardHandler) GetChartData(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	req := domain.ChartRequest{
		Type:        c.DefaultQuery("type", "line"),
		Period:      c.DefaultQuery("period", "month"),
		StartDate:   c.Query("start_date"),
		EndDate:     c.Query("end_date"),
		Granularity: c.DefaultQuery("granularity", "month"),
	}
	chartData, err := h.usecase.GetChartData(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Failed to get chart data")
		return
	}
	response.OK(c, "Chart data retrieved successfully", chartData)
}
