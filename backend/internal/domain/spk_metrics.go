package domain

import (
	"context"
	"errors"
)

// SPKMetricType enumerates how a raw indicator value is derived from a user's
// transactions. The set is code-defined because the scoring engine is what
// interprets it; a metric config only *selects* one of these.
type SPKMetricType string

const (
	// MetricSavingRatio = (income - expense) / income.
	MetricSavingRatio SPKMetricType = "saving_ratio"
	// MetricNeedsRatio = share of expense that falls in the configured
	// "needs" categories.
	MetricNeedsRatio SPKMetricType = "needs_ratio"
	// MetricConsistency = active transaction days / days in period.
	MetricConsistency SPKMetricType = "consistency"
	// MetricExpenseTrend = how much expense fell versus the previous period.
	MetricExpenseTrend SPKMetricType = "expense_trend"
	// MetricDiversification = 1 - largest single-category share.
	MetricDiversification SPKMetricType = "diversification"
)

// Valid reports whether t is a known metric type.
func (t SPKMetricType) Valid() bool {
	switch t {
	case MetricSavingRatio, MetricNeedsRatio, MetricConsistency, MetricExpenseTrend, MetricDiversification:
		return true
	default:
		return false
	}
}

// SPKMetricIndicator is one weighted criterion in an SPK model.
type SPKMetricIndicator struct {
	// Key is the stable identifier used in the score breakdown. Unique within a
	// model so the breakdown map cannot silently collide.
	Key string `json:"key"`
	// Label is the human-readable name shown in the UI.
	Label string `json:"label"`
	// Weight is 0..1. All weights in a model must sum to 1.0.
	Weight float64 `json:"weight"`
	// Target is the "ideal" raw value that scores 100. Interpretation depends on
	// Direction: for higher_better, score = raw/target; for lower_better,
	// score = target/raw (capped at 100).
	Target float64 `json:"target"`
	// Type selects the derivation formula.
	Type SPKMetricType `json:"type"`
	// Categories is only used by needs_ratio: the category names counted as
	// "needs". Empty means the compiled default list.
	Categories []string `json:"categories,omitempty"`
	// Direction is "higher_better" (default) or "lower_better".
	Direction string `json:"direction,omitempty"`
}

// DirectionOrDefault normalises the direction, defaulting to higher_better.
func (i SPKMetricIndicator) DirectionOrDefault() string {
	if i.Direction == "lower_better" {
		return "lower_better"
	}
	return "higher_better"
}

// SPKStatusThreshold maps a minimum score to a status label. Thresholds are
// evaluated from the highest Min downwards.
type SPKStatusThreshold struct {
	Min   float64 `json:"min"`
	Label string  `json:"label"`
}

// SPKMetricModel is a complete, resolved SPK configuration: the ordered
// indicators plus the score→status mapping.
type SPKMetricModel struct {
	Indicators []SPKMetricIndicator `json:"indicators"`
	Thresholds []SPKStatusThreshold `json:"status_thresholds"`
	// Source records where the model came from, for the UI to explain the
	// precedence: "user" (personal override) or "default" (super admin).
	Source string `json:"source"`
	// Name is the default model's name (empty for a personal override).
	Name string `json:"name,omitempty"`
}

// SPKMetricRepository persists the default and per-user SPK models.
type SPKMetricRepository interface {
	// GetActiveDefault returns the active super-admin default model, or nil when
	// none has been seeded.
	GetActiveDefault(ctx context.Context) (*SPKMetricModel, error)
	// GetUserOverride returns a user's personal model, or nil when they have not
	// customised their metrics.
	GetUserOverride(ctx context.Context, userID string) (*SPKMetricModel, error)
	// UpsertUserOverride stores (or replaces) a user's personal model.
	UpsertUserOverride(ctx context.Context, userID string, m SPKMetricModel) error
	// DeleteUserOverride reverts a user to the default model.
	DeleteUserOverride(ctx context.Context, userID string) error
	// ReplaceDefault deactivates the current default and inserts a new active
	// one in a single transaction, returning the new model.
	ReplaceDefault(ctx context.Context, name, description string, m SPKMetricModel, updatedBy *string) (*SPKMetricModel, error)
}

// SPK metric validation errors. Exported so the HTTP layer can map them to 400
// rather than 500.
var (
	ErrSPKNoIndicators     = errors.New("model SPK harus memiliki minimal satu indikator")
	ErrSPKWeightSum        = errors.New("total bobot indikator harus 1.0")
	ErrSPKDuplicateKey     = errors.New("kunci indikator harus unik")
	ErrSPKInvalidType      = errors.New("tipe indikator tidak dikenal")
	ErrSPKInvalidWeight    = errors.New("bobot harus berada di antara 0 dan 1")
	ErrSPKInvalidTarget    = errors.New("target harus lebih besar dari 0")
	ErrSPKMissingLabel     = errors.New("label indikator wajib diisi")
	ErrSPKNoThresholds     = errors.New("ambang status wajib diisi")
	ErrSPKThresholdOrder   = errors.New("ambang status harus diurutkan menurun berdasarkan min")
	ErrSPKThresholdInvalid = errors.New("ambang status memiliki nilai min atau label yang tidak valid")
)

// weightEpsilon is the tolerance for the "weights sum to 1.0" check, so a model
// authored as 0.3+0.25+0.15+0.2+0.1 is accepted despite float rounding.
const weightEpsilon = 1e-6

// ValidateMetricModel checks a model before it is persisted. It is pure so the
// admin and user handlers share exactly one rule set (and it is unit-testable).
func ValidateMetricModel(m SPKMetricModel) error {
	if len(m.Indicators) == 0 {
		return ErrSPKNoIndicators
	}
	if len(m.Thresholds) == 0 {
		return ErrSPKNoThresholds
	}

	seen := make(map[string]struct{}, len(m.Indicators))
	sum := 0.0
	for _, ind := range m.Indicators {
		if ind.Key == "" || ind.Label == "" {
			return ErrSPKMissingLabel
		}
		if _, dup := seen[ind.Key]; dup {
			return ErrSPKDuplicateKey
		}
		seen[ind.Key] = struct{}{}
		if !ind.Type.Valid() {
			return ErrSPKInvalidType
		}
		if ind.Weight <= 0 || ind.Weight > 1 {
			return ErrSPKInvalidWeight
		}
		if ind.Target <= 0 {
			return ErrSPKInvalidTarget
		}
		sum += ind.Weight
	}
	if diff := sum - 1.0; diff > weightEpsilon || diff < -weightEpsilon {
		return ErrSPKWeightSum
	}

	prev := 1e18
	for _, th := range m.Thresholds {
		if th.Label == "" || th.Min < 0 || th.Min > 100 {
			return ErrSPKThresholdInvalid
		}
		if th.Min > prev {
			return ErrSPKThresholdOrder
		}
		prev = th.Min
	}
	return nil
}

// DefaultSPKMetricModel returns the compiled fallback model — the same values
// the engine hard-coded before metrics became configurable. It is used when the
// database has no active default (e.g. a fresh DB or a unit test), so scoring
// never depends on a seed having run.
func DefaultSPKMetricModel() SPKMetricModel {
	return SPKMetricModel{
		Source: "default",
		Name:   "Model SPK Default",
		Indicators: []SPKMetricIndicator{
			{Key: "saving_ratio", Label: "Rasio Tabungan", Weight: 0.30, Target: 0.20, Type: MetricSavingRatio},
			{Key: "needs_vs_wants", Label: "Kebutuhan vs Keinginan", Weight: 0.25, Target: 0.70, Type: MetricNeedsRatio, Categories: []string{
				"Makanan & Minuman", "Kesehatan", "Utilitas & Tagihan", "Transportasi", "Pendidikan", "Tempat Tinggal",
			}},
			{Key: "consistency", Label: "Konsistensi Catatan", Weight: 0.15, Target: 1.0, Type: MetricConsistency},
			{Key: "expense_trend", Label: "Tren Pengeluaran", Weight: 0.20, Target: 1.0, Type: MetricExpenseTrend},
			{Key: "diversification", Label: "Diversifikasi", Weight: 0.10, Target: 1.0, Type: MetricDiversification},
		},
		Thresholds: []SPKStatusThreshold{
			{Min: 80, Label: string(SPKVeryGood)},
			{Min: 60, Label: string(SPKGood)},
			{Min: 40, Label: string(SPKFair)},
			{Min: 20, Label: string(SPKWasteful)},
			{Min: 0, Label: string(SPKCritical)},
		},
	}
}
