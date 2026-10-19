package usecase

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/currency"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/pkg/logger"
)

type ReportUsecase struct {
	txRepo domain.TransactionRepository
	// currencySvc + displayCurrency convert IDR aggregates to the user's display
	// currency at the report's end date. Optional (nil keeps IDR-only behaviour).
	currencySvc     *currency.Service
	displayCurrency func(ctx context.Context, userID uuid.UUID) string
}

func NewReportUsecase(txRepo domain.TransactionRepository) *ReportUsecase {
	return &ReportUsecase{txRepo: txRepo}
}

// WithCurrency attaches conversion for display-currency reports.
func (u *ReportUsecase) WithCurrency(svc *currency.Service, display func(ctx context.Context, userID uuid.UUID) string) *ReportUsecase {
	u.currencySvc = svc
	u.displayCurrency = display
	return u
}

func (u *ReportUsecase) Generate(ctx context.Context, userID uuid.UUID, req domain.ReportRequest) (*domain.ReportResponse, error) {
	now := time.Now()

	var start, end, prevStart, prevEnd time.Time
	var useCustomRange bool

	if req.StartDate != "" && req.EndDate != "" {
		useCustomRange = true
		var err error
		start, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			logger.Error("invalid report start_date", zap.Error(err))
			start, end = domain.ParseReportPeriod(now, req.Period)
			useCustomRange = false
		}
		end, err = time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			logger.Error("invalid report end_date", zap.Error(err))
			start, end = domain.ParseReportPeriod(now, req.Period)
			useCustomRange = false
		}
	}

	if useCustomRange {
		if req.CompareStart != "" && req.CompareEnd != "" {
			prevStart, _ = time.Parse("2006-01-02", req.CompareStart)
			prevEnd, _ = time.Parse("2006-01-02", req.CompareEnd)
		} else {
			diff := end.Sub(start)
			prevEnd = start.AddDate(0, 0, -1)
			prevStart = prevEnd.Add(-diff)
		}
	} else {
		start, end = domain.ParseReportPeriod(now, req.Period)
		prevStart, prevEnd = domain.ParsePrevPeriod(now, req.Period)
	}

	currentStats, err := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: start, EndDate: end})
	if err != nil {
		return nil, err
	}

	categories, err := u.getCategoriesByType(ctx, userID, req.Type, start, end)
	if err != nil {
		return nil, err
	}

	prevCategories, err := u.getCategoriesByType(ctx, userID, req.Type, prevStart, prevEnd)
	if err != nil {
		return nil, err
	}

	prevMap := make(map[string]float64)
	for _, c := range prevCategories {
		prevMap[c.CategoryName] = c.Total
	}

	totalForType := currentStats.TotalExpense
	if req.Type == "income" {
		totalForType = currentStats.TotalIncome
	} else if req.Type == "savings" {
		totalForType = currentStats.TotalSavings
	}
	var categoryReports []domain.CategoryReport
	for _, c := range categories {
		pct := 0.0
		if totalForType > 0 {
			pct = math.Round((c.Total/totalForType)*10000) / 100
		}
		trend := 0.0
		if prev, ok := prevMap[c.CategoryName]; ok && prev > 0 {
			trend = math.Round(((c.Total-prev)/prev)*10000) / 100
		}
		categoryReports = append(categoryReports, domain.CategoryReport{
			CategoryName: c.CategoryName,
			Total:        math.Round(c.Total*100) / 100,
			Percentage:   pct,
			Trend:        trend,
			Count:        c.Count,
		})
	}
	if categoryReports == nil {
		categoryReports = []domain.CategoryReport{}
	}

	var monthly []domain.MonthlyComparison
	if useCustomRange {
		// Use time series for custom range
		days := end.Sub(start).Hours() / 24
		granularity := "month"
		switch {
		case days <= 35:
			granularity = "day"
		case days <= 95:
			granularity = "week"
		}
		points, _ := u.txRepo.GetTimeSeries(ctx, userID, start, end, granularity)
		for _, p := range points {
			monthly = append(monthly, domain.MonthlyComparison{
				Month:   p.Period,
				Income:  math.Round(p.Income*100) / 100,
				Expense: math.Round(p.Expense*100) / 100,
				Savings: math.Round(p.Savings*100) / 100,
			})
		}
	} else {
		months := 1
		switch req.Period {
		case "last_month":
			months = 1
		case "3months":
			months = 3
		case "year":
			months = 12
		default:
			months = 1
		}

		for i := months - 1; i >= 0; i-- {
			t := now.AddDate(0, -i, 0)
			year, month, _ := t.Date()
			income, _ := u.getMonthlyTotal(ctx, userID, year, int(month), "income")
			expense, _ := u.getMonthlyTotal(ctx, userID, year, int(month), "expense")
			savings, _ := u.getMonthlyTotal(ctx, userID, year, int(month), "savings")
			monthly = append(monthly, domain.MonthlyComparison{
				Month:   t.Format("Jan 2006"),
				Income:  math.Round(income*100) / 100,
				Expense: math.Round(expense*100) / 100,
				Savings: math.Round(savings*100) / 100,
			})
		}
	}
	if monthly == nil {
		monthly = []domain.MonthlyComparison{}
	}

	surplus := math.Round((currentStats.TotalIncome-currentStats.TotalExpense)*100) / 100

	resp := &domain.ReportResponse{
		Summary: domain.ReportSummary{
			TotalIncome:  math.Round(currentStats.TotalIncome*100) / 100,
			TotalExpense: math.Round(currentStats.TotalExpense*100) / 100,
			TotalSavings: math.Round(currentStats.TotalSavings*100) / 100,
			Surplus:      surplus,
		},
		Categories: categoryReports,
		Monthly:    monthly,
	}
	u.applyDisplayCurrency(ctx, userID, end, resp)
	return resp, nil
}

// applyDisplayCurrency converts every IDR aggregate in the response into the
// user's display currency using the rate effective at the report's end date (so
// a historical report is not rewritten by a later rate). When there is no rate
// for that date (e.g. the period predates the rate history) or no display
// currency, the totals stay in base IDR and CurrencyCode is pinned to IDR so the
// frontend labels them correctly instead of assuming the active currency.
func (u *ReportUsecase) applyDisplayCurrency(ctx context.Context, userID uuid.UUID, ref time.Time, resp *domain.ReportResponse) {
	resp.CurrencyCode = currency.BaseCode
	resp.RateUsed = 1

	if u.currencySvc == nil || u.displayCurrency == nil {
		return
	}
	code := u.displayCurrency(ctx, userID)
	if code == "" || code == currency.BaseCode {
		return
	}
	rate, err := u.currencySvc.RateAsOf(ctx, code, ref)
	if err != nil || rate == nil || rate.RateToBase == 0 {
		return
	}
	digits := 2
	if m, merr := u.currencySvc.Meta(ctx, code); merr == nil && m != nil {
		digits = m.DecimalDigits
	}
	conv := func(v float64) float64 { return currency.RoundTo(v/rate.RateToBase, digits) }

	resp.Summary.TotalIncome = conv(resp.Summary.TotalIncome)
	resp.Summary.TotalExpense = conv(resp.Summary.TotalExpense)
	resp.Summary.TotalSavings = conv(resp.Summary.TotalSavings)
	resp.Summary.Surplus = conv(resp.Summary.Surplus)
	for i := range resp.Categories {
		resp.Categories[i].Total = conv(resp.Categories[i].Total)
	}
	for i := range resp.Monthly {
		resp.Monthly[i].Income = conv(resp.Monthly[i].Income)
		resp.Monthly[i].Expense = conv(resp.Monthly[i].Expense)
		resp.Monthly[i].Savings = conv(resp.Monthly[i].Savings)
	}
	resp.CurrencyCode = code
	resp.RateUsed = rate.RateToBase
	eff := rate.EffectiveAt
	resp.RateAt = &eff
}

// CurrencyFormat returns the catalog symbol and decimal digits for a code so
// exports render a dynamically added currency as configured. ok is false when
// the catalog is unavailable or the code is unknown (caller keeps defaults).
func (u *ReportUsecase) CurrencyFormat(ctx context.Context, code string) (string, int, bool) {
	if u.currencySvc == nil || code == "" {
		return "", 0, false
	}
	m, err := u.currencySvc.Meta(ctx, code)
	if err != nil || m == nil {
		return "", 0, false
	}
	return m.Symbol, m.DecimalDigits, true
}

func (u *ReportUsecase) getCategoriesByType(ctx context.Context, userID uuid.UUID, typ string, start, end time.Time) ([]domain.CategoryExpenseSummary, error) {
	switch typ {
	case "income":
		return u.txRepo.GetCategoryIncomesByPeriod(ctx, userID, start, end)
	case "savings":
		return u.txRepo.GetCategorySavingsByPeriod(ctx, userID, start, end)
	case "expense":
		return u.txRepo.GetCategoryExpensesByPeriod(ctx, userID, start, end)
	default:
		return u.txRepo.GetCategoryExpensesByPeriod(ctx, userID, start, end)
	}
}

func (u *ReportUsecase) getMonthlyTotal(ctx context.Context, userID uuid.UUID, year, month int, txType string) (float64, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).AddDate(0, 0, -1)

	stats, err := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: start, EndDate: end})
	if err != nil {
		return 0, err
	}

	if txType == "income" {
		return stats.TotalIncome, nil
	}
	if txType == "savings" {
		return stats.TotalSavings, nil
	}
	return stats.TotalExpense, nil
}
