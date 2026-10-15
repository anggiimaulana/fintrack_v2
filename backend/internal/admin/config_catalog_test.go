package admin

import "testing"

// TestEstimateWarnings covers the heuristic thresholds the estimator surfaces
// so an operator is warned before a configuration goes live.
func TestEstimateWarnings(t *testing.T) {
	cases := []struct {
		name     string
		chats    float64
		margin   int64
		marginPc float64
		ratio    int64
		want     int // number of warnings
	}{
		{"under one chat", 0.5, 100, 50, 5, 1},
		{"few chats", 2, 100, 50, 5, 1},
		{"healthy", 10, 100, 50, 5, 0},
		{"negative margin", 10, -1, -5, 5, 1},
		{"thin margin", 10, 10, 10, 5, 1},
		{"extreme ratio", 10, 100, 50, 50, 1},
		{"multiple issues", 1, -1, -5, 50, 3},
	}
	for _, c := range cases {
		got := estimateWarnings(c.chats, c.margin, c.marginPc, c.ratio)
		if len(got) != c.want {
			t.Errorf("%s: got %d warnings (%v), want %d", c.name, len(got), got, c.want)
		}
	}
}

// TestEnumString proves the JSON-quoted setting default is unwrapped.
func TestEnumString(t *testing.T) {
	cases := map[string]string{
		`"ceil"`:  "ceil",
		`"floor"`: "floor",
		`ceil`:    "ceil",
		``:        "",
	}
	for in, want := range cases {
		if got := enumString([]byte(in)); got != want {
			t.Errorf("enumString(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestRound2 keeps the estimator's float output stable for the UI.
func TestRound2(t *testing.T) {
	cases := map[float64]float64{2.514: 2.51, 2.515: 2.52, 3.0: 3.0, 0.0: 0.0}
	for in, want := range cases {
		if got := round2(in); got != want {
			t.Errorf("round2(%v) = %v, want %v", in, got, want)
		}
	}
}

// TestAvailableResolvers ensures every advertised resolver key is unique — a
// duplicate would make one capability unreachable from the metric editor.
func TestAvailableResolvers(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range availableResolvers() {
		if seen[r.Key] {
			t.Errorf("duplicate resolver key %q", r.Key)
		}
		seen[r.Key] = true
		if r.Label == "" {
			t.Errorf("resolver %q has no label", r.Key)
		}
	}
}
