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
}

func NewAIHandler(aiUC *usecase.AIUsecase, transactionUC *usecase.TransactionUsecase) *AIHandler {
	return &AIHandler{aiUC: aiUC, transactionUC: transactionUC}
}

func (h *AIHandler) WithLedger(l *billing.LedgerService) *AIHandler {
	h.ledger = l
	return h
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
		res, err := h.ledger.Reserve(c.Request.Context(), userID, 1, billing.ModalityChat, idempotencyKey)
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

	go h.aiUC.Chat(c.Request.Context(), userID, req, sessionID, source, msgChan)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	var gotContent bool
	defer func() {
		if reservation != nil {
			if gotContent {
				_ = h.ledger.Commit(context.Background(), userID, reservation.ID, 1, "commit:"+reservation.ID.String())
			} else {
				_ = h.ledger.Release(context.Background(), userID, reservation.ID, "release:"+reservation.ID.String())
			}
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

	result, err := h.aiUC.AnalyzeReceipt(c.Request.Context(), userID, imageData, contentType)
	if err != nil {
		logger.Error("receipt analysis failed", zap.Error(err))
		response.InternalError(c, "Gagal menganalisis struk belanja. Silakan coba beberapa saat lagi.")
		return
	}

	response.OK(c, "Receipt analyzed", result)
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

	result, err := h.aiUC.ProcessVoice(c.Request.Context(), userID, req.Audio)
	if err != nil {
		response.InternalError(c, "Failed to process voice")
		return
	}

	response.OK(c, "Voice processed", result)
}
