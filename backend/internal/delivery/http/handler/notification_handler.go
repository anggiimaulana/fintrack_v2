package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/mail"
	"github.com/fintrack/backend/internal/notification"
	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/response"
	"github.com/fintrack/backend/pkg/validator"
)

// NotificationHandler exposes the user's notification inbox and preferences.
type NotificationHandler struct {
	repo    domain.NotificationRepository
	service *notification.Service
}

func NewNotificationHandler(repo domain.NotificationRepository, service *notification.Service) *NotificationHandler {
	return &NotificationHandler{repo: repo, service: service}
}

// GetPreferences handles GET /me/notification-preferences.
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	prefs, err := h.repo.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Gagal memuat preferensi notifikasi")
		return
	}
	response.OK(c, "Preferensi notifikasi berhasil dimuat", prefs)
}

// UpdatePreferences handles PUT /me/notification-preferences.
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req domain.NotificationPreferences
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Format preferensi tidak valid", "validation")
		return
	}

	if err := h.repo.SavePreferences(c.Request.Context(), userID, req); err != nil {
		response.InternalError(c, "Gagal menyimpan preferensi notifikasi")
		return
	}
	response.OK(c, "Preferensi notifikasi berhasil disimpan", req)
}

// List handles GET /me/notifications, returning one page of the inbox plus
// stats and pagination meta. Backs both the header bell (small limit) and the
// full notifications page (page/per_page/sort/order/q/filter_status).
func (h *NotificationHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	ctx := c.Request.Context()

	// The bell calls without page/per_page and only needs a handful of rows; the
	// page passes explicit table params. Support both from one endpoint.
	limit := 0
	if v := c.Query("limit"); v != "" {
		limit = parseIntDefault(v, 0)
	}

	filter := domain.NotificationFilter{
		Page:    parseIntDefault(c.Query("page"), 1),
		PerPage: parseIntDefault(c.Query("per_page"), 20),
		Sort:    c.Query("sort"),
		Order:   c.Query("order"),
		Search:  validator.SanitizeString(c.Query("q")),
		Status:  c.Query("filter_status"),
	}
	if filter.Sort == "" {
		filter.Sort = "created_at"
	}
	if filter.Order == "" {
		filter.Order = "desc"
	}

	items, total, err := h.repo.ListPaged(ctx, userID, filter)
	if err != nil {
		response.InternalError(c, "Gagal memuat notifikasi")
		return
	}

	// `limit` is a convenience for the bell: return only the newest N rows. The
	// page omits it and uses page/per_page. Both share this one endpoint so there
	// is a single cache entry shape.
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}

	response.Paginated(c, "Notifikasi berhasil dimuat", items, total, filter.Page, filter.PerPage)
}

// Stats handles GET /me/notifications/stats.
func (h *NotificationHandler) Stats(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	stats, err := h.repo.Stats(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Gagal memuat statistik notifikasi")
		return
	}
	response.OK(c, "Statistik notifikasi berhasil dimuat", stats)
}

// GetByID handles GET /me/notifications/:id.
func (h *NotificationHandler) GetByID(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID notifikasi tidak valid", "validation")
		return
	}
	n, err := h.repo.GetByID(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotificationNotFound) {
			response.NotFound(c, "Notifikasi tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal memuat notifikasi")
		return
	}
	response.OK(c, "Notifikasi berhasil dimuat", n)
}

// MarkRead handles POST /me/notifications/:id/read.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID notifikasi tidak valid", "validation")
		return
	}
	if err := h.repo.MarkRead(c.Request.Context(), userID, id); err != nil {
		response.InternalError(c, "Gagal menandai notifikasi")
		return
	}
	h.emitRead(c, userID, []uuid.UUID{id})
	response.OK(c, "Notifikasi ditandai sudah dibaca", nil)
}

// MarkReadBulk handles POST /me/notifications/read with a JSON body of ids.
// An empty/missing `ids` marks ALL of the user's notifications read, so the
// header bell's "mark all" and the page's bulk action share one endpoint.
func (h *NotificationHandler) MarkReadBulk(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	ids := parseIDList(c)

	if len(ids) == 0 {
		if err := h.repo.MarkAllRead(c.Request.Context(), userID); err != nil {
			response.InternalError(c, "Gagal menandai notifikasi")
			return
		}
		h.emitRead(c, userID, nil) // nil = all
		response.OK(c, "Semua notifikasi ditandai sudah dibaca", nil)
		return
	}

	if err := h.repo.MarkReadMany(c.Request.Context(), userID, ids); err != nil {
		response.InternalError(c, "Gagal menandai notifikasi")
		return
	}
	h.emitRead(c, userID, ids)
	response.OK(c, "Notifikasi ditandai sudah dibaca", nil)
}

// MarkAllRead handles POST /me/notifications/read-all (kept for the bell).
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.repo.MarkAllRead(c.Request.Context(), userID); err != nil {
		response.InternalError(c, "Gagal menandai notifikasi")
		return
	}
	h.emitRead(c, userID, nil)
	response.OK(c, "Semua notifikasi ditandai sudah dibaca", nil)
}

// DeleteOne handles DELETE /me/notifications/:id.
func (h *NotificationHandler) DeleteOne(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID notifikasi tidak valid", "validation")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), userID, id); err != nil {
		response.InternalError(c, "Gagal menghapus notifikasi")
		return
	}
	h.emitDeleted(c, userID, []uuid.UUID{id})
	response.OK(c, "Notifikasi berhasil dihapus", nil)
}

// DeleteBulk handles POST /me/notifications/delete with a JSON body of ids.
func (h *NotificationHandler) DeleteBulk(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	ids := parseIDList(c)
	if len(ids) == 0 {
		response.BadRequest(c, "Tidak ada notifikasi yang dipilih", "validation")
		return
	}
	if err := h.repo.DeleteMany(c.Request.Context(), userID, ids); err != nil {
		response.InternalError(c, "Gagal menghapus notifikasi")
		return
	}
	h.emitDeleted(c, userID, ids)
	response.OK(c, "Notifikasi berhasil dihapus", nil)
}

// Clear handles DELETE /me/notifications (remove everything).
func (h *NotificationHandler) Clear(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.repo.Clear(c.Request.Context(), userID); err != nil {
		response.InternalError(c, "Gagal menghapus notifikasi")
		return
	}
	h.emitDeleted(c, userID, nil) // nil = all
	response.OK(c, "Notifikasi berhasil dibersihkan", nil)
}

// emitRead publishes the read event so other tabs/pages patch in place.
func (h *NotificationHandler) emitRead(c *gin.Context, userID uuid.UUID, ids []uuid.UUID) {
	if h.service == nil {
		return
	}
	payload := contracts.NotificationChangedPayload{IDs: uuidStrings(ids), Unread: h.unreadCount(c, userID)}
	h.service.EmitNotification(c.Request.Context(), userID, contracts.EventNotificationRead, payload)
}

// emitDeleted publishes the delete event.
func (h *NotificationHandler) emitDeleted(c *gin.Context, userID uuid.UUID, ids []uuid.UUID) {
	if h.service == nil {
		return
	}
	payload := contracts.NotificationChangedPayload{IDs: uuidStrings(ids), Unread: h.unreadCount(c, userID)}
	h.service.EmitNotification(c.Request.Context(), userID, contracts.EventNotificationDeleted, payload)
}

func (h *NotificationHandler) unreadCount(c *gin.Context, userID uuid.UUID) int {
	n, err := h.repo.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		return 0
	}
	return n
}

// parseIDList reads a JSON `{ "ids": [...] }` body, tolerating a missing body.
func parseIDList(c *gin.Context) []uuid.UUID {
	var body struct {
		IDs []string `json:"ids"`
	}
	_ = c.ShouldBindJSON(&body)
	out := make([]uuid.UUID, 0, len(body.IDs))
	for _, s := range body.IDs {
		if id, err := uuid.Parse(s); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// SendTestEmail handles POST /me/notifications/test-email: sends a one-off test
// message to the caller's own address so they can verify delivery end-to-end.
func (h *NotificationHandler) SendTestEmail(c *gin.Context) {
	email := c.MustGet("email").(string)

	if h.service == nil {
		response.InternalError(c, "Layanan notifikasi belum dikonfigurasi")
		return
	}
	if err := h.service.SendTestEmail(c.Request.Context(), email); err != nil {
		if errors.Is(err, mail.ErrDisabled) {
			response.Error(c, 409, "Email belum dikonfigurasi. Atur SMTP terlebih dahulu di panel Super Admin.", "mail_disabled")
			return
		}
		response.Error(c, 502, "Gagal mengirim email uji. Periksa konfigurasi SMTP.", "mail_failed")
		return
	}
	response.OK(c, "Email uji berhasil dikirim", gin.H{"to": email})
}
