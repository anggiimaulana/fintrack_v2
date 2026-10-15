package billing

// RoundingMode controls how raw provider tokens are converted to plan-quota
// units. The active mode is operator-configurable (setting
// `quota_rounding_mode`), so this file defines the CAPABILITY while the
// database stores the POLICY.
type RoundingMode string

const (
	// RoundCeil rounds up: never charge less than the real provider cost.
	// This is the conservative default for the business.
	RoundCeil RoundingMode = "ceil"
	// RoundFloor rounds down: friendlier to the user, but a very small request
	// can round to 0 — pair it with a minimum charge (quota_min_charge_units).
	RoundFloor RoundingMode = "floor"
	// RoundRound rounds to nearest.
	RoundRound RoundingMode = "round"
)

// ParseRoundingMode maps a configured string to a RoundingMode, defaulting to
// RoundCeil for anything unrecognised (the safest choice).
func ParseRoundingMode(v string) RoundingMode {
	switch RoundingMode(v) {
	case RoundFloor:
		return RoundFloor
	case RoundRound:
		return RoundRound
	default:
		return RoundCeil
	}
}

// RoundUnits converts raw provider tokens to plan-quota units using the given
// ratio and rounding mode.
//
// Invariants:
//   - ratio < 1 is coerced to 1 (a ratio of 0 would divide by zero);
//   - raw <= 0 yields 0 (nothing to charge);
//   - RoundUnits(n, 1, m) == n for every mode (ratio 1 is the identity).
func RoundUnits(raw, ratio int64, mode RoundingMode) int64 {
	if ratio < 1 {
		ratio = 1
	}
	if raw <= 0 {
		return 0
	}
	switch mode {
	case RoundFloor:
		return raw / ratio
	case RoundRound:
		return (raw + ratio/2) / ratio
	default: // RoundCeil
		return (raw + ratio - 1) / ratio
	}
}

// ApplyMinCharge enforces a floor on the charged units. It exists because
// `floor` rounding can otherwise charge 0 for a tiny request, which would let a
// caller consume AI for free (a quota-drain vector). A minimum of 0 disables it.
func ApplyMinCharge(units, minCharge, rawTokens int64) int64 {
	if rawTokens <= 0 {
		return 0
	}
	if minCharge > 0 && units < minCharge {
		return minCharge
	}
	return units
}
