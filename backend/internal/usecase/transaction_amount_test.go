package usecase

import (
	"errors"
	"math"
	"testing"

	"github.com/fintrack/backend/internal/domain"
)

func TestGuardBaseOverflow(t *testing.T) {
	cases := []struct {
		name   string
		amount float64
		rate   float64
		bad    bool
	}{
		{"idr", 1_000_000, 1, false},
		{"jpy normal", 10_000, 110, false},
		{"near ceiling", 9e16, 1, false},
		{"overflow", 1e13, 1e6, true},
		{"infinite rate", 1, math.Inf(1), true},
		{"nan amount", math.NaN(), 1, true},
	}
	for _, tc := range cases {
		err := guardBaseOverflow(tc.amount, tc.rate)
		if tc.bad && !errors.Is(err, domain.ErrInvalidAmount) {
			t.Fatalf("%s: expected ErrInvalidAmount, got %v", tc.name, err)
		}
		if !tc.bad && err != nil {
			t.Fatalf("%s: unexpected error %v", tc.name, err)
		}
	}
}
