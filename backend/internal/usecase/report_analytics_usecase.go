package usecase

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// Analytics computes the deep-dive payload for the report tabs: period-over-
// period deltas (MoM/QoQ/YoY), biggest category movers, a daily series for the
// heatmap, statistical anomalies, derived ratios, and the best/worst month.
//
// It is a method on ReportUsecase so it shares the transaction repository and
// the same period-parsing rules as the base report.
func (u *ReportUsecase) Analytics(ctx context.Context, userID uuid.UUID, req domain.ReportRequest) (*domain.ReportAnalytics, error) {
	now := time.Now()

	start, end, _, _ := resolveReportRange(now, req)

	// Ratios + best/worst need a multi-month view; use the year to date so the
	// runway and the best/worst month are meaningful even for a one-month report.
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	comparison := u.periodComparison(ctx, userID, now, req, start, end)

	// Top movers: current vs previous period, by category.
	currentCats, _ := u.getCategoriesByType(ctx, userID, req.Type, start, end)
	prevStart, prevEnd := previousWindow(start, end)
	prevCats, _ := u.getCategoriesByType(ctx, userID, req.Type, prevStart, prevEnd)
	topMovers := domain.ComputeTopMovers(toCategoryMap(currentCats), toCategoryMap(prevCats), 5)

	daily := u.dailySeries(ctx, userID, start, end)

	// Anomalies need per-month history per category.
	series := u.categoryMonthlySeries(ctx, userID, yearStart, end)
	anomalies := domain.DetectAnomalies(series, 2.0)

	ratios := u.ratios(ctx, userID, start, end)
	bestWorst := u.bestWorst(ctx, userID, yearStart, end)

	return &domain.ReportAnalytics{
		Comparison: comparison,
		TopMovers:  topMovers,
		Daily:      daily,
		Anomalies:  anomalies,
		Ratios:     ratios,
		BestWorst:  bestWorst,
	}, nil
}

// periodComparison builds MoM, QoQ, and YoY deltas anchored at the report end
// date, each against the same-length window immediately before it.
func (u *ReportUsecase) periodComparison(ctx context.Context, userID uuid.UUID, now time.Time, req domain.ReportRequest, start, end time.Time) domain.PeriodComparison {
	build := func(label string, offsetMonths int) domain.PeriodComparisonSet {
		// The comparison window is the same span shifted back by offsetMonths.
		curStart := start
		curEnd := end
		prevStart := curStart.AddDate(0, -offsetMonths, 0)
		prevEnd := curEnd.AddDate(0, -offsetMonths, 0)

		cur := u.safeStats(ctx, userID, curStart, curEnd)
		prev := u.safeStats(ctx, userID, prevStart, prevEnd)

		return domain.PeriodComparisonSet{
			Label:   label,
			Start:   prevStart.Format("2006-01-02"),
			End:     prevEnd.Format("2006-01-02"),
			Income:  domain.NewPeriodDelta(cur.TotalIncome, prev.TotalIncome),
			Expense: domain.NewPeriodDelta(cur.TotalExpense, prev.TotalExpense),
			Savings: domain.NewPeriodDelta(cur.TotalSavings, prev.TotalSavings),
			Surplus: domain.NewPeriodDelta(cur.TotalIncome-cur.TotalExpense, prev.TotalIncome-prev.TotalExpense),
		}
	}
	return domain.PeriodComparison{
		MoM: build("Bulan lalu", 1),
		QoQ: build("Kuartal lalu", 3),
		YoY: build("Tahun lalu", 12),
	}
}

// safeStats returns stats or a zero struct on error (analytics must not fail
// the whole page because one comparison window is unavailable).
func (u *ReportUsecase) safeStats(ctx context.Context, userID uuid.UUID, start, end time.Time) domain.TransactionStats {
	stats, err := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: start, EndDate: end})
	if err != nil || stats == nil {
		return domain.TransactionStats{}
	}
	return *stats
}

// dailySeries returns per-day totals across the period, zero-filled so the
// heatmap has a cell for every day.
func (u *ReportUsecase) dailySeries(ctx context.Context, userID uuid.UUID, start, end time.Time) []domain.DailyPoint {
	points, _ := u.txRepo.GetTimeSeries(ctx, userID, start, end, "day")
	byDate := make(map[string]domain.TimeSeriesPoint, len(points))
	for _, p := range points {
		byDate[p.Period] = p
	}

	out := []domain.DailyPoint{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		p := byDate[key]
		out = append(out, domain.DailyPoint{
			Date:    key,
			Income:  math.Round(p.Income*100) / 100,
			Expense: math.Round(p.Expense*100) / 100,
			Savings: math.Round(p.Savings*100) / 100,
			Net:     math.Round((p.Income-p.Expense)*100) / 100,
		})
	}
	return out
}

// categoryMonthlySeries returns each category's month-by-month totals, oldest
// first, for anomaly detection.
func (u *ReportUsecase) categoryMonthlySeries(ctx context.Context, userID uuid.UUID, start, end time.Time) []domain.CategoryMonthlySeries {
	rows, _ := u.txRepo.GetCategoryTimeSeries(ctx, userID, start, end)

	// Build the ordered month buckets first so every category aligns.
	months := []string{}
	seen := map[string]bool{}
	for d := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location()); !d.After(end); d = d.AddDate(0, 1, 0) {
		key := d.Format("2006-01")
		if !seen[key] {
			seen[key] = true
			months = append(months, key)
		}
	}

	perCategory := map[string]map[string]float64{}
	order := []string{}
	for _, r := range rows {
		if _, ok := perCategory[r.CategoryName]; !ok {
			perCategory[r.CategoryName] = map[string]float64{}
			order = append(order, r.CategoryName)
		}
		perCategory[r.CategoryName][r.Period] = r.Total
	}

	out := make([]domain.CategoryMonthlySeries, 0, len(order))
	for _, name := range order {
		totals := make([]float64, len(months))
		for i, m := range months {
			totals[i] = perCategory[name][m]
		}
		out = append(out, domain.CategoryMonthlySeries{CategoryName: name, Totals: totals})
	}
	return out
}

// ratios derives savings rate, expense ratio, and runway from the period stats.
func (u *ReportUsecase) ratios(ctx context.Context, userID uuid.UUID, start, end time.Time) domain.FinancialRatios {
	stats := u.safeStats(ctx, userID, start, end)

	months := monthsBetween(start, end)
	if months < 1 {
		months = 1
	}

	var savingsRate, expenseRatio, runway float64
	if stats.TotalIncome > 0 {
		savingsRate = math.Round(((stats.TotalIncome-stats.TotalExpense)/stats.TotalIncome)*10000) / 100
		expenseRatio = math.Round((stats.TotalExpense/stats.TotalIncome)*10000) / 100
	}
	avgMonthlyExpense := stats.TotalExpense / float64(months)
	if avgMonthlyExpense > 0 {
		runway = math.Round((stats.TotalSavings/avgMonthlyExpense)*100) / 100
		// A runway above 10 years is not actionable; cap it so the UI stays sane.
		if runway > 120 {
			runway = 120
		}
	}

	return domain.FinancialRatios{
		SavingsRate:   savingsRate,
		ExpenseRatio:  expenseRatio,
		RunwayMonths:  runway,
		MonthsSpanned: months,
	}
}

// bestWorst finds the months with the highest and lowest surplus over a range.
func (u *ReportUsecase) bestWorst(ctx context.Context, userID uuid.UUID, start, end time.Time) domain.BestWorstPeriod {
	points, _ := u.txRepo.GetTimeSeries(ctx, userID, start, end, "month")
	if len(points) == 0 {
		return domain.BestWorstPeriod{}
	}
	best := points[0]
	worst := points[0]
	for _, p := range points {
		surplus := p.Income - p.Expense
		if surplus > best.Income-best.Expense {
			best = p
		}
		if surplus < worst.Income-worst.Expense {
			worst = p
		}
	}
	return domain.BestWorstPeriod{
		BestMonth:  best.Period,
		BestValue:  math.Round((best.Income-best.Expense)*100) / 100,
		WorstMonth: worst.Period,
		WorstValue: math.Round((worst.Income-worst.Expense)*100) / 100,
	}
}

// ─── helpers ──────────────────────────────────────────────────

// resolveReportRange centralises the period/range parsing used by Generate and
// Analytics so both agree on the window.
func resolveReportRange(now time.Time, req domain.ReportRequest) (start, end, prevStart, prevEnd time.Time) {
	if req.StartDate != "" && req.EndDate != "" {
		s, err1 := time.Parse("2006-01-02", req.StartDate)
		e, err2 := time.Parse("2006-01-02", req.EndDate)
		if err1 == nil && err2 == nil {
			if req.CompareStart != "" && req.CompareEnd != "" {
				ps, _ := time.Parse("2006-01-02", req.CompareStart)
				pe, _ := time.Parse("2006-01-02", req.CompareEnd)
				return s, e, ps, pe
			}
			ps, pe := previousWindow(s, e)
			return s, e, ps, pe
		}
	}
	start, end = domain.ParseReportPeriod(now, req.Period)
	prevStart, prevEnd = domain.ParsePrevPeriod(now, req.Period)
	return start, end, prevStart, prevEnd
}

// previousWindow returns the equal-length window immediately before [start,end].
func previousWindow(start, end time.Time) (time.Time, time.Time) {
	span := end.Sub(start)
	prevEnd := start.AddDate(0, 0, -1)
	prevStart := prevEnd.Add(-span)
	return prevStart, prevEnd
}

// monthsBetween counts the calendar months touched by [start,end], inclusive.
func monthsBetween(start, end time.Time) int {
	if end.Before(start) {
		return 0
	}
	return (end.Year()-start.Year())*12 + int(end.Month()) - int(start.Month()) + 1
}

// toCategoryMap flattens category summaries into name→total.
func toCategoryMap(cats []domain.CategoryExpenseSummary) map[string]float64 {
	out := make(map[string]float64, len(cats))
	for _, c := range cats {
		out[c.CategoryName] = c.Total
	}
	return out
}
