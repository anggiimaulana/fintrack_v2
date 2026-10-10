package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SPKStatus string

const (
	SPKVeryGood      SPKStatus = "Sangat Baik"
	SPKGood          SPKStatus = "Baik"
	SPKFair          SPKStatus = "Cukup"
	SPKWasteful      SPKStatus = "Boros"
	SPKCritical      SPKStatus = "Kritis"
)

type SPKResult struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	UserID       uuid.UUID              `json:"user_id" db:"user_id"`
	Score        float64                `json:"score" db:"score"`
	Status       SPKStatus              `json:"status" db:"status"`
	Breakdown    map[string]float64     `json:"breakdown,omitempty" db:"breakdown"`
	PeriodStart  *time.Time             `json:"period_start,omitempty" db:"period_start"`
	PeriodEnd    *time.Time             `json:"period_end,omitempty" db:"period_end"`
	CalculatedAt time.Time              `json:"calculated_at" db:"calculated_at"`
}

type SPKRepository interface {
	Create(ctx context.Context, result *SPKResult) error
	GetLatest(ctx context.Context, userID uuid.UUID) (*SPKResult, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]SPKResult, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type SPKIndicator struct {
	SavingRatio       float64
	NeedsVsWantsRatio float64
	Consistency       float64
	ExpenseTrend      float64
	Diversification   float64
}

const (
	WeightSavingRatio       = 0.30
	WeightNeedsVsWants      = 0.25
	WeightConsistency       = 0.15
	WeightExpenseTrend      = 0.20
	WeightDiversification   = 0.10
)
