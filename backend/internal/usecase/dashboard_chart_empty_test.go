package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// stubTxRepo embeds the repository interface so a test only needs to implement
// the methods it exercises; any other call would panic (none are made here).
type stubTxRepo struct {
	domain.TransactionRepository
	categoryRows []domain.CategoryTimeSeriesRow
	categoryErr  error
	seriesPoints []domain.TimeSeriesPoint
	seriesErr    error
}

func (s stubTxRepo) GetCategoryTimeSeries(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.CategoryTimeSeriesRow, error) {
	return s.categoryRows, s.categoryErr
}

func (s stubTxRepo) GetTimeSeries(context.Context, uuid.UUID, time.Time, time.Time, string) ([]domain.TimeSeriesPoint, error) {
	return s.seriesPoints, s.seriesErr
}

// Regression: an empty result used to return a nil slice, which Go marshals to
// JSON `null`. The frontend then called `series.forEach(...)` on null and threw
// "Cannot read properties of null (reading 'forEach')".
func TestGetCategoryChartDataEmptyMarshalsToArrays(t *testing.T) {
	u := NewDashboardUsecase(stubTxRepo{}, nil)

	got, err := u.GetCategoryChartData(context.Background(), uuid.New(), domain.ChartRequest{
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Labels == nil || got.Series == nil {
		t.Fatalf("labels/series must be non-nil, got labels=%v series=%v", got.Labels, got.Series)
	}

	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(payload), `"series":[]`) {
		t.Errorf("series must serialize as [], got %s", payload)
	}
	if strings.Contains(string(payload), `"series":null`) {
		t.Errorf("series must not serialize as null, got %s", payload)
	}
}

func TestGetChartDataEmptyMarshalsToArrays(t *testing.T) {
	u := NewDashboardUsecase(stubTxRepo{}, nil)

	got, err := u.GetChartData(context.Background(), uuid.New(), domain.ChartRequest{
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(payload), "null") {
		t.Errorf("empty chart must not serialize any field as null, got %s", payload)
	}
}

func TestGetCategoryChartDataKeepsRows(t *testing.T) {
	u := NewDashboardUsecase(stubTxRepo{
		categoryRows: []domain.CategoryTimeSeriesRow{
			{Period: "2026-01", CategoryName: "Makanan", Total: 150000},
			{Period: "2026-02", CategoryName: "Makanan", Total: 200000},
			{Period: "2026-01", CategoryName: "Transport", Total: 50000},
		},
	}, nil)

	got, err := u.GetCategoryChartData(context.Background(), uuid.New(), domain.ChartRequest{
		StartDate: "2026-01-01",
		EndDate:   "2026-02-28",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Labels) != 2 || got.Labels[0] != "2026-01" || got.Labels[1] != "2026-02" {
		t.Fatalf("unexpected labels: %v", got.Labels)
	}
	if len(got.Series) != 2 {
		t.Fatalf("expected 2 series, got %d", len(got.Series))
	}
	for _, s := range got.Series {
		if len(s.Data) != len(got.Labels) {
			t.Errorf("series %q data length %d != labels %d", s.Name, len(s.Data), len(got.Labels))
		}
		if s.Color == "" {
			t.Errorf("series %q missing color", s.Name)
		}
	}
}
