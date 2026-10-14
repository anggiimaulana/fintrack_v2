package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
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
}

func NewAIHandler(aiUC *usecase.AIUsecase, transactionUC *usecase.TransactionUsecase) *AIHandler {
	return &AIHandler{aiUC: aiUC, transactionUC: transactionUC}
}

func (h *AIHandler) WithLedger(l *billing.LedgerService) *AIHandler {
	h.ledger = l
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
		res, err := h.ledger.Reserve(c.Request.Context(), userID, h.estimateFor(c.Request.Context(), modality), modality, idempotencyKey)
		if err != nil {
			if errors.Is(err, billing.ErrInsufficientQuota) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"success": false,
					"error":   "insufficient_quota",
					"message": "Sisa kuota token Anda tidak mencukupi untuk melakukan permintaan ini.",
				})
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
	if reservation != nil && reservation.Amount > 0 {
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
			_ = h.ledger.Release(ctx, userID, reservation.ID, "release:"+reservation.ID.String())
		case gotContent:
			_ = h.ledger.Commit(ctx, userID, reservation.ID, actualTokens, "commit:"+reservation.ID.String())
		default:
			// No content and not explicitly flagged: treat as unserved.
			_ = h.ledger.Release(ctx, userID, reservation.ID, "release:"+reservation.ID.String())
		}
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
		res, rerr := h.ledger.Reserve(c.Request.Context(), userID, h.estimateFor(c.Request.Context(), billing.ModalityVision), billing.ModalityVision, "vision-reserve:"+reqID)
		if rerr != nil {
			if errors.Is(rerr, billing.ErrInsufficientQuota) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"success": false,
					"error":   "insufficient_quota",
					"message": "Sisa kuota Anda tidak mencukupi untuk memindai struk ini.",
				})
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
	}

	if err != nil {
		logger.Error("receipt analysis failed", zap.Error(err))
		response.InternalError(c, "Gagal menganalisis struk belanja. Silakan coba beberapa saat lagi.")
		return
	}

	response.OK(c, "Receipt analyzed", result)
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
		if actual <= 0 {
			actual = reservation.Amount
		}
		if actual <= 0 {
			actual = 1
		}
		_ = ledger.Commit(ctx, userID, reservation.ID, actual, "vision-commit:"+reservation.ID.String())
		return
	}
	// Unusable result or error: return the reserved quota untouched.
	_ = ledger.Release(ctx, userID, reservation.ID, "vision-release:"+reservation.ID.String())
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

	response.Created(c, "Transaction created from receipt", tx)
}

// ─── Voice Processing ──────────────────────────────────────────

func (h *AIHandler) ProcessVoice(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		Audio string `json:"audio" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Audio data is required", err.Error())
		return
	}

	if len(req.Audio) > 5*1024*1024 {
		response.BadRequest(c, "Audio too large. Max 5MB.", "file_too_large")
		return
	}

	// Charge voice to the ledger (reserve → call → commit/release).
	var reservation *billing.Reservation
	if h.ledger != nil {
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		res, rerr := h.ledger.Reserve(c.Request.Context(), userID, h.estimateFor(c.Request.Context(), billing.ModalityVoice), billing.ModalityVoice, "voice-reserve:"+reqID)
		if rerr != nil {
			if errors.Is(rerr, billing.ErrInsufficientQuota) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"success": false,
					"error":   "insufficient_quota",
					"message": "Sisa kuota Anda tidak mencukupi untuk memproses suara ini.",
				})
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
		response.InternalError(c, "Failed to process voice")
		return
	}

	response.OK(c, "Voice processed", result)
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
		if actual <= 0 {
			actual = reservation.Amount
		}
		if actual <= 0 {
			actual = 1
		}
		_ = ledger.Commit(ctx, userID, reservation.ID, actual, "voice-commit:"+reservation.ID.String())
		return
	}
	_ = ledger.Release(ctx, userID, reservation.ID, "voice-release:"+reservation.ID.String())
}
