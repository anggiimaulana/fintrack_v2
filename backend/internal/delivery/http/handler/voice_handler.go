package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/internal/voice"
	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/response"
)

// maxTranscriptLen bounds a recorded transcript so a malicious client cannot
// write unbounded text into the history table.
const maxTranscriptLen = 8000

// maxLanguageLen matches the varchar(16) language column; maxDurationMS bounds
// a single recording to 10 minutes so a bad client value cannot overflow the
// integer column (the recorder already caps at the configured duration).
const (
	maxLanguageLen = 16
	maxDurationMS  = 600_000
)

// VoiceHandler exposes the voice-input configuration and the per-user
// transcription history.
type VoiceHandler struct {
	repo      domain.VoiceTranscriptionRepository
	ent       *entitlement.Resolver
	cfg       voice.ConfigReader
	emitter   UsageEmitter
	now       func() time.Time
}

// NewVoiceHandler builds the handler. `ent` may be nil in tests that do not
// exercise the daily limit; the config reader may be nil (defaults apply).
func NewVoiceHandler(repo domain.VoiceTranscriptionRepository, ent *entitlement.Resolver, cfg voice.ConfigReader) *VoiceHandler {
	return &VoiceHandler{repo: repo, ent: ent, cfg: cfg, now: func() time.Time { return time.Now().UTC() }}
}

// WithUsageEmitter installs the realtime emitter used to announce daily-limit
// changes so the client's quota bars update live.
func (h *VoiceHandler) WithUsageEmitter(e UsageEmitter) *VoiceHandler {
	h.emitter = e
	return h
}

// WithClock overrides the time source (tests).
func (h *VoiceHandler) WithClock(now func() time.Time) *VoiceHandler {
	h.now = now
	return h
}

// Config handles GET /me/voice-config — the effective, dynamic voice
// configuration plus the caller's daily-limit state.
func (h *VoiceHandler) Config(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	ctx := c.Request.Context()

	cfg := voice.Load(ctx, h.cfg)

	// Resolve entitlements ONCE and reuse for both the limit and the feature, so
	// the config endpoint does not pay for two resolution round-trips.
	var limits map[string]int64
	serverAvailable := false
	if h.ent != nil {
		res, err := h.ent.Resolve(ctx, userID)
		if err != nil {
			response.InternalError(c, "Gagal memuat konfigurasi suara")
			return
		}
		limits = res.Limits
		serverAvailable = res.Feature("voice_input")
	} else {
		limits = map[string]int64{}
	}

	daily := h.dailyUsage(ctx, userID, limits, cfg)

	response.OK(c, "Konfigurasi suara berhasil dimuat", gin.H{
		"max_duration_seconds":    cfg.MaxDurationSeconds,
		"language_mode":           cfg.LanguageMode,
		"default_language":        cfg.DefaultLanguage,
		"languages":               cfg.Languages,
		"language_per_recording":  cfg.LanguageMode == voice.LanguageModeMulti,
		"daily_limit":             daily,
		"daily_reset_timezone":    cfg.ResetTimezone,
		"daily_resets_at":         voice.ResetsAt(h.now(), cfg.ResetTimezone),
		"history_retention_days":  cfg.HistoryRetentionDay,
		"allow_edit_before_send":  cfg.AllowEditBeforeSend,
		"vu_meter_enabled":        cfg.VUMeterEnabled,
		"fallback_to_browser":     cfg.FallbackToBrowser,
		"server_enabled":          cfg.ServerEnabled,
		"server_available":        serverAvailable,
	})
}

// ListHistory handles GET /me/voice-history.
func (h *VoiceHandler) ListHistory(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	page := parseIntDefault(c.Query("page"), 1)
	perPage := parseIntDefault(c.Query("per_page"), 20)

	items, total, err := h.repo.ListByUser(c.Request.Context(), userID, page, perPage)
	if err != nil {
		response.InternalError(c, "Gagal memuat riwayat suara")
		return
	}
	if items == nil {
		items = []domain.VoiceTranscription{}
	}
	response.Paginated(c, "Riwayat suara berhasil dimuat", items, total, page, perPage)
}

// Record handles POST /me/voice-history — the client reports a BROWSER-mode
// transcription so the daily limit counts it (server-mode rows are written by
// the /ai/voice handler). The route is gated by limitGate(voice_requests_per_day).
func (h *VoiceHandler) Record(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req domain.RecordVoiceTranscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	transcript := strings.TrimSpace(req.Transcript)
	if transcript == "" {
		response.BadRequest(c, "Transkrip tidak boleh kosong", "validation")
		return
	}
	if len(transcript) > maxTranscriptLen {
		response.BadRequest(c, "Transkrip terlalu panjang", "validation")
		return
	}

	cfg := voice.Load(c.Request.Context(), h.cfg)
	lang := strings.TrimSpace(req.Language)
	if lang == "" {
		lang = cfg.DefaultLanguage
	}
	// The DB column is varchar(16); validate here so a bad value is a 400, not a
	// 500 from the database.
	if len(lang) > maxLanguageLen {
		response.BadRequest(c, "Kode bahasa tidak valid", "validation")
		return
	}
	// duration_ms is a non-negative integer column; clamp the input so a
	// negative or absurd value cannot overflow it.
	duration := req.DurationMS
	if duration < 0 {
		duration = 0
	}
	if duration > maxDurationMS {
		duration = maxDurationMS
	}

	// Claim a daily slot ATOMICALLY before writing the row. The middleware gate
	// is a pre-check; this conditional increment is what makes the cap exact
	// under concurrency (two simultaneous requests cannot both exceed the limit).
	day, _ := voice.DayWindow(h.now(), cfg.ResetTimezone)
	limit := int64(0)
	if h.ent != nil {
		if res, rerr := h.ent.Resolve(c.Request.Context(), userID); rerr == nil {
			limit = res.Limit(voiceLimitKey)
		}
	}
	// A limit of 0 means "not in plan"; -1 is unlimited. Both bypass the
	// conditional claim (the gate already decided), so only a positive finite
	// limit is consumed here.
	if limit > 0 {
		granted, _, cerr := h.repo.ConsumeDailyUsage(c.Request.Context(), userID, day, limit)
		if cerr != nil {
			response.InternalError(c, "Gagal mencatat pemakaian suara")
			return
		}
		if !granted {
			c.JSON(402, gin.H{
				"success": false, "error": "limit_exceeded",
				"message": "Anda telah mencapai batas suara harian. Silakan tingkatkan paket.",
				"limit_key": voiceLimitKey, "limit_value": limit, "upgrade_hint": true,
			})
			return
		}
	}

	t := &domain.VoiceTranscription{
		UserID:     userID,
		Transcript: transcript,
		Language:   lang,
		Mode:       domain.VoiceModeBrowser,
		DurationMS: duration,
	}
	if err := h.repo.Create(c.Request.Context(), t); err != nil {
		response.InternalError(c, "Gagal menyimpan riwayat suara")
		return
	}

	emitUsageChanged(c.Request.Context(), h.emitter, userID, "voice_browser", UsageKeyVoicePerDay)
	response.Created(c, "Riwayat suara tersimpan", t)
}

// DeleteOne handles DELETE /me/voice-history/:id.
func (h *VoiceHandler) DeleteOne(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID riwayat tidak valid", "invalid_id")
		return
	}
	if err := h.repo.DeleteOne(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Riwayat suara tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal menghapus riwayat suara")
		return
	}
	// Deleting history does NOT refund the daily counter (append-only guard),
	// so no usage event is emitted here.
	response.OK(c, "Riwayat suara dihapus", nil)
}

// Clear handles DELETE /me/voice-history.
func (h *VoiceHandler) Clear(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.repo.ClearByUser(c.Request.Context(), userID); err != nil {
		response.InternalError(c, "Gagal membersihkan riwayat suara")
		return
	}
	response.OK(c, "Riwayat suara dibersihkan", nil)
}

// ─── helpers ─────────────────────────────────────────────────────

// dailyUsage computes the caller's daily voice consumption against the plan
// limit. It reads the append-only counter (not the history rows) so deleting
// history cannot refund the day (R-14).
//
// A plan WITHOUT the limit key is reported as "not available" (limit 0) rather
// than "unlimited": the middleware gate fails closed on an absent key, so the
// UI must agree with the gate instead of showing an enabled mic that will 402.
func (h *VoiceHandler) dailyUsage(ctx context.Context, userID uuid.UUID, limits map[string]int64, cfg voice.Config) entitlement.UsageEntry {
	limit, ok := limits[voiceLimitKey]
	if !ok {
		return entitlement.UsageEntry{Limit: 0, Remaining: 0, Unlimited: false}
	}
	day, _ := voice.DayWindow(h.now(), cfg.ResetTimezone)
	used, err := h.repo.DailyUsage(ctx, userID, day)
	if err != nil {
		// A counter failure must not fail the config read; report unknown usage
		// as zero against the limit (the gate remains authoritative).
		used = 0
	}
	entry := entitlement.UsageEntry{Limit: limit, Used: used}
	if billing.IsUnlimited(limit) {
		entry.Unlimited = true
		entry.Remaining = -1
		return entry
	}
	entry.Remaining = limit - used
	if entry.Remaining < 0 {
		entry.Remaining = 0
	}
	return entry
}

// voiceLimitKey mirrors the entitlement limit key for the daily voice cap.
const voiceLimitKey = "voice_requests_per_day"

// compile-time guards.
var (
	_ = contracts.EventUsageChanged
	_ = middleware.InvalidateUserCache
)
