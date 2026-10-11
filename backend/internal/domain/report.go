package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ReportSummary struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	TotalSavings float64 `json:"total_savings"`
	Surplus      float64 `json:"surplus"`
}

type CategoryReport struct {
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
	Percentage   float64 `json:"percentage"`
	Trend        float64 `json:"trend"`
	Count        int     `json:"count"`
}

type MonthlyComparison struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Savings float64 `json:"savings"`
}

type ReportResponse struct {
	Summary    ReportSummary       `json:"summary"`
	Categories []CategoryReport    `json:"categories"`
	Monthly    []MonthlyComparison `json:"monthly"`
}

type ReportRequest struct {
	Period       string `form:"period"`
	Type         string `form:"type"`
	StartDate    string `form:"start_date"`
	EndDate      string `form:"end_date"`
	CompareStart string `form:"compare_start"`
	CompareEnd   string `form:"compare_end"`
}

type ReportUsecase interface {
	Generate(ctx context.Context, userID uuid.UUID, req ReportRequest) (*ReportResponse, error)
}

func ParseReportPeriod(now time.Time, period string) (time.Time, time.Time) {
	switch period {
	case "last_month":
		firstOfCurrent := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := firstOfCurrent.AddDate(0, 0, -1)
		start := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, end
	case "3months":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -2, 0)
		return start, now
	case "year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return start, now
	default:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, now
	}
}

func ParsePrevPeriod(now time.Time, period string) (time.Time, time.Time) {
	switch period {
	case "last_month":
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
		endPrev := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
		startPrev := time.Date(endPrev.Year(), endPrev.Month(), 1, 0, 0, 0, 0, now.Location())
		return startPrev, endPrev
	case "3months":
		prevEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
		prevStart := time.Date(prevEnd.Year(), prevEnd.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -2, 0)
		return prevStart, prevEnd
	case "year":
		start := time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year()-1, 12, 31, 0, 0, 0, 0, now.Location())
		return start, end
	default:
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
		start := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, end
	}
}
