package usecase

import (
	"context"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

type CategoryUsecase struct {
	incomeRepo      domain.IncomeCategoryRepository
	expenseRepo     domain.ExpenseCategoryRepository
	savingsRepo     domain.SavingsCategoryRepository
	subcategoryRepo domain.ExpenseSubcategoryRepository
}

func NewCategoryUsecase(
	incomeRepo domain.IncomeCategoryRepository,
	expenseRepo domain.ExpenseCategoryRepository,
	savingsRepo domain.SavingsCategoryRepository,
	subcategoryRepo domain.ExpenseSubcategoryRepository,
) *CategoryUsecase {
	return &CategoryUsecase{
		incomeRepo:      incomeRepo,
		expenseRepo:     expenseRepo,
		savingsRepo:     savingsRepo,
		subcategoryRepo: subcategoryRepo,
	}
}

func (u *CategoryUsecase) CreateIncome(ctx context.Context, adminID uuid.UUID, req domain.CreateCategoryRequest) (*domain.IncomeCategory, error) {
	c := &domain.IncomeCategory{
		Name:      req.Name,
		Icon:      req.Icon,
		Color:     req.Color,
		Description: req.Description,
		SortOrder: req.SortOrder,
		CreatedBy: &adminID,
	}
	if c.Icon == "" {
		c.Icon = "circle"
	}
	if c.Color == "" {
		c.Color = "#3b5bdb"
	}
	if err := u.incomeRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *CategoryUsecase) CreateExpense(ctx context.Context, adminID uuid.UUID, req domain.CreateCategoryRequest) (*domain.ExpenseCategory, error) {
	c := &domain.ExpenseCategory{
		Name:        req.Name,
		Icon:        req.Icon,
		Color:       req.Color,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		CreatedBy:   &adminID,
	}
	if c.Icon == "" {
		c.Icon = "circle"
	}
	if c.Color == "" {
		c.Color = "#3b5bdb"
	}
	if err := u.expenseRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *CategoryUsecase) CreateSubcategory(ctx context.Context, req domain.CreateSubcategoryRequest) (*domain.ExpenseSubcategory, error) {
	s := &domain.ExpenseSubcategory{
		CategoryID: req.CategoryID,
		Name:       req.Name,
		Icon:       req.Icon,
		Color:      req.Color,
		SortOrder:  req.SortOrder,
	}
	if s.Icon == "" {
		s.Icon = "circle"
	}
	if s.Color == "" {
		s.Color = "#3b5bdb"
	}
	if err := u.subcategoryRepo.Create(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (u *CategoryUsecase) ListIncome(ctx context.Context, includeInactive bool) ([]domain.IncomeCategory, error) {
	return u.incomeRepo.List(ctx, includeInactive)
}

func (u *CategoryUsecase) ListExpense(ctx context.Context, includeInactive bool) ([]domain.ExpenseCategory, error) {
	return u.expenseRepo.List(ctx, includeInactive)
}

func (u *CategoryUsecase) ListSubcategories(ctx context.Context, categoryID uuid.UUID, includeInactive bool) ([]domain.ExpenseSubcategory, error) {
	return u.subcategoryRepo.ListByCategory(ctx, categoryID, includeInactive)
}

func (u *CategoryUsecase) UpdateIncome(ctx context.Context, id uuid.UUID, req domain.UpdateCategoryRequest) (*domain.IncomeCategory, error) {
	c, err := u.incomeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Icon != nil {
		c.Icon = *req.Icon
	}
	if req.Color != nil {
		c.Color = *req.Color
	}
	if req.Description != nil {
		c.Description = req.Description
	}
	if req.SortOrder != nil {
		c.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}
	if err := u.incomeRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *CategoryUsecase) UpdateExpense(ctx context.Context, id uuid.UUID, req domain.UpdateCategoryRequest) (*domain.ExpenseCategory, error) {
	c, err := u.expenseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Icon != nil {
		c.Icon = *req.Icon
	}
	if req.Color != nil {
		c.Color = *req.Color
	}
	if req.Description != nil {
		c.Description = req.Description
	}
	if req.SortOrder != nil {
		c.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}
	if err := u.expenseRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *CategoryUsecase) UpdateSubcategory(ctx context.Context, id uuid.UUID, req domain.UpdateSubcategoryRequest) (*domain.ExpenseSubcategory, error) {
	s, err := u.subcategoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	if req.Name != nil {
		s.Name = *req.Name
	}
	if req.Icon != nil {
		s.Icon = *req.Icon
	}
	if req.Color != nil {
		s.Color = *req.Color
	}
	if req.SortOrder != nil {
		s.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		s.IsActive = *req.IsActive
	}
	if err := u.subcategoryRepo.Update(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (u *CategoryUsecase) DeleteIncome(ctx context.Context, id uuid.UUID) error {
	return u.incomeRepo.SoftDelete(ctx, id)
}

func (u *CategoryUsecase) DeleteExpense(ctx context.Context, id uuid.UUID) error {
	return u.expenseRepo.SoftDelete(ctx, id)
}

func (u *CategoryUsecase) DeleteSubcategory(ctx context.Context, id uuid.UUID) error {
	return u.subcategoryRepo.SoftDelete(ctx, id)
}

func (u *CategoryUsecase) ListSavings(ctx context.Context, includeInactive bool) ([]domain.SavingsCategory, error) {
	return u.savingsRepo.List(ctx, includeInactive)
}

func (u *CategoryUsecase) CreateSavings(ctx context.Context, adminID uuid.UUID, req domain.CreateCategoryRequest) (*domain.SavingsCategory, error) {
	c := &domain.SavingsCategory{
		Name:        req.Name,
		Icon:        req.Icon,
		Color:       req.Color,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		CreatedBy:   &adminID,
	}
	if c.Icon == "" {
		c.Icon = "piggy-bank"
	}
	if c.Color == "" {
		c.Color = "#f59e0b"
	}
	if err := u.savingsRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *CategoryUsecase) UpdateSavings(ctx context.Context, id uuid.UUID, req domain.UpdateCategoryRequest) (*domain.SavingsCategory, error) {
	c, err := u.savingsRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Icon != nil {
		c.Icon = *req.Icon
	}
	if req.Color != nil {
		c.Color = *req.Color
	}
	if req.Description != nil {
		c.Description = req.Description
	}
	if req.SortOrder != nil {
		c.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}
	if err := u.savingsRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *CategoryUsecase) DeleteSavings(ctx context.Context, id uuid.UUID) error {
	return u.savingsRepo.SoftDelete(ctx, id)
}
