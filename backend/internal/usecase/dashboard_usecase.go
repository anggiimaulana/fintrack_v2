package usecase

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/pkg/logger"
)

type DashboardUsecase struct {
	txRepo  domain.TransactionRepository
	spkUsec *SPKUsecase
}

func NewDashboardUsecase(txRepo domain.TransactionRepository, spkUsecase *SPKUsecase) *DashboardUsecase {
	return &DashboardUsecase{
		txRepo:  txRepo,
		spkUsec: spkUsecase,
	}
}

func (u *DashboardUsecase) GetOverview(ctx context.Context, userID uuid.UUID, req domain.OverviewRequest) (*domain.DashboardOverview, error) {
	now := time.Now()

	var currentStart, currentEnd, prevStart, prevEnd time.Time

	if req.StartDate != "" && req.EndDate != "" {
		var err error
		currentStart, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			logger.Error("invalid start_date", zap.Error(err))
			currentStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		}
		currentEnd, err = time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			logger.Error("invalid end_date", zap.Error(err))
			currentEnd = now
		}

		if req.CompareStart != "" && req.CompareEnd != "" {
			prevStart, _ = time.Parse("2006-01-02", req.CompareStart)
			prevEnd, _ = time.Parse("2006-01-02", req.CompareEnd)
		} else {
			// Default comparison: previous same-length period
			diff := currentEnd.Sub(currentStart)
			prevEnd = currentStart.AddDate(0, 0, -1)
			prevStart = prevEnd.Add(-diff)
		}
	} else {
		currentYear, currentMonth, _ := now.Date()
		prevYear, prevMonth, _ := now.AddDate(0, -1, 0).Date()

		currentStart = time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, now.Location())
		currentEnd = now
		prevStart = time.Date(prevYear, prevMonth, 1, 0, 0, 0, 0, now.Location())
		prevEnd = currentStart.AddDate(0, 0, -1)
	}

	currentStats, err := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{
		StartDate: currentStart,
		EndDate:   currentEnd,
	})
	if err != nil {
		return nil, err
	}

	// prevStats and recent transactions are independent — fetch in parallel.
	var (
		prevStats  *domain.TransactionStats
		recentTxns []domain.Transaction
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var e error
		prevStats, e = u.txRepo.GetStats(gctx, userID, domain.StatsPeriod{
			StartDate: prevStart,
			EndDate:   prevEnd,
		})
		return e
	})
	g.Go(func() error {
		var e error
		recentTxns, e = u.txRepo.GetRecentByUser(gctx, userID, 5)
		return e
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	latestSPK, _ := u.spkUsec.GetLatest(ctx, userID)

	overview := &domain.DashboardOverview{
		CurrentMonth: domain.PeriodSummary{
			Income:  math.Round(currentStats.TotalIncome*100) / 100,
			Expense: math.Round(currentStats.TotalExpense*100) / 100,
			Savings: math.Round(currentStats.TotalSavings*100) / 100,
			Balance: math.Round((currentStats.TotalIncome-currentStats.TotalExpense)*100) / 100,
		},
		PrevMonth: domain.PeriodSummary{
			Income:  math.Round(prevStats.TotalIncome*100) / 100,
			Expense: math.Round(prevStats.TotalExpense*100) / 100,
			Savings: math.Round(prevStats.TotalSavings*100) / 100,
			Balance: math.Round((prevStats.TotalIncome-prevStats.TotalExpense)*100) / 100,
		},
		RecentTxns: recentTxns,
		LatestSPK:  latestSPK,
	}

	overview.Change = calculateChange(currentStats, prevStats)

	return overview, nil
}

func calculateChange(current, prev *domain.TransactionStats) domain.ChangeSummary {
	change := domain.ChangeSummary{}

	if prev.TotalIncome > 0 {
		change.IncomeChange = math.Round(((current.TotalIncome-prev.TotalIncome)/prev.TotalIncome)*10000) / 100
	}
	if prev.TotalExpense > 0 {
		change.ExpenseChange = math.Round(((current.TotalExpense-prev.TotalExpense)/prev.TotalExpense)*10000) / 100
	}
	if prev.TotalSavings > 0 {
		change.SavingsChange = math.Round(((current.TotalSavings-prev.TotalSavings)/prev.TotalSavings)*10000) / 100
	}

	prevBalance := prev.TotalIncome - prev.TotalExpense
	if prevBalance > 0 {
		currentBalance := current.TotalIncome - current.TotalExpense
		change.BalanceChange = math.Round(((currentBalance-prevBalance)/prevBalance)*10000) / 100
	}

	return change
}

func (u *DashboardUsecase) GetChartData(ctx context.Context, userID uuid.UUID, req domain.ChartRequest) (*domain.ChartData, error) {
	now := time.Now()
	var labels []string
	var incomeData []float64
	var expenseData []float64
	var savingsData []float64

	if req.StartDate != "" && req.EndDate != "" {
		start, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			logger.Error("invalid chart start_date", zap.Error(err))
			start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		}
		end, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			logger.Error("invalid chart end_date", zap.Error(err))
			end = now
		}

		granularity := req.Granularity
		if granularity == "" {
			days := end.Sub(start).Hours() / 24
			switch {
			case days <= 35:
				granularity = "day"
			case days <= 95:
				granularity = "week"
			default:
				granularity = "month"
			}
		}

		points, err := u.txRepo.GetTimeSeries(ctx, userID, start, end, granularity)
		if err != nil {
			logger.Error("failed to get time series", zap.Error(err))
			return &domain.ChartData{Labels: []string{}, Series: []domain.ChartSeries{}}, nil
		}

		for _, p := range points {
			labels = append(labels, p.Period)
			incomeData = append(incomeData, p.Income)
			expenseData = append(expenseData, p.Expense)
			savingsData = append(savingsData, p.Savings)
		}

		if len(labels) == 0 {
			labels = append(labels, "Tidak ada data")
		}
	} else {
		// Compute date range and granularity based on period
		var periodStart time.Time
		var granularity string

		switch req.Period {
		case "week":
			// Current week (Monday to now)
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			periodStart = time.Date(now.Year(), now.Month(), now.Day()-(weekday-1), 0, 0, 0, 0, now.Location())
			granularity = "day"
		case "month":
			periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			granularity = "day"
		case "3month":
			periodStart = time.Date(now.Year(), now.Month()-2, 1, 0, 0, 0, 0, now.Location())
			granularity = "week"
		case "6month":
			periodStart = time.Date(now.Year(), now.Month()-5, 1, 0, 0, 0, 0, now.Location())
			granularity = "month"
		case "year":
			periodStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
			granularity = "month"
		default:
			periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			granularity = "day"
		}

		points, err := u.txRepo.GetTimeSeries(ctx, userID, periodStart, now, granularity)
		if err != nil {
			logger.Error("failed to get time series for period", zap.Error(err))
			return &domain.ChartData{Labels: []string{}, Series: []domain.ChartSeries{}}, nil
		}

		for _, p := range points {
			labels = append(labels, p.Period)
			incomeData = append(incomeData, p.Income)
			expenseData = append(expenseData, p.Expense)
			savingsData = append(savingsData, p.Savings)
		}

		if len(labels) == 0 {
			labels = append(labels, "Tidak ada data")
		}
	}

	return &domain.ChartData{
		Labels: labels,
		Series: []domain.ChartSeries{
			{Name: "Pemasukan", Data: incomeData, Color: "#10b981"},
			{Name: "Pengeluaran", Data: expenseData, Color: "#f43f5e"},
			{Name: "Tabungan", Data: savingsData, Color: "#f59e0b"},
		},
	}, nil
}

func (u *DashboardUsecase) GetCategoryChartData(ctx context.Context, userID uuid.UUID, req domain.ChartRequest) (*domain.ChartData, error) {
	now := time.Now()
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		end = now
	}

	rows, err := u.txRepo.GetCategoryTimeSeries(ctx, userID, start, end)
	if err != nil {
		return &domain.ChartData{Labels: []string{}, Series: []domain.ChartSeries{}}, nil
	}

	// Collect all unique periods and categories
	periodSet := make(map[string]bool)
	catMap := make(map[string]map[string]float64) // category -> period -> total
	var catOrder []string

	for _, row := range rows {
		periodSet[row.Period] = true
		if _, ok := catMap[row.CategoryName]; !ok {
			catMap[row.CategoryName] = make(map[string]float64)
			catOrder = append(catOrder, row.CategoryName)
		}
		catMap[row.CategoryName][row.Period] = row.Total
	}

	// Sort periods chronologically
	var labels []string
	for p := range periodSet {
		labels = append(labels, p)
	}
	sort.Strings(labels)

	if len(labels) == 0 {
		labels = append(labels, "Tidak ada data")
	}

	catColors := []string{"#3b5bdb", "#10b981", "#f59e0b", "#f43f5e", "#06b6d4", "#8b5cf6", "#ec4899", "#14b8a6", "#eab308", "#a855f7"}
	var series []domain.ChartSeries
	for i, cat := range catOrder {
		var data []float64
		for _, lbl := range labels {
			data = append(data, catMap[cat][lbl])
		}
		series = append(series, domain.ChartSeries{
			Name:  cat,
			Data:  data,
			Color: catColors[i%len(catColors)],
		})
	}

	return &domain.ChartData{Labels: labels, Series: series}, nil
}

func (u *DashboardUsecase) getMonthlyTotal(ctx context.Context, userID uuid.UUID, year, month int, txType string) (float64, error) {
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
