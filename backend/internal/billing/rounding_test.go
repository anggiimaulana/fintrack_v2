package billing

import "testing"

func TestRoundUnits_AllModes(t *testing.T) {
	cases := []struct {
		raw, ratio int64
		mode       RoundingMode
		want       int64
	}{
		// ceil (default)
		{0, 5, RoundCeil, 0},
		{1, 5, RoundCeil, 1},
		{4, 5, RoundCeil, 1},
		{5, 5, RoundCeil, 1},
		{6, 5, RoundCeil, 2},
		{9945, 5, RoundCeil, 1989},
		{100, 1, RoundCeil, 100},
		{100, 0, RoundCeil, 100}, // ratio 0 coerced to 1
		{-5, 5, RoundCeil, 0},
		// floor
		{9945, 5, RoundFloor, 1989},
		{4, 5, RoundFloor, 0}, // can be 0 -> needs min charge
		{9, 5, RoundFloor, 1},
		// round
		{7, 5, RoundRound, 1},
		{8, 5, RoundRound, 2},
		{2, 5, RoundRound, 0},
		{3, 5, RoundRound, 1},
	}
	for _, c := range cases {
		if got := RoundUnits(c.raw, c.ratio, c.mode); got != c.want {
			t.Errorf("RoundUnits(%d,%d,%s) = %d, want %d", c.raw, c.ratio, c.mode, got, c.want)
		}
	}
}

// I-8: ratio 1 is the identity for every mode.
func TestRoundUnits_RatioOneIsIdentity(t *testing.T) {
	for _, mode := range []RoundingMode{RoundCeil, RoundFloor, RoundRound} {
		for _, n := range []int64{0, 1, 999, 9945} {
			if got := RoundUnits(n, 1, mode); got != n {
				t.Errorf("RoundUnits(%d,1,%s) = %d, want %d", n, mode, got, n)
			}
		}
	}
}

func TestApplyMinCharge(t *testing.T) {
	cases := []struct{ units, min, raw, want int64 }{
		{0, 1, 4, 1},   // floor gave 0 -> raised to min
		{0, 0, 4, 0},   // no minimum -> stays 0
		{5, 1, 100, 5}, // already above min
		{0, 1, 0, 0},   // nothing consumed -> no charge even with a minimum
		{2, 5, 100, 5}, // raised to min
	}
	for _, c := range cases {
		if got := ApplyMinCharge(c.units, c.min, c.raw); got != c.want {
			t.Errorf("ApplyMinCharge(%d,%d,%d) = %d, want %d", c.units, c.min, c.raw, got, c.want)
		}
	}
}

func TestParseRoundingMode(t *testing.T) {
	cases := map[string]RoundingMode{
		"ceil": RoundCeil, "floor": RoundFloor, "round": RoundRound,
		"": RoundCeil, "bogus": RoundCeil, "CEIL": RoundCeil,
	}
	for in, want := range cases {
		if got := ParseRoundingMode(in); got != want {
			t.Errorf("ParseRoundingMode(%q) = %s, want %s", in, got, want)
		}
	}
}
