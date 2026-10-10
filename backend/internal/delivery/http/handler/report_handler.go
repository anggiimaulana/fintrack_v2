package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

type ReportHandler struct {
	usecase *usecase.ReportUsecase
}

func NewReportHandler(usecase *usecase.ReportUsecase) *ReportHandler {
	return &ReportHandler{usecase: usecase}
}

func (h *ReportHandler) Generate(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	req := domain.ReportRequest{
		Period:      c.DefaultQuery("period", "current_month"),
		Type:        c.Query("type"),
		StartDate:   c.Query("start_date"),
		EndDate:     c.Query("end_date"),
		CompareStart: c.Query("compare_start"),
		CompareEnd:  c.Query("compare_end"),
	}

	report, err := h.usecase.Generate(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Failed to generate report")
		return
	}

	response.OK(c, "Report generated successfully", report)
}
