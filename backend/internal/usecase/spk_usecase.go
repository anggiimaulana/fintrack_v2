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

// SPKUsecase computes the financial health score (SPK) from a user's
// transactions.
//
// The scoring model is data-driven: indicators (name, weight, target, category
// set) and the score→status thresholds come from the resolved metric model —
// the user's personal override when present, otherwise the super-admin default,
// otherwise the compiled fallback. See domain.SPKMetricModel.
type SPKUsecase struct {
	txRepo  domain.TransactionRepository
	spkRepo domain.SPKRepository
	metrics domain.SPKMetricRepository
	tx      domain.TxRunner
}

func NewSPKUsecase(txRepo domain.TransactionRepository, spkRepo domain.SPKRepository, metrics domain.SPKMetricRepository, tx domain.TxRunner) *SPKUsecase {
	return &SPKUsecase{
		txRepo:  txRepo,
		spkRepo: spkRepo,
		metrics: metrics,
		tx:      tx,
	}
}

// ResolveMetricModel returns the model that applies to a user, in precedence
// order: personal override → super-admin default → compiled fallback. A repo
// error is logged and treated as "no row" so scoring degrades gracefully rather
// than failing the request.
func (u *SPKUsecase) ResolveMetricModel(ctx context.Context, userID uuid.UUID) domain.SPKMetricModel {
	if u.metrics != nil {
		if override, err := u.metrics.GetUserOverride(ctx, userID.String()); err == nil && override != nil && len(override.Indicators) > 0 {
			override.Source = "user"
			return *override
		} else if err != nil {
			logger.Warn("spk: resolve user override failed", zap.Error(err))
		}
		if def, err := u.metrics.GetActiveDefault(ctx); err == nil && def != nil && len(def.Indicators) > 0 {
			def.Source = "default"
			return *def
		} else if err != nil {
			logger.Warn("spk: resolve default model failed", zap.Error(err))
		}
	}
	return domain.DefaultSPKMetricModel()
}

// Calculate computes and persists an SPK result. The whole read+insert runs in
// a single transaction guarded by a per-user advisory lock, so concurrent
// mutations cannot interleave or duplicate rows.
func (u *SPKUsecase) Calculate(ctx context.Context, userID uuid.UUID) (*domain.SPKResult, error) {
	var result *domain.SPKResult

	// The metric model is read once, outside the scoring transaction: it is
	// configuration, not user data, so it does not need the row lock and reading
	// it here keeps the locked section short.
	model := u.ResolveMetricModel(ctx, userID)

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

		rawValues := u.rawIndicatorValues(txCtx, userID, model, periodStart, periodEnd, prevPeriodStart)

		// Score each indicator against its target and weight it. The breakdown
		// keeps the *weighted contribution* per indicator key (rounded to 2dp),
		// matching the shape the UI already renders.
		breakdown := map[string]float64{}
		totalScore := 0.0
		for _, ind := range model.Indicators {
			raw := rawValues[ind.Type]
			score := scoreIndicator(ind, raw)
			totalScore += score * ind.Weight
			breakdown[ind.Key] = math.Round(score*100) / 100
		}
		totalScore = math.Max(0, math.Min(100, totalScore))

		result = &domain.SPKResult{
			UserID:      userID,
			Score:       math.Round(totalScore*100) / 100,
			Status:      determineStatus(model.Thresholds, totalScore),
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

// rawIndicatorValues derives every indicator type's raw value in one pass over
// the period's data. Raw values are un-clamped; scoreIndicator applies the
// target and direction. Only the types present in the model are guaranteed to be
// populated, but computing all of them is cheap and keeps the code simple.
func (u *SPKUsecase) rawIndicatorValues(ctx context.Context, userID uuid.UUID, model domain.SPKMetricModel, periodStart, periodEnd, prevPeriodStart time.Time) map[domain.SPKMetricType]float64 {
	out := map[domain.SPKMetricType]float64{
		domain.MetricSavingRatio:     0,
		domain.MetricNeedsRatio:      0,
		domain.MetricConsistency:     0,
		domain.MetricExpenseTrend:    0,
		domain.MetricDiversification: 0,
	}

	stats, err := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: periodStart, EndDate: periodEnd})
	if err != nil {
		logger.Error("failed to get stats for SPK indicators", zap.Error(err))
		return out
	}

	income := stats.TotalIncome
	expense := stats.TotalExpense

	if income > 0 {
		out[domain.MetricSavingRatio] = (income - expense) / income
	}

	// "Needs" categories come from the model so a user can re-classify them.
	needsCategories := u.needsSet(model)
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
		out[domain.MetricNeedsRatio] = totalNeeds / expense
		out[domain.MetricDiversification] = 1 - maxCategoryShare
	}

	activeDays, _ := u.txRepo.GetDailyTransactionCount(ctx, userID, periodStart, periodEnd)
	daysInPeriod := int(periodEnd.Sub(periodStart).Hours()/24) + 1
	if daysInPeriod <= 0 {
		daysInPeriod = 30
	}
	out[domain.MetricConsistency] = float64(activeDays) / float64(daysInPeriod)

	prevStats, err := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{
		StartDate: prevPeriodStart,
		EndDate:   periodStart.AddDate(0, 0, -1),
	})
	if err != nil {
		logger.Error("failed to get prev stats for SPK", zap.Error(err))
	}
	prevExpense := prevStats.TotalExpense
	switch {
	case prevExpense > 0:
		trend := (expense - prevExpense) / prevExpense
		// "Improvement" = how much expense fell; stored as a ratio where 1.0 is
		// "no change or better" and 0 is "doubled".
		out[domain.MetricExpenseTrend] = math.Max(0, 1-trend)
	case expense > 0:
		// No prior period to compare: neutral.
		out[domain.MetricExpenseTrend] = 0.5
	default:
		out[domain.MetricExpenseTrend] = 1
	}

	return out
}

// needsSet builds the set of category names treated as "needs" for the model,
// falling back to the compiled default list when the model omits it.
func (u *SPKUsecase) needsSet(model domain.SPKMetricModel) map[string]bool {
	set := map[string]bool{}
	for _, ind := range model.Indicators {
		if ind.Type != domain.MetricNeedsRatio {
			continue
		}
		for _, c := range ind.Categories {
			set[c] = true
		}
	}
	if len(set) == 0 {
		for _, ind := range domain.DefaultSPKMetricModel().Indicators {
			if ind.Type == domain.MetricNeedsRatio {
				for _, c := range ind.Categories {
					set[c] = true
				}
			}
		}
	}
	return set
}

// scoreIndicator converts a raw value into a 0..100 sub-score against the
// indicator's target and direction.
//
//	higher_better: score = 100 * raw / target   (raw ≥ target ⇒ 100)
//	lower_better:  score = 100 * target / raw   (raw ≤ target ⇒ 100)
//
// A raw value of 0 for lower_better scores 100 (nothing to penalise); for
// higher_better it scores 0.
func scoreIndicator(ind domain.SPKMetricIndicator, raw float64) float64 {
	if ind.Target <= 0 {
		return 0
	}
	if ind.DirectionOrDefault() == "lower_better" {
		// Lower raw is better; at or below target the score is full marks.
		if raw <= 0 || raw <= ind.Target {
			return 100
		}
		return math.Min(100, ind.Target/raw*100)
	}
	if raw <= 0 {
		return 0
	}
	return math.Min(100, raw/ind.Target*100)
}

// determineStatus maps a score to a label using the model's thresholds: the
// label with the highest Min that the score still reaches. When the score is
// below every threshold, the lowest threshold's label applies. An empty list
// falls back to the compiled defaults so a mis-configured model still yields a
// label.
func determineStatus(thresholds []domain.SPKStatusThreshold, score float64) domain.SPKStatus {
	if len(thresholds) == 0 {
		thresholds = domain.DefaultSPKMetricModel().Thresholds
	}
	best := thresholds[0]
	matched := false
	lowest := thresholds[0]
	for _, th := range thresholds {
		if th.Min < lowest.Min {
			lowest = th
		}
		if score >= th.Min && (!matched || th.Min > best.Min) {
			best = th
			matched = true
		}
	}
	if !matched {
		best = lowest
	}
	return domain.SPKStatus(best.Label)
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

// MetricModel exposes the resolved model for a user (used by the UI to render
// the breakdown labels and explain the precedence).
func (u *SPKUsecase) MetricModel(ctx context.Context, userID uuid.UUID) domain.SPKMetricModel {
	return u.ResolveMetricModel(ctx, userID)
}
