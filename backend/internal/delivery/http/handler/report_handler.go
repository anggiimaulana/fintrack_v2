package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/export"
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
		Period:       c.DefaultQuery("period", "current_month"),
		Type:         c.Query("type"),
		StartDate:    c.Query("start_date"),
		EndDate:      c.Query("end_date"),
		CompareStart: c.Query("compare_start"),
		CompareEnd:   c.Query("compare_end"),
	}

	report, err := h.usecase.Generate(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Failed to generate report")
		return
	}

	response.OK(c, "Report generated successfully", report)
}

// Analytics handles GET /reports/analytics — the deep-dive payload for the
// report tabs (period comparisons, top movers, daily heatmap series, anomalies,
// ratios, best/worst month). Gated by the same `reports_export` feature.
func (h *ReportHandler) Analytics(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	req := domain.ReportRequest{
		Period:       c.DefaultQuery("period", "current_month"),
		Type:         c.Query("type"),
		StartDate:    c.Query("start_date"),
		EndDate:      c.Query("end_date"),
		CompareStart: c.Query("compare_start"),
		CompareEnd:   c.Query("compare_end"),
	}

	analytics, err := h.usecase.Analytics(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Gagal memuat analitik laporan")
		return
	}
	response.OK(c, "Analitik laporan berhasil dimuat", analytics)
}

// Export handles GET /reports/export?format=csv|xlsx|pdf. It reuses the same
// report generation as GET /reports (so the numbers always agree) and streams
// the file with a download disposition. Gated by the same `reports_export`
// feature as the report itself.
func (h *ReportHandler) Export(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	format, ok := export.ParseFormat(c.Query("format"))
	if !ok {
		response.BadRequest(c, "Format tidak didukung. Gunakan csv, xlsx, atau pdf.", "invalid_format")
		return
	}

	req := domain.ReportRequest{
		Period:       c.DefaultQuery("period", "current_month"),
		Type:         c.Query("type"),
		StartDate:    c.Query("start_date"),
		EndDate:      c.Query("end_date"),
		CompareStart: c.Query("compare_start"),
		CompareEnd:   c.Query("compare_end"),
	}

	report, err := h.usecase.Generate(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Gagal membuat laporan")
		return
	}

	// The period label used in the filename: prefer the explicit range, else the
	// preset period name.
	period := req.Period
	if req.StartDate != "" && req.EndDate != "" {
		period = req.StartDate + "_" + req.EndDate
	}

	in := export.ReportInput{
		Report:      report,
		Type:        req.Type,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Period:      period,
		GeneratedAt: time.Now(),
	}
	// Render the report currency with the catalog's configured symbol/precision
	// so a Super-Admin-added currency exports correctly without a code change.
	if symbol, digits, ok := h.usecase.CurrencyFormat(c.Request.Context(), report.CurrencyCode); ok {
		in.CurrencySymbol = symbol
		in.CurrencyDigits = &digits
	}

	payload, err := export.Render(in, format)
	if err != nil {
		response.InternalError(c, "Gagal membuat berkas ekspor")
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+format.Filename(period)+`"`)
	c.Data(http.StatusOK, format.ContentType(), payload)
}
