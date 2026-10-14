package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
	"github.com/fintrack/backend/pkg/validator"
)

type TransactionHandler struct {
	usecase *usecase.TransactionUsecase
}

func NewTransactionHandler(usecase *usecase.TransactionUsecase) *TransactionHandler {
	return &TransactionHandler{usecase: usecase}
}

func (h *TransactionHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	filter := domain.TransactionFilter{
		Page:    page,
		PerPage: perPage,
	}

	if typeStr := c.Query("type"); typeStr != "" {
		t := domain.TransactionType(typeStr)
		filter.Type = &t
	}

	if categoryID := c.Query("category_id"); categoryID != "" {
		id, err := uuid.Parse(categoryID)
		if err == nil {
			filter.CategoryID = &id
		}
	}

	if dateFrom := c.Query("date_from"); dateFrom != "" {
		t, err := time.Parse("2006-01-02", dateFrom)
		if err == nil {
			filter.DateFrom = &t
		}
	}

	if dateTo := c.Query("date_to"); dateTo != "" {
		t, err := time.Parse("2006-01-02", dateTo)
		if err == nil {
			filter.DateTo = &t
		}
	}

	if search := c.Query("search"); search != "" {
		filter.Search = validator.SanitizeString(search)
	}

	sortBy := c.Query("sort")
	if sortBy == "" {
		sortBy = c.Query("sortBy")
	}
	sortOrder := c.Query("order")
	if sortOrder == "" {
		sortOrder = c.Query("sortOrder")
	}
	filter.SortBy = sortBy
	filter.SortOrder = sortOrder
	transactions, total, err := h.usecase.List(c.Request.Context(), userID, filter)
	if err != nil {
		response.InternalError(c, "Failed to list transactions")
		return
	}

	if transactions == nil {
		transactions = []domain.Transaction{}
	}

	response.Paginated(c, "Transactions retrieved successfully", transactions, total, page, perPage)
}

func (h *TransactionHandler) GetByID(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid transaction ID", err.Error())
		return
	}

	tx, err := h.usecase.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		response.InternalError(c, "Failed to get transaction")
		return
	}
	if tx == nil {
		response.NotFound(c, "Transaction not found")
		return
	}

	response.OK(c, "Transaction retrieved successfully", tx)
}

func (h *TransactionHandler) Create(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req domain.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
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

	tx, err := h.usecase.Create(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalError(c, "Failed to create transaction")
		return
	}

	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.Created(c, "Transaction created successfully", tx)
}

func (h *TransactionHandler) Update(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid transaction ID", err.Error())
		return
	}

	var req domain.UpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	tx, err := h.usecase.Update(c.Request.Context(), userID, id, req)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			response.Conflict(c, "Data telah berubah, muat ulang")
			return
		}
		response.InternalError(c, "Failed to update transaction")
		return
	}
	if tx == nil {
		response.NotFound(c, "Transaction not found")
		return
	}

	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.OK(c, "Transaction updated successfully", tx)
}

func (h *TransactionHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid transaction ID", err.Error())
		return
	}

	if err := h.usecase.SoftDelete(c.Request.Context(), id, userID); err != nil {
		response.InternalError(c, "Failed to delete transaction")
		return
	}

	middleware.InvalidateUserCache(c.Request.Context(), userID)
	response.OK(c, "Transaction deleted successfully", nil)
}

func (h *TransactionHandler) GetStats(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endDate := now

	if fromStr := c.Query("date_from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			startDate = t
		}
	} else if fromStr := c.Query("start_date"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			startDate = t
		}
	}

	if toStr := c.Query("date_to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			endDate = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
		}
	} else if toStr := c.Query("end_date"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			endDate = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
		}
	}

	period := domain.StatsPeriod{
		StartDate: startDate,
		EndDate:   endDate,
	}
	stats, err := h.usecase.GetStats(c.Request.Context(), userID, period)
	if err != nil {
		response.InternalError(c, "Failed to get transaction stats")
		return
	}

	response.OK(c, "Transaction stats retrieved successfully", stats)
}
