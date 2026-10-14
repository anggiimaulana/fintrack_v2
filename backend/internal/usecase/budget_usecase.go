package usecase

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// BudgetUsecase owns per-category budgets and the actual-vs-budget comparison.
type BudgetUsecase struct {
	budgetRepo domain.BudgetRepository
	txRepo     domain.TransactionRepository
}

func NewBudgetUsecase(budgetRepo domain.BudgetRepository, txRepo domain.TransactionRepository) *BudgetUsecase {
	return &BudgetUsecase{budgetRepo: budgetRepo, txRepo: txRepo}
}

// List returns the user's budgets (without actuals).
func (u *BudgetUsecase) List(ctx context.Context, userID uuid.UUID) ([]domain.Budget, error) {
	return u.budgetRepo.ListByUser(ctx, userID)
}

// Upsert creates or updates a budget.
func (u *BudgetUsecase) Upsert(ctx context.Context, userID uuid.UUID, req domain.UpsertBudgetRequest) (*domain.Budget, error) {
	return u.budgetRepo.Upsert(ctx, userID, req)
}

// Delete removes a budget.
func (u *BudgetUsecase) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return u.budgetRepo.Delete(ctx, id, userID)
}

// Status compares each budget against actual spend in [start,end].
//
// The actuals come from one grouped query (spend per category), so the number of
// queries does not grow with the number of budgets.
func (u *BudgetUsecase) Status(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.BudgetStatus, error) {
	budgets, err := u.budgetRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(budgets) == 0 {
		return []domain.BudgetStatus{}, nil
	}

	summaries, err := u.txRepo.GetCategoryExpensesByPeriod(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}
	spend := make(map[string]float64, len(summaries))
	for _, s := range summaries {
		spend[s.CategoryName] = s.Total
	}

	out := make([]domain.BudgetStatus, 0, len(budgets))
	for _, b := range budgets {
		actual := math.Round(spend[b.CategoryName]*100) / 100
		remaining := math.Round((float64(b.AmountIDR)-actual)*100) / 100
		usedPct := 0.0
		if b.AmountIDR > 0 {
			usedPct = math.Round((actual/float64(b.AmountIDR))*10000) / 100
		}
		out = append(out, domain.BudgetStatus{
			Budget:    b,
			Actual:    actual,
			Remaining: remaining,
			UsedPct:   usedPct,
			Over:      actual > float64(b.AmountIDR),
		})
	}
	return out, nil
}
