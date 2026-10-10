package usecase

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/pkg/logger"
)

type SPKUsecase struct {
	txRepo  domain.TransactionRepository
	spkRepo domain.SPKRepository
	tx      domain.TxRunner
}

func NewSPKUsecase(txRepo domain.TransactionRepository, spkRepo domain.SPKRepository, tx domain.TxRunner) *SPKUsecase {
	return &SPKUsecase{
		txRepo:  txRepo,
		spkRepo: spkRepo,
		tx:      tx,
	}
}

var needsCategories = map[string]bool{
	"Makanan & Minuman":  true,
	"Kesehatan":          true,
	"Utilitas & Tagihan": true,
	"Transportasi":       true,
	"Pendidikan":         true,
	"Tempat Tinggal":     true,
}

// Calculate computes and persists an SPK result. The whole read+insert runs in
// a single transaction guarded by a per-user advisory lock, so concurrent
// mutations cannot interleave or duplicate rows.
func (u *SPKUsecase) Calculate(ctx context.Context, userID uuid.UUID) (*domain.SPKResult, error) {
	var result *domain.SPKResult

	err := u.tx.RunInTx(ctx, func(txCtx context.Context) error {
		if err := u.tx.AdvisoryXactLock(txCtx, "spk:"+userID.String()); err != nil {
			return err
		}

		now := time.Now()
		periodEnd := now
		periodStart := now.AddDate(0, -1, 0)
		prevPeriodStart := now.AddDate(0, -2, 0)

		stats, err := u.txRepo.GetStats(txCtx, userID, domain.StatsPeriod{StartDate: periodStart, EndDate: periodEnd})
		if err != nil {
			logger.Error("failed to get stats for SPK", zap.Error(err))
			return err
		}

		// No transactions: delete any stale result and stop.
		if stats.TotalIncome == 0 && stats.TotalExpense == 0 && stats.TotalSavings == 0 {
			existing, _ := u.spkRepo.GetLatest(txCtx, userID)
			if existing != nil {
				_ = u.spkRepo.Delete(txCtx, existing.ID, userID)
			}
			return nil
		}

		indicators := u.calculateIndicators(txCtx, userID, periodStart, periodEnd, prevPeriodStart)
		totalScore := u.calculateTotalScore(indicators)

		breakdown := map[string]float64{
			"saving_ratio":    math.Round(indicators.SavingRatio*100) / 100,
			"needs_vs_wants":  math.Round(indicators.NeedsVsWantsRatio*100) / 100,
			"consistency":     math.Round(indicators.Consistency*100) / 100,
			"expense_trend":   math.Round(indicators.ExpenseTrend*100) / 100,
			"diversification": math.Round(indicators.Diversification*100) / 100,
		}

		result = &domain.SPKResult{
			UserID:      userID,
			Score:       math.Round(totalScore*100) / 100,
			Status:      u.determineStatus(totalScore),
			Breakdown:   breakdown,
			PeriodStart: &periodStart,
			PeriodEnd:   &periodEnd,
		}

		if err := u.spkRepo.Create(txCtx, result); err != nil {
			logger.Error("failed to save SPK result", zap.Error(err))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (u *SPKUsecase) calculateIndicators(ctx context.Context, userID uuid.UUID, periodStart, periodEnd, prevPeriodStart time.Time) domain.SPKIndicator {
	var indicators domain.SPKIndicator

	stats, err := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: periodStart, EndDate: periodEnd})
	if err != nil {
		logger.Error("failed to get stats for SPK", zap.Error(err))
		return indicators
	}

	prevStats, err := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{
		StartDate: prevPeriodStart,
		EndDate:   periodStart.AddDate(0, 0, -1),
	})
	if err != nil {
		logger.Error("failed to get prev stats for SPK", zap.Error(err))
	}

	income := stats.TotalIncome
	expense := stats.TotalExpense

	if income > 0 {
		savingRatio := (income - expense) / income
		indicators.SavingRatio = math.Min(savingRatio*100/0.2, 100)
	}

	categorySummaries, _ := u.txRepo.GetCategoryExpensesByPeriod(ctx, userID, periodStart, periodEnd)
	totalNeeds := 0.0
	maxCategoryShare := 0.0

	for _, s := range categorySummaries {
		if needsCategories[s.CategoryName] {
			totalNeeds += s.Total
		}
		if expense > 0 {
			if share := s.Total / expense; share > maxCategoryShare {
				maxCategoryShare = share
			}
		}
	}

	if expense > 0 {
		needsRatio := totalNeeds / expense
		indicators.NeedsVsWantsRatio = math.Min(needsRatio*100/0.7, 100)
	}

	activeDays, _ := u.txRepo.GetDailyTransactionCount(ctx, userID, periodStart, periodEnd)
	daysInPeriod := int(periodEnd.Sub(periodStart).Hours()/24) + 1
	if daysInPeriod <= 0 {
		daysInPeriod = 30
	}
	indicators.Consistency = float64(activeDays) / float64(daysInPeriod) * 100

	prevExpense := prevStats.TotalExpense
	if prevExpense > 0 {
		trend := (expense - prevExpense) / prevExpense
		indicators.ExpenseTrend = math.Max(0, math.Min(100, (1-trend)*100))
	} else if expense > 0 {
		indicators.ExpenseTrend = 50
	} else {
		indicators.ExpenseTrend = 100
	}

	indicators.Diversification = (1 - maxCategoryShare) * 100

	return indicators
}

func (u *SPKUsecase) calculateTotalScore(ind domain.SPKIndicator) float64 {
	total := 0.0
	total += ind.SavingRatio * domain.WeightSavingRatio
	total += ind.NeedsVsWantsRatio * domain.WeightNeedsVsWants
	total += ind.Consistency * domain.WeightConsistency
	total += ind.ExpenseTrend * domain.WeightExpenseTrend
	total += ind.Diversification * domain.WeightDiversification
	return math.Max(0, math.Min(100, total))
}

func (u *SPKUsecase) determineStatus(score float64) domain.SPKStatus {
	switch {
	case score >= 80:
		return domain.SPKVeryGood
	case score >= 60:
		return domain.SPKGood
	case score >= 40:
		return domain.SPKFair
	case score >= 20:
		return domain.SPKWasteful
	default:
		return domain.SPKCritical
	}
}

// GetLatest returns the newest SPK result, self-healing by calculating once if
// the user has transactions but no stored result yet.
func (u *SPKUsecase) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.SPKResult, error) {
	existing, err := u.spkRepo.GetLatest(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	hasTx, err := u.txRepo.HasTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !hasTx {
		return nil, nil
	}

	return u.Calculate(ctx, userID)
}

func (u *SPKUsecase) GetHistory(ctx context.Context, userID uuid.UUID, limit int) ([]domain.SPKResult, error) {
	return u.spkRepo.ListByUser(ctx, userID, limit)
}
