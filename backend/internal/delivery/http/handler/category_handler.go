package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

type CategoryHandler struct {
	usecase *usecase.CategoryUsecase
}

func NewCategoryHandler(usecase *usecase.CategoryUsecase) *CategoryHandler {
	return &CategoryHandler{usecase: usecase}
}

func (h *CategoryHandler) ListIncome(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"
	categories, err := h.usecase.ListIncome(c.Request.Context(), includeInactive)
	if err != nil {
		response.InternalError(c, "Failed to list income categories")
		return
	}
	if categories == nil {
		categories = []domain.IncomeCategory{}
	}
	response.OK(c, "Income categories retrieved successfully", categories)
}

func (h *CategoryHandler) ListExpense(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"
	categories, err := h.usecase.ListExpense(c.Request.Context(), includeInactive)
	if err != nil {
		response.InternalError(c, "Failed to list expense categories")
		return
	}
	if categories == nil {
		categories = []domain.ExpenseCategory{}
	}
	response.OK(c, "Expense categories retrieved successfully", categories)
}

func (h *CategoryHandler) ListSubcategories(c *gin.Context) {
	categoryID, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		response.BadRequest(c, "Invalid category ID", err.Error())
		return
	}
	includeInactive := c.Query("include_inactive") == "true"
	subcategories, err := h.usecase.ListSubcategories(c.Request.Context(), categoryID, includeInactive)
	if err != nil {
		response.InternalError(c, "Failed to list subcategories")
		return
	}
	if subcategories == nil {
		subcategories = []domain.ExpenseSubcategory{}
	}
	response.OK(c, "Subcategories retrieved successfully", subcategories)
}

// ListSubcategoriesByQuery serves GET /admin/subcategories?category_id=<uuid>
// (the FE contract) — distinct from the path-param route.
func (h *CategoryHandler) ListSubcategoriesByQuery(c *gin.Context) {
	raw := c.Query("category_id")
	if raw == "" {
		response.BadRequest(c, "category_id is required", "missing_category_id")
		return
	}
	categoryID, err := uuid.Parse(raw)
	if err != nil {
		response.BadRequest(c, "Invalid category ID", err.Error())
		return
	}
	includeInactive := c.Query("include_inactive") == "true"
	subcategories, err := h.usecase.ListSubcategories(c.Request.Context(), categoryID, includeInactive)
	if err != nil {
		response.InternalError(c, "Failed to list subcategories")
		return
	}
	if subcategories == nil {
		subcategories = []domain.ExpenseSubcategory{}
	}
	response.OK(c, "Subcategories retrieved successfully", subcategories)
}

func (h *CategoryHandler) CreateIncome(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)
	var req domain.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	category, err := h.usecase.CreateIncome(c.Request.Context(), adminID, req)
	if err != nil {
		response.InternalError(c, "Failed to create income category")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.Created(c, "Income category created successfully", category)
}

func (h *CategoryHandler) CreateExpense(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)
	var req domain.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	category, err := h.usecase.CreateExpense(c.Request.Context(), adminID, req)
	if err != nil {
		response.InternalError(c, "Failed to create expense category")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.Created(c, "Expense category created successfully", category)
}

func (h *CategoryHandler) CreateSubcategory(c *gin.Context) {
	var req domain.CreateSubcategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	subcategory, err := h.usecase.CreateSubcategory(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, "Failed to create subcategory")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.Created(c, "Subcategory created successfully", subcategory)
}

func (h *CategoryHandler) UpdateIncome(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid category ID", err.Error())
		return
	}
	var req domain.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	category, err := h.usecase.UpdateIncome(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Income category not found")
			return
		}
		response.InternalError(c, "Failed to update income category")
		return
	}
	if category == nil {
		response.NotFound(c, "Income category not found")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Income category updated successfully", category)
}

func (h *CategoryHandler) UpdateExpense(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid category ID", err.Error())
		return
	}
	var req domain.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	category, err := h.usecase.UpdateExpense(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Expense category not found")
			return
		}
		response.InternalError(c, "Failed to update expense category")
		return
	}
	if category == nil {
		response.NotFound(c, "Expense category not found")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Expense category updated successfully", category)
}

func (h *CategoryHandler) UpdateSubcategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid subcategory ID", err.Error())
		return
	}
	var req domain.UpdateSubcategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	subcategory, err := h.usecase.UpdateSubcategory(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Subcategory not found")
			return
		}
		response.InternalError(c, "Failed to update subcategory")
		return
	}
	if subcategory == nil {
		response.NotFound(c, "Subcategory not found")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Subcategory updated successfully", subcategory)
}

func (h *CategoryHandler) DeleteIncome(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid category ID", err.Error())
		return
	}
	if err := h.usecase.DeleteIncome(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Income category not found")
			return
		}
		response.InternalError(c, "Failed to delete income category")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Income category deleted successfully", nil)
}

func (h *CategoryHandler) DeleteExpense(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid category ID", err.Error())
		return
	}
	if err := h.usecase.DeleteExpense(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Expense category not found")
			return
		}
		response.InternalError(c, "Failed to delete expense category")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Expense category deleted successfully", nil)
}

func (h *CategoryHandler) DeleteSubcategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid subcategory ID", err.Error())
		return
	}
	if err := h.usecase.DeleteSubcategory(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Subcategory not found")
			return
		}
		response.InternalError(c, "Failed to delete subcategory")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Subcategory deleted successfully", nil)
}

func (h *CategoryHandler) ListSavings(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"
	categories, err := h.usecase.ListSavings(c.Request.Context(), includeInactive)
	if err != nil {
		response.InternalError(c, "Failed to list savings categories")
		return
	}
	if categories == nil {
		categories = []domain.SavingsCategory{}
	}
	response.OK(c, "Savings categories retrieved successfully", categories)
}

func (h *CategoryHandler) CreateSavings(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)
	var req domain.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	category, err := h.usecase.CreateSavings(c.Request.Context(), adminID, req)
	if err != nil {
		response.InternalError(c, "Failed to create savings category")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.Created(c, "Savings category created successfully", category)
}

func (h *CategoryHandler) UpdateSavings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid category ID", err.Error())
		return
	}
	var req domain.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}
	category, err := h.usecase.UpdateSavings(c.Request.Context(), id, req)
	if err != nil {
		response.InternalError(c, "Failed to update savings category")
		return
	}
	if category == nil {
		response.NotFound(c, "Savings category not found")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Savings category updated successfully", category)
}

func (h *CategoryHandler) DeleteSavings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid category ID", err.Error())
		return
	}
	if err := h.usecase.DeleteSavings(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(c, "Savings category not found")
			return
		}
		response.InternalError(c, "Failed to delete savings category")
		return
	}
	middleware.InvalidateAllCache(c.Request.Context())
	response.OK(c, "Savings category deleted successfully", nil)
}
