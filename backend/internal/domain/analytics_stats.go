package domain

import "math"

// ─── Pure statistics for report analytics ─────────────────────
//
// Kept dependency-free and side-effect-free so the maths is unit-testable
// without a database. The usecase supplies the data; these functions decide.

// Mean returns the arithmetic mean, or 0 for an empty slice.
func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// StdDev returns the population standard deviation, or 0 for fewer than two
// values (a single observation has no spread to judge).
func StdDev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := Mean(xs)
	sumSq := 0.0
	for _, x := range xs {
		d := x - m
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(xs)))
}

// CategoryMonthlySeries is one category's per-month totals, oldest first. The
// final element is treated as "current"; the rest form the baseline history.
type CategoryMonthlySeries struct {
	CategoryName string
	Totals       []float64
}

// DetectAnomalies flags categories whose latest monthly total deviates from
// their own history by more than `threshold` standard deviations.
//
// Requires at least three observations (two history points + current) so a
// single prior month cannot masquerade as a stable baseline, and a non-zero
// spread — otherwise every category would look anomalous or none would.
// Results are ordered by absolute z-score, largest first.
func DetectAnomalies(series []CategoryMonthlySeries, threshold float64) []CategoryAnomaly {
	if threshold <= 0 {
		threshold = 2.0
	}
	out := []CategoryAnomaly{}
	for _, s := range series {
		if len(s.Totals) < 3 {
			continue
		}
		history := s.Totals[:len(s.Totals)-1]
		current := s.Totals[len(s.Totals)-1]
		mean := Mean(history)
		sd := StdDev(history)
		if sd == 0 {
			continue
		}
		z := (current - mean) / sd
		if math.Abs(z) < threshold {
			continue
		}
		direction := "spike"
		if z < 0 {
			direction = "drop"
		}
		out = append(out, CategoryAnomaly{
			CategoryName: s.CategoryName,
			Current:      round2(current),
			Mean:         round2(mean),
			StdDev:       round2(sd),
			ZScore:       math.Round(z*100) / 100,
			Direction:    direction,
		})
	}
	// Stable order: strongest deviation first, then by name for determinism.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			a, b := out[j-1], out[j]
			if math.Abs(b.ZScore) > math.Abs(a.ZScore) ||
				(math.Abs(b.ZScore) == math.Abs(a.ZScore) && b.CategoryName < a.CategoryName) {
				out[j-1], out[j] = out[j], out[j-1]
			} else {
				break
			}
		}
	}
	return out
}

// ComputeTopMovers returns the categories with the largest increases ("up") and
// decreases ("down") between two periods. `limit` caps each list.
func ComputeTopMovers(current, previous map[string]float64, limit int) TopMovers {
	if limit <= 0 {
		limit = 5
	}
	all := make([]CategoryMover, 0, len(current))
	seen := map[string]bool{}
	for name, cur := range current {
		seen[name] = true
		prev := previous[name]
		all = append(all, CategoryMover{
			CategoryName: name,
			Current:      round2(cur),
			Previous:     round2(prev),
			Change:       round2(cur - prev),
			ChangePct:    pctChange(cur, prev),
		})
	}
	// Categories that existed before but vanished this period are a full decrease.
	for name, prev := range previous {
		if seen[name] {
			continue
		}
		all = append(all, CategoryMover{
			CategoryName: name,
			Current:      0,
			Previous:     round2(prev),
			Change:       round2(-prev),
			ChangePct:    pctChange(0, prev),
		})
	}

	up := make([]CategoryMover, 0, limit)
	down := make([]CategoryMover, 0, limit)
	// Sort by absolute change, descending, so the biggest moves surface first.
	sortMoversByAbsChange(all)
	for _, m := range all {
		if m.Change > 0 && len(up) < limit {
			up = append(up, m)
		}
	}
	for _, m := range all {
		if m.Change < 0 && len(down) < limit {
			down = append(down, m)
		}
	}
	return TopMovers{Up: up, Down: down}
}

// sortMoversByAbsChange orders movers by |Change| descending, then by name.
func sortMoversByAbsChange(xs []CategoryMover) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0; j-- {
			a, b := xs[j-1], xs[j]
			if math.Abs(b.Change) > math.Abs(a.Change) ||
				(math.Abs(b.Change) == math.Abs(a.Change) && b.CategoryName < a.CategoryName) {
				xs[j-1], xs[j] = xs[j], xs[j-1]
			} else {
				break
			}
		}
	}
}

// NewPeriodDelta builds a delta between two values.
func NewPeriodDelta(current, previous float64) PeriodDelta {
	return PeriodDelta{
		Current:   round2(current),
		Previous:  round2(previous),
		Change:    round2(current - previous),
		ChangePct: pctChange(current, previous),
	}
}

// pctChange returns the percentage change from prev to cur, or 0 when prev is 0
// (an undefined ratio is reported as no change rather than infinity).
func pctChange(cur, prev float64) float64 {
	if prev == 0 {
		return 0
	}
	return math.Round(((cur-prev)/prev)*10000) / 100
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
