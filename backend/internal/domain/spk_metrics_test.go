package domain

import (
	"errors"
	"testing"
)

func validModel() SPKMetricModel {
	return SPKMetricModel{
		Indicators: []SPKMetricIndicator{
			{Key: "saving_ratio", Label: "Rasio Tabungan", Weight: 0.5, Target: 0.2, Type: MetricSavingRatio},
			{Key: "consistency", Label: "Konsistensi", Weight: 0.5, Target: 1.0, Type: MetricConsistency},
		},
		Thresholds: []SPKStatusThreshold{
			{Min: 80, Label: "Sangat Baik"},
			{Min: 50, Label: "Baik"},
			{Min: 0, Label: "Kritis"},
		},
	}
}

func TestValidateMetricModel_AcceptsValidModel(t *testing.T) {
	if err := ValidateMetricModel(validModel()); err != nil {
		t.Fatalf("expected valid model, got %v", err)
	}
}

func TestValidateMetricModel_RejectsBadModels(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*SPKMetricModel)
		wantErr error
	}{
		{"no indicators", func(m *SPKMetricModel) { m.Indicators = nil }, ErrSPKNoIndicators},
		{"no thresholds", func(m *SPKMetricModel) { m.Thresholds = nil }, ErrSPKNoThresholds},
		{"weights not summing to 1", func(m *SPKMetricModel) { m.Indicators[0].Weight = 0.1 }, ErrSPKWeightSum},
		{"duplicate key", func(m *SPKMetricModel) { m.Indicators[1].Key = m.Indicators[0].Key }, ErrSPKDuplicateKey},
		{"unknown type", func(m *SPKMetricModel) { m.Indicators[0].Type = "nope" }, ErrSPKInvalidType},
		{"weight out of range", func(m *SPKMetricModel) { m.Indicators[0].Weight = 1.5; m.Indicators[1].Weight = -0.5 }, ErrSPKInvalidWeight},
		{"zero target", func(m *SPKMetricModel) { m.Indicators[0].Target = 0 }, ErrSPKInvalidTarget},
		{"missing label", func(m *SPKMetricModel) { m.Indicators[0].Label = "" }, ErrSPKMissingLabel},
		{"thresholds out of order", func(m *SPKMetricModel) {
			m.Thresholds = []SPKStatusThreshold{{Min: 0, Label: "a"}, {Min: 80, Label: "b"}}
		}, ErrSPKThresholdOrder},
		{"threshold min out of range", func(m *SPKMetricModel) {
			m.Thresholds = []SPKStatusThreshold{{Min: 120, Label: "a"}}
		}, ErrSPKThresholdInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validModel()
			tc.mutate(&m)
			err := ValidateMetricModel(m)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestValidateMetricModel_AcceptsFloatWeightSum(t *testing.T) {
	// 0.30+0.25+0.15+0.20+0.10 does not sum to exactly 1.0 in binary floats;
	// the epsilon must absorb that.
	m := DefaultSPKMetricModel()
	if err := ValidateMetricModel(m); err != nil {
		t.Fatalf("default model must validate, got %v", err)
	}
}

func TestDefaultSPKMetricModel_MatchesHistoricalWeights(t *testing.T) {
	m := DefaultSPKMetricModel()
	want := map[string]float64{
		"saving_ratio":    0.30,
		"needs_vs_wants":  0.25,
		"consistency":     0.15,
		"expense_trend":   0.20,
		"diversification": 0.10,
	}
	if len(m.Indicators) != len(want) {
		t.Fatalf("expected %d indicators, got %d", len(want), len(m.Indicators))
	}
	for _, ind := range m.Indicators {
		if w, ok := want[ind.Key]; !ok {
			t.Fatalf("unexpected indicator key %q", ind.Key)
		} else if ind.Weight != w {
			t.Errorf("weight for %q = %v, want %v", ind.Key, ind.Weight, w)
		}
	}
	// The compiled fallback must be a valid model too.
	if err := ValidateMetricModel(m); err != nil {
		t.Fatalf("default model invalid: %v", err)
	}
}

func TestSPKMetricIndicator_DirectionOrDefault(t *testing.T) {
	if got := (SPKMetricIndicator{}).DirectionOrDefault(); got != "higher_better" {
		t.Errorf("empty direction = %q, want higher_better", got)
	}
	if got := (SPKMetricIndicator{Direction: "lower_better"}).DirectionOrDefault(); got != "lower_better" {
		t.Errorf("lower_better direction = %q, want lower_better", got)
	}
	if got := (SPKMetricIndicator{Direction: "garbage"}).DirectionOrDefault(); got != "higher_better" {
		t.Errorf("garbage direction = %q, want higher_better", got)
	}
}
