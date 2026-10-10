package domain

import (
	"context"
	"github.com/google/uuid"
)

type DashboardOverview struct {
	CurrentMonth PeriodSummary `json:"current_month"`
	PrevMonth    PeriodSummary `json:"prev_month"`
	Change       ChangeSummary `json:"change"`
	RecentTxns   []Transaction `json:"recent_transactions"`
	LatestSPK    *SPKResult    `json:"latest_spk,omitempty"`
}

type PeriodSummary struct {
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Savings float64 `json:"savings"`
	Balance float64 `json:"balance"`
}

type ChangeSummary struct {
	IncomeChange   float64 `json:"income_change_pct"`
	ExpenseChange  float64 `json:"expense_change_pct"`
	SavingsChange  float64 `json:"savings_change_pct"`
	BalanceChange  float64 `json:"balance_change_pct"`
}

type DateRange struct {
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

type ChartRequest struct {
	Type        string `form:"type"`
	Period      string `form:"period"`
	StartDate   string `form:"start_date"`
	EndDate     string `form:"end_date"`
	Granularity string `form:"granularity"`
}

type OverviewRequest struct {
	StartDate    string `form:"start_date"`
	EndDate      string `form:"end_date"`
	CompareStart string `form:"compare_start"`
	CompareEnd   string `form:"compare_end"`
}

type ChartData struct {
	Labels []string    `json:"labels"`
	Series []ChartSeries `json:"series"`
}

type ChartSeries struct {
	Name   string    `json:"name"`
	Data   []float64 `json:"data"`
	Color  string    `json:"color,omitempty"`
}

type DashboardRepository interface {
	GetOverview(ctx context.Context, userID uuid.UUID) (*DashboardOverview, error)
	GetChartData(ctx context.Context, userID uuid.UUID, chartType, period string) (*ChartData, error)
}

type DashboardUsecase interface {
	GetOverview(ctx context.Context, userID uuid.UUID, req OverviewRequest) (*DashboardOverview, error)
	GetChartData(ctx context.Context, userID uuid.UUID, req ChartRequest) (*ChartData, error)
}
