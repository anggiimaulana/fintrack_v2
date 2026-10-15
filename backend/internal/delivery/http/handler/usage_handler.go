package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/usage"
	"github.com/fintrack/backend/pkg/pagination"
	"github.com/fintrack/backend/pkg/response"
	"github.com/fintrack/backend/pkg/validator"
)

// UsageHandler exposes the signed-in user's own consumption: a time series for
// charts and a paginated history ledger.
type UsageHandler struct {
	svc  *usage.Service
	repo usage.Repository
}

// NewUsageHandler builds the handler.
func NewUsageHandler(svc *usage.Service, repo usage.Repository) *UsageHandler {
	return &UsageHandler{svc: svc, repo: repo}
}

// historySortWhitelist are the columns the history table may sort by.
var historySortWhitelist = []string{"created_at", "kind", "amount", "cost_idr"}

// Timeseries handles GET /api/v1/me/usage/timeseries.
//
// Query: range=24h|7d|30d|3m|1y|custom, start & end (custom), plus optional
// granularity override is not exposed — the service picks a sensible bucket
// size from the span.
func (h *UsageHandler) Timeseries(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	window, err := h.svc.ResolveWindow(c.Query("range"), c.Query("start"), c.Query("end"))
	if err != nil {
		response.BadRequest(c, "Rentang waktu tidak valid", "invalid_range")
		return
	}

	payload, err := h.svc.Timeseries(c.Request.Context(), userID, window)
	if err != nil {
		response.InternalError(c, "Gagal memuat data penggunaan")
		return
	}
	response.OK(c, "Data penggunaan berhasil dimuat", payload)
}

// History handles GET /api/v1/me/usage/history.
//
// Query: page, per_page, sort (comma list), order (comma list), q,
// filter_kind, start_date, end_date.
func (h *UsageHandler) History(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	page := parseIntDefault(c.Query("page"), 1)
	perPage := parseIntDefault(c.Query("per_page"), 20)
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	f := usage.HistoryFilter{
		Kind:    validator.SanitizeString(c.Query("filter_kind")),
		Search:  validator.SanitizeString(c.Query("q")),
		Page:    page,
		PerPage: perPage,
	}
	if start := parseDateParam(c.Query("start_date"), false); start != nil {
		f.Start = start
	}
	if end := parseDateParam(c.Query("end_date"), true); end != nil {
		f.End = end
	}

	// Multi-sort: `sort=a,b&order=desc,asc`, validated against the whitelist.
	pq := pagination.Parse(c, pagination.Options{
		AllowedSort:  historySortWhitelist,
		DefaultSort:  "created_at",
		DefaultOrder: "desc",
	})
	for _, s := range pq.Sorts {
		f.Sorts = append(f.Sorts, usage.SortField{Column: s.Column, Order: s.Order})
	}

	items, total, err := h.repo.History(c.Request.Context(), userID, f)
	if err != nil {
		response.InternalError(c, "Gagal memuat riwayat penggunaan")
		return
	}
	response.Paginated(c, "Riwayat penggunaan berhasil dimuat", items, total, f.Page, f.PerPage)
}

// parseDateParam parses an RFC3339 timestamp or a bare YYYY-MM-DD date. A bare
// end date is advanced by 24h so the end day is inclusive.
func parseDateParam(raw string, endOfDay bool) *time.Time {
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return nil
		}
		if endOfDay {
			t = t.Add(24 * time.Hour)
		}
	}
	t = t.UTC()
	return &t
}
