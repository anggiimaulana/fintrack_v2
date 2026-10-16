package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/internal/voice"
	"github.com/fintrack/backend/pkg/logger"
	"github.com/fintrack/backend/pkg/response"
)

type AIHandler struct {
	aiUC          *usecase.AIUsecase
	transactionUC *usecase.TransactionUsecase
	ledger        *billing.LedgerService
	// estimates supplies per-modality reserve amounts (P3), editable by a Super
	// Admin. Nil falls back to a built-in default.
	estimates *billing.EstimateStore
	// usageEmitter publishes usage_changed after AI consumption so the quota
	// bars refresh live. Nil disables emission (tests).
	usageEmitter UsageEmitter
	// voiceRepo records server-mode transcriptions (history + daily counter).
	// Nil disables recording (tests that predate the feature).
	voiceRepo domain.VoiceTranscriptionRepository
	// voiceCfg resolves the default language for a voice request. Nil falls back
	// to the package default.
	voiceCfg voice.ConfigReader
	// voiceDaily atomically claims a daily slot for a server-mode transcription
	// given the plan limit, returning (granted, newValue). Nil disables it.
	voiceDaily func(ctx context.Context, userID uuid.UUID, limit int64) (bool, int64, error)
	// entResolver resolves the plan limit for the daily-slot claim. Nil skips the
	// conditional claim (the middleware gate still applies).
	entResolver *entitlement.Resolver
}

func NewAIHandler(aiUC *usecase.AIUsecase, transactionUC *usecase.TransactionUsecase) *AIHandler {
	return &AIHandler{aiUC: aiUC, transactionUC: transactionUC}
}

// WithVoice installs the voice-history repository and config reader so
// /ai/voice records transcriptions and resolves the default language.
func (h *AIHandler) WithVoice(repo domain.VoiceTranscriptionRepository, cfg voice.ConfigReader) *AIHandler {
	h.voiceRepo = repo
	h.voiceCfg = cfg
	return h
}

// WithVoiceDailyCounter installs the atomic daily-slot claim used after a
// successful server-mode transcription (R-14 + concurrency safety).
func (h *AIHandler) WithVoiceDailyCounter(fn func(ctx context.Context, userID uuid.UUID, limit int64) (bool, int64, error)) *AIHandler {
	h.voiceDaily = fn
	return h
}

// WithEntitlementsResolver installs the resolver used to read the daily voice
// limit for the conditional slot claim.
func (h *AIHandler) WithEntitlementsResolver(r *entitlement.Resolver) *AIHandler {
	h.entResolver = r
	return h
}

func (h *AIHandler) WithLedger(l *billing.LedgerService) *AIHandler {
	h.ledger = l
	return h
}

// WithUsageEmitter installs the realtime emitter used to announce quota changes.
func (h *AIHandler) WithUsageEmitter(e UsageEmitter) *AIHandler {
	h.usageEmitter = e
	return h
}

// WithEstimates installs the per-modality reserve estimator (P3).
func (h *AIHandler) WithEstimates(e *billing.EstimateStore) *AIHandler {
	h.estimates = e
	return h
}

// estimateFor returns the reserve amount for a modality. It prefers the
// configured estimate and never returns less than 1 (a reserve must hold
// something so the quota gate is meaningful).
func (h *AIHandler) estimateFor(ctx context.Context, modality billing.Modality) int64 {
	if h.estimates != nil {
		if v := h.estimates.For(ctx, modality); v > 0 {
			return v
		}
	}
	def := billing.DefaultQuotaEstimate()
	switch modality {
	case billing.ModalityVision:
		return def.Vision
	case billing.ModalityVoice:
		return def.Voice
	case billing.ModalityEmbedding:
		return def.Embedding
	case billing.ModalityReasoning:
		return def.Reasoning
	default:
		return def.Chat
	}
}

// chatEstimate returns the reserve for one chat turn. It scales with the
// request so a long conversation is not under-reserved (which would clamp the
// commit and silently drain the cycle) and a short one is not over-reserved
// (which would refuse a request that actually fits).
func (h *AIHandler) chatEstimate(ctx context.Context, modality billing.Modality, message string) int64 {
	if h.estimates != nil {
		if v := h.estimates.ForChat(ctx, modality, int64(len(message))); v > 0 {
			return v
		}
	}
	return h.estimateFor(ctx, modality)
}

func (h *AIHandler) visionEstimate(ctx context.Context, imageBytes int64) int64 {
	if h.estimates != nil {
		const approxReceiptPromptChars = 800
		if v := h.estimates.ForVision(ctx, imageBytes, approxReceiptPromptChars); v > 0 {
			return v
		}
	}
	return h.estimateFor(ctx, billing.ModalityVision)
}

func (h *AIHandler) voiceEstimate(ctx context.Context, audioBytes int64) int64 {
	if h.estimates != nil {
		if v := h.estimates.ForVoice(ctx, audioBytes); v > 0 {
			return v
		}
	}
	return h.estimateFor(ctx, billing.ModalityVoice)
}

// resultSucceeded reports whether a vision/voice result map represents success.
// The adapters swallow upstream errors into a map with success=false, so callers
// must inspect the map, not just the returned error, before charging the ledger.
func resultSucceeded(res map[string]interface{}) bool {
	if res == nil {
		return false
	}
	ok, _ := res["success"].(bool)
	return ok
}

// ─── Chat with SSE Streaming ───────────────────────────────────

func (h *AIHandler) Chat(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req domain.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", err.Error())
		return
	}

	if req.Message == "" {
		response.BadRequest(c, "Message is required", "validation_error")
		return
	}

	var reservation *billing.Reservation
	if h.ledger != nil {
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		idempotencyKey := "chat-reserve:" + reqID
		// Reasoning mode is billed at its own weight (reasoning multiplier),
		// mirroring the use case it is routed to upstream.
		modality := billing.ModalityChat
		if req.Mode == domain.LLMReasoning {
			modality = billing.ModalityReasoning
		}
		res, err := h.ledger.Reserve(c.Request.Context(), userID, h.chatEstimate(c.Request.Context(), modality, req.Message), modality, idempotencyKey)
		if err != nil {
			if errors.Is(err, billing.ErrInsufficientQuota) {
				response.QuotaExceeded(c, "Sisa kuota token Anda tidak mencukupi untuk melakukan permintaan ini.")
				return
			}
			response.InternalError(c, "Gagal mengalokasikan kuota token")
			return
		}
		reservation = res
	}
	var sessionID *uuid.UUID
	if sid := c.Query("session_id"); sid != "" {
		parsed, err := uuid.Parse(sid)
		if err == nil {
			sessionID = &parsed
		}
	}

	source := c.Query("source")
	if source == "" {
		source = "chatbot"
	}

	msgChan := make(chan domain.ChatStreamEvent, 100)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("chat worker panicked", zap.Any("panic", r))
			}
		}()
		h.aiUC.Chat(c.Request.Context(), userID, req, sessionID, source, msgChan)
	}()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	var gotContent bool
	// noCharge is set when the turn consumed no billable AI work (kill switch,
	// content filter, cooldown, gateway error, canned template). The reservation
	// is released so the user is not billed for an unserved turn.
	var noCharge bool
	// actualTokens accumulates the real token usage reported on the terminal
	// event. It defaults to the reserved estimate so the ledger always settles
	// with a sane value even if the provider omits usage.
	actualTokens := int64(1)
	if reservation != nil && reservation.RawTokens > 0 {
		actualTokens = reservation.RawTokens
	} else if reservation != nil && reservation.Amount > 0 {
		actualTokens = reservation.Amount
	}
	defer func() {
		if reservation == nil {
			return
		}
		ctx := context.Background()
		switch {
		case noCharge:
			// Nothing billable happened: return the held quota untouched.
			if err := h.ledger.Release(ctx, userID, reservation.ID, "release:"+reservation.ID.String()); err != nil {
				logger.Error("chat release failed", zap.Error(err), zap.String("user_id", userID.String()), zap.String("reservation_id", reservation.ID.String()))
			}
		case gotContent:
			if err := h.ledger.Commit(ctx, userID, reservation.ID, actualTokens, "commit:"+reservation.ID.String()); err != nil {
				logger.Error("chat commit failed", zap.Error(err), zap.String("user_id", userID.String()), zap.String("reservation_id", reservation.ID.String()))
			}
		default:
			// No content and not explicitly flagged: treat as unserved.
			if err := h.ledger.Release(ctx, userID, reservation.ID, "release:"+reservation.ID.String()); err != nil {
				logger.Error("chat unserved release failed", zap.Error(err), zap.String("user_id", userID.String()), zap.String("reservation_id", reservation.ID.String()))
			}
		}
		// A reservation was made, so the request counter moved and the token
		// bucket settled. Announce the change for live quota bars.
		emitUsageChanged(ctx, h.usageEmitter, userID, "ai_chat",
			UsageKeyAITokens, UsageKeyAIRequests)
	}()

	c.Stream(func(w io.Writer) bool {
		event, ok := <-msgChan
		if !ok {
			return false
		}
		if event.Type == "content" && event.Content != "" {
			gotContent = true
		}
		if event.Type == "done" {
			if event.NoCharge {
				noCharge = true
			}
			if event.Usage != nil && event.Usage.TotalTokens > 0 {
				actualTokens = int64(event.Usage.TotalTokens)
			}
		}

		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		} else if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}

		if event.Done {
			return false
		}
		return true
	})
}

// ─── Sessions CRUD ─────────────────────────────────────────────

func (h *AIHandler) CreateSession(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		Title  string `json:"title" binding:"required"`
		Source string `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Title is required", err.Error())
		return
	}

	session, err := h.aiUC.CreateSession(c.Request.Context(), userID, req.Title, req.Source)
	if err != nil {
		response.InternalError(c, "Failed to create session")
		return
	}

	response.Created(c, "Session created", session)
}

func (h *AIHandler) ListSessions(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	source := c.Query("source")

	sessions, err := h.aiUC.ListSessions(c.Request.Context(), userID, source)
	if err != nil {
		response.InternalError(c, "Failed to list sessions")
		return
	}

	if sessions == nil {
		sessions = []domain.ChatSession{}
	}

	response.OK(c, "Sessions retrieved", sessions)
}

func (h *AIHandler) UpdateSession(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid session ID", err.Error())
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Title is required", err.Error())
		return
	}

	session, err := h.aiUC.UpdateSession(c.Request.Context(), userID, id, req.Title)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Sesi tidak ditemukan")
			return
		}
		response.InternalError(c, "Failed to update session")
		return
	}

	response.OK(c, "Session updated", session)
}

func (h *AIHandler) DeleteSession(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid session ID", err.Error())
		return
	}

	if err := h.aiUC.DeleteSession(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Sesi tidak ditemukan")
			return
		}
		response.InternalError(c, "Failed to delete session")
		return
	}

	response.OK(c, "Session deleted", nil)
}

func (h *AIHandler) ListMessages(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid session ID", err.Error())
		return
	}

	messages, err := h.aiUC.ListMessages(c.Request.Context(), userID, sessionID)
	if err != nil {
		response.InternalError(c, "Failed to list messages")
		return
	}

	if messages == nil {
		messages = []domain.ChatMessage{}
	}

	response.OK(c, "Messages retrieved", messages)
}

func (h *AIHandler) CreateMessage(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid session ID", err.Error())
		return
	}

	var req struct {
		Role    string  `json:"role" binding:"required"`
		Content string  `json:"content" binding:"required"`
		Mode    *string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Role and content are required", err.Error())
		return
	}

	msg, err := h.aiUC.CreateMessage(c.Request.Context(), userID, sessionID, req.Role, req.Content, req.Mode)
	if err != nil {
		response.InternalError(c, "Failed to create message")
		return
	}

	response.Created(c, "Message created", msg)
}
// ─── Receipt Analysis ──────────────────────────────────────────

func (h *AIHandler) AnalyzeReceipt(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		response.BadRequest(c, "Image file is required", err.Error())
		return
	}
	defer file.Close()

	if header.Size > 2*1024*1024 {
		response.BadRequest(c, "File size exceeds 2MB limit. Kompres dulu gambar Anda.", "file_too_large")
		return
	}

	contentType := header.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/webp": true,
		"image/heic": true,
	}
	if !allowedTypes[contentType] {
		response.BadRequest(c, "File type not supported. Use JPG, PNG, WEBP, or HEIC.", "invalid_type")
		return
	}

	imageData, err := io.ReadAll(file)
	if err != nil {
		logger.Error("failed to read receipt file", zap.Error(err))
		response.InternalError(c, "Failed to read file")
		return
	}

	// Charge the vision scan to the ledger (reserve → call → commit/release).
	// The estimated amount is 1 unit of the modality weight; the commit settles
	// the provider-reported (or estimated) token count. This is what makes
	// `ocr_scans_per_cycle` count and `ai_tokens_per_cycle` decrease for OCR.
	var reservation *billing.Reservation
	if h.ledger != nil {
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		res, rerr := h.ledger.Reserve(c.Request.Context(), userID, h.visionEstimate(c.Request.Context(), header.Size), billing.ModalityVision, "vision-reserve:"+reqID)
		if rerr != nil {
			if errors.Is(rerr, billing.ErrInsufficientQuota) {
				response.QuotaExceeded(c, "Sisa kuota Anda tidak mencukupi untuk memindai struk ini.")
				return
			}
			response.InternalError(c, "Gagal mengalokasikan kuota")
			return
		}
		reservation = res
	}

	result, usage, err := h.aiUC.AnalyzeReceipt(c.Request.Context(), userID, imageData, contentType)

	// Settle the reservation. The user is only charged when the scan actually
	// produced a usable result; any other outcome returns the held quota.
	if reservation != nil {
		settleVision(h.ledger, userID, reservation, result, usage, err)
		// The scan attempt is counted either way (the reserve entry is what
		// `ocr_scans_per_cycle` measures), so announce the change regardless.
		emitUsageChanged(c.Request.Context(), h.usageEmitter, userID, "ai_ocr",
			UsageKeyAITokens, UsageKeyAIRequests, UsageKeyOCRScans)
	}

	if err != nil {
		logger.Error("receipt analysis failed", zap.Error(err))
		response.InternalError(c, "Gagal menganalisis struk belanja. Silakan coba beberapa saat lagi.")
		return
	}

	response.Created(c, "Receipt analyzed", result)
}

// settleVision commits the real vision token usage on success and releases the
// reservation otherwise, so a failed scan never costs the user quota.
func settleVision(
	ledger *billing.LedgerService,
	userID uuid.UUID,
	reservation *billing.Reservation,
	result map[string]interface{},
	usage domain.TokenUsage,
	callErr error,
) {
	ctx := context.Background()
	if callErr == nil && resultSucceeded(result) {
		actual := int64(usage.TotalTokens)
		if actual <= 0 && reservation != nil {
			actual = reservation.RawTokens
			if actual <= 0 {
				actual = reservation.Amount
			}
		}
		if actual <= 0 {
			actual = 1
		}
		if err := ledger.Commit(ctx, userID, reservation.ID, actual, "vision-commit:"+reservation.ID.String()); err != nil {
			logger.Error("vision commit failed", zap.Error(err), zap.String("user_id", userID.String()), zap.String("reservation_id", reservation.ID.String()))
		}
		return
	}
	// Unusable result or error: return the reserved quota untouched.
	if err := ledger.Release(ctx, userID, reservation.ID, "vision-release:"+reservation.ID.String()); err != nil {
		logger.Error("vision release failed", zap.Error(err), zap.String("user_id", userID.String()), zap.String("reservation_id", reservation.ID.String()))
	}
}

func (h *AIHandler) CreateTransactionFromReceipt(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req domain.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", err.Error())
		return
	}

	if req.Type == domain.TransactionIncome && req.IncomeCategoryID == nil {
		response.BadRequest(c, "Income category is required for income transactions", "validation_error")
		return
	}
	if req.Type == domain.TransactionExpense && req.ExpenseCategoryID == nil {
		response.BadRequest(c, "Expense category is required for expense transactions", "validation_error")
		return
	}
	if req.Type == domain.TransactionSavings && req.SavingsCategoryID == nil {
		response.BadRequest(c, "Savings category is required for savings transactions", "validation_error")
		return
	}

	req.Source = domain.SourceReceipt

	tx, err := h.transactionUC.Create(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Failed to create transaction")
		return
	}

	// Creating from a receipt also adds a row, moving `max_transactions`.
	emitUsageChanged(c.Request.Context(), h.usageEmitter, userID, "transaction_from_receipt", UsageKeyMaxTransactions)
	response.Created(c, "Transaction created from receipt", tx)
}

// ─── Voice Processing ──────────────────────────────────────────

func (h *AIHandler) ProcessVoice(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		Audio string `json:"audio" binding:"required"`
		// Language is the per-recording language (N-02); empty falls back to the
		// configured default.
		Language string `json:"language"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Audio data is required", err.Error())
		return
	}

	if len(req.Audio) > 5*1024*1024 {
		response.BadRequest(c, "Audio too large. Max 5MB.", "file_too_large")
		return
	}

	lang := strings.TrimSpace(req.Language)
	if lang == "" && h.voiceCfg != nil {
		lang = voice.Load(c.Request.Context(), h.voiceCfg).DefaultLanguage
	}
	if lang == "" {
		lang = voice.DefaultLanguage
	}

	// Estimate the DECODED audio size without allocating a second full buffer:
	// base64 expands 3 bytes into 4 chars, so length*3/4 is the exact decoded
	// size for a well-formed payload (padding only shrinks it). Decoding here
	// would double peak memory for a 5 MB request for no benefit — the usecase
	// decodes it anyway.
	audioBytes := int64(len(req.Audio) * 3 / 4)
	// Charge voice to the ledger (reserve → call → commit/release).
	var reservation *billing.Reservation
	if h.ledger != nil {
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		res, rerr := h.ledger.Reserve(c.Request.Context(), userID, h.voiceEstimate(c.Request.Context(), audioBytes), billing.ModalityVoice, "voice-reserve:"+reqID)
		if rerr != nil {
			if errors.Is(rerr, billing.ErrInsufficientQuota) {
				response.QuotaExceeded(c, "Sisa kuota Anda tidak mencukupi untuk memproses suara ini.")
				return
			}
			response.InternalError(c, "Gagal mengalokasikan kuota")
			return
		}
		reservation = res
	}

	result, usage, err := h.aiUC.ProcessVoice(c.Request.Context(), userID, req.Audio)

	// Settle: charge only a usable transcription; otherwise return the quota.
	if reservation != nil {
		settleVoice(h.ledger, userID, reservation, result, usage, err)
	}

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"error":   "voice_provider_unavailable",
			"message": "Gagal memproses audio. Silakan coba lagi.",
			"data":    result,
		})
		return
	}

	if ok, _ := result["success"].(bool); !ok {
		msg := "Gagal memproses audio. Silakan coba lagi."
		if m, _ := result["message"].(string); m != "" {
			msg = m
		}
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"error":   "voice_provider_unavailable",
			"message": msg,
			"data":    result,
		})
		return
	}

	// Resolve the daily limit so the claim is conditional (never exceed the cap).
	dailyLimit := int64(0)
	if h.entResolver != nil {
		if res, rerr := h.entResolver.Resolve(c.Request.Context(), userID); rerr == nil {
			dailyLimit = res.Limit("voice_requests_per_day")
		}
	}

	// Record the server-mode transcription (durable history) AND claim the daily
	// slot BEFORE announcing usage. Ordering matters: the client refetches the
	// voice config on `usage_changed`, so the counter must already be
	// incremented or the UI shows a stale daily count (off-by-one).
	h.recordVoiceHistory(c.Request.Context(), userID, result, lang, int64(usage.TotalTokens), dailyLimit)

	// Announce the consumption (tokens + request + daily cap) for live quota bars.
	emitUsageChanged(c.Request.Context(), h.usageEmitter, userID, "ai_voice",
		UsageKeyAITokens, UsageKeyAIRequests, UsageKeyVoicePerDay)

	response.Created(c, "Voice processed", result)
}

// recordVoiceHistory stores a successful server-mode transcription and claims a
// daily slot. Best-effort: failures are logged, never surfaced (the
// transcription already succeeded).
func (h *AIHandler) recordVoiceHistory(ctx context.Context, userID uuid.UUID, result map[string]interface{}, lang string, tokens int64, dailyLimit int64) {
	if h.voiceRepo == nil || result == nil {
		return
	}
	ok, _ := result["success"].(bool)
	if !ok {
		return
	}
	transcript, _ := result["transcript"].(string)
	if strings.TrimSpace(transcript) == "" {
		return
	}

	// Claim the daily slot FIRST (atomic; closes the check-then-increment race).
	// A non-positive limit is unlimited / not-in-plan and bypasses the claim.
	if dailyLimit > 0 && h.voiceDaily != nil {
		if _, _, err := h.voiceDaily(ctx, userID, dailyLimit); err != nil {
			logger.Warn("failed to claim voice daily slot", zap.Error(err), zap.String("user_id", userID.String()))
		}
	}

	entry := &domain.VoiceTranscription{
		UserID:     userID,
		Transcript: transcript,
		Language:   lang,
		Mode:       domain.VoiceModeServer,
		TokensUsed: tokens,
	}
	if err := h.voiceRepo.Create(ctx, entry); err != nil {
		logger.Warn("failed to record voice history", zap.Error(err), zap.String("user_id", userID.String()))
	}
}

// settleVoice commits the real audio token usage on success and releases the
// reservation otherwise, so a failed transcription never costs the user quota.
func settleVoice(
	ledger *billing.LedgerService,
	userID uuid.UUID,
	reservation *billing.Reservation,
	result map[string]interface{},
	usage domain.TokenUsage,
	callErr error,
) {
	ctx := context.Background()
	if callErr == nil && resultSucceeded(result) {
		actual := int64(usage.TotalTokens)
		if actual <= 0 && reservation != nil {
			actual = reservation.RawTokens
			if actual <= 0 {
				actual = reservation.Amount
			}
		}
		if actual <= 0 {
			actual = 1
		}
		if err := ledger.Commit(ctx, userID, reservation.ID, actual, "voice-commit:"+reservation.ID.String()); err != nil {
			logger.Error("voice commit failed", zap.Error(err), zap.String("user_id", userID.String()), zap.String("reservation_id", reservation.ID.String()))
		}
		return
	}
	if err := ledger.Release(ctx, userID, reservation.ID, "voice-release:"+reservation.ID.String()); err != nil {
		logger.Error("voice release failed", zap.Error(err), zap.String("user_id", userID.String()), zap.String("reservation_id", reservation.ID.String()))
	}
}
