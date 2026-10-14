package domain

import (
	"math"
	"testing"
)

func TestMean(t *testing.T) {
	if got := Mean(nil); got != 0 {
		t.Errorf("Mean(nil) = %v, want 0", got)
	}
	if got := Mean([]float64{2, 4, 6}); got != 4 {
		t.Errorf("Mean = %v, want 4", got)
	}
}

func TestStdDev(t *testing.T) {
	if got := StdDev([]float64{5}); got != 0 {
		t.Errorf("StdDev of one value = %v, want 0", got)
	}
	// Population std dev of {2,4,4,4,5,5,7,9} is 2.
	got := StdDev([]float64{2, 4, 4, 4, 5, 5, 7, 9})
	if math.Abs(got-2) > 1e-9 {
		t.Errorf("StdDev = %v, want 2", got)
	}
}

func TestDetectAnomalies_FlagsSpikeAndDrop(t *testing.T) {
	series := []CategoryMonthlySeries{
		// History hovers around 100 (with real variance), then spikes to 500.
		{CategoryName: "Belanja", Totals: []float64{100, 110, 90, 100, 500}},
		// History hovers around 200, then collapses to 10.
		{CategoryName: "Transport", Totals: []float64{200, 210, 190, 200, 10}},
		// Steady with no anomaly.
		{CategoryName: "Utilitas", Totals: []float64{50, 51, 49, 50, 50}},
	}
	got := DetectAnomalies(series, 2.0)
	if len(got) != 2 {
		t.Fatalf("expected 2 anomalies, got %d: %+v", len(got), got)
	}
	// Ordered by |z| descending: the 500 spike is further out than the 10 drop.
	if got[0].CategoryName != "Belanja" || got[0].Direction != "spike" {
		t.Errorf("first anomaly = %+v, want Belanja/spike", got[0])
	}
	if got[1].CategoryName != "Transport" || got[1].Direction != "drop" {
		t.Errorf("second anomaly = %+v, want Transport/drop", got[1])
	}
}

func TestDetectAnomalies_RequiresEnoughHistory(t *testing.T) {
	// Two points is not enough to judge spread.
	got := DetectAnomalies([]CategoryMonthlySeries{
		{CategoryName: "X", Totals: []float64{100, 900}},
	}, 2.0)
	if len(got) != 0 {
		t.Errorf("expected no anomalies with <3 points, got %+v", got)
	}
}

func TestDetectAnomalies_ZeroSpreadIsNotAnomalous(t *testing.T) {
	// A perfectly constant history has zero std dev; division by zero must be
	// avoided and nothing flagged.
	got := DetectAnomalies([]CategoryMonthlySeries{
		{CategoryName: "X", Totals: []float64{100, 100, 100, 100}},
	}, 2.0)
	if len(got) != 0 {
		t.Errorf("expected no anomalies with zero spread, got %+v", got)
	}
}

func TestDetectAnomalies_DefaultThreshold(t *testing.T) {
	// threshold <= 0 must default to 2.0 rather than flagging everything.
	series := []CategoryMonthlySeries{{CategoryName: "X", Totals: []float64{100, 100, 100, 100, 100}}}
	if got := DetectAnomalies(series, 0); len(got) != 0 {
		t.Errorf("expected no anomalies with default threshold, got %+v", got)
	}
}

func TestComputeTopMovers(t *testing.T) {
	current := map[string]float64{
		"Makanan":   1_500_000, // up from 1_000_000
		"Transport": 300_000,   // down from 800_000
		"Baru":      200_000,   // did not exist before
	}
	previous := map[string]float64{
		"Makanan":   1_000_000,
		"Transport": 800_000,
		"Hilang":    500_000, // vanished this period
	}

	got := ComputeTopMovers(current, previous, 5)

	// Up: Makanan (+500k), Baru (+200k).
	if len(got.Up) != 2 || got.Up[0].CategoryName != "Makanan" {
		t.Fatalf("up movers = %+v", got.Up)
	}
	if got.Up[0].Change != 500_000 || got.Up[0].ChangePct != 50 {
		t.Errorf("Makanan change = %+v, want +500000 / +50%%", got.Up[0])
	}
	// Down: Transport (-500k), Hilang (-500k) — equal magnitude, name breaks tie.
	if len(got.Down) != 2 {
		t.Fatalf("down movers = %+v", got.Down)
	}
	if got.Down[0].CategoryName != "Hilang" || got.Down[0].Change != -500_000 {
		t.Errorf("first down mover = %+v, want Hilang/-500000", got.Down[0])
	}
	if got.Down[0].ChangePct != -100 {
		t.Errorf("Hilang change pct = %v, want -100", got.Down[0].ChangePct)
	}
}

func TestComputeTopMovers_Limit(t *testing.T) {
	current := map[string]float64{"a": 10, "b": 20, "c": 30, "d": 40}
	previous := map[string]float64{}
	got := ComputeTopMovers(current, previous, 2)
	if len(got.Up) != 2 {
		t.Fatalf("expected 2 up movers, got %d", len(got.Up))
	}
	// The largest increases first.
	if got.Up[0].CategoryName != "d" || got.Up[1].CategoryName != "c" {
		t.Errorf("up movers not ordered by size: %+v", got.Up)
	}
}

func TestNewPeriodDelta(t *testing.T) {
	d := NewPeriodDelta(150, 100)
	if d.Change != 50 || d.ChangePct != 50 {
		t.Errorf("delta = %+v, want change 50 / 50%%", d)
	}
	// Previous zero → change is the full current, pct is 0 (undefined ratio).
	d = NewPeriodDelta(80, 0)
	if d.Change != 80 || d.ChangePct != 0 {
		t.Errorf("delta from zero = %+v, want change 80 / 0%%", d)
	}
}
