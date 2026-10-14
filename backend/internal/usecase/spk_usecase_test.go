package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// fakeSPKMetricRepo is an in-memory SPKMetricRepository for resolution tests.
type fakeSPKMetricRepo struct {
	def        *domain.SPKMetricModel
	override   *domain.SPKMetricModel
	defErr     error
	overrideE  error
	upserted   *domain.SPKMetricModel
	upsertUser string
	deleted    string
}

func (f *fakeSPKMetricRepo) GetActiveDefault(context.Context) (*domain.SPKMetricModel, error) {
	return f.def, f.defErr
}
func (f *fakeSPKMetricRepo) GetUserOverride(context.Context, string) (*domain.SPKMetricModel, error) {
	return f.override, f.overrideE
}
func (f *fakeSPKMetricRepo) UpsertUserOverride(_ context.Context, userID string, m domain.SPKMetricModel) error {
	f.upserted = &m
	f.upsertUser = userID
	return nil
}
func (f *fakeSPKMetricRepo) DeleteUserOverride(_ context.Context, userID string) error {
	f.deleted = userID
	return nil
}
func (f *fakeSPKMetricRepo) ReplaceDefault(context.Context, string, string, domain.SPKMetricModel, *string) (*domain.SPKMetricModel, error) {
	return f.def, nil
}

func modelWithOneIndicator(weight, target float64, typ domain.SPKMetricType) domain.SPKMetricModel {
	return domain.SPKMetricModel{
		Indicators: []domain.SPKMetricIndicator{
			{Key: "only", Label: "Only", Weight: weight, Target: target, Type: typ},
		},
		Thresholds: []domain.SPKStatusThreshold{{Min: 0, Label: "Kritis"}},
	}
}

func TestResolveMetricModel_Precedence(t *testing.T) {
	userID := uuid.New()

	t.Run("user override wins", func(t *testing.T) {
		repo := &fakeSPKMetricRepo{
			def:      new(modelWithOneIndicator(1, 1, domain.MetricConsistency)),
			override: new(modelWithOneIndicator(1, 1, domain.MetricSavingRatio)),
		}
		u := &SPKUsecase{metrics: repo}
		got := u.ResolveMetricModel(context.Background(), userID)
		if got.Source != "user" || got.Indicators[0].Type != domain.MetricSavingRatio {
			t.Fatalf("expected user override, got %+v", got)
		}
	})

	t.Run("default when no override", func(t *testing.T) {
		repo := &fakeSPKMetricRepo{def: new(modelWithOneIndicator(1, 1, domain.MetricConsistency))}
		u := &SPKUsecase{metrics: repo}
		got := u.ResolveMetricModel(context.Background(), userID)
		if got.Source != "default" || got.Indicators[0].Type != domain.MetricConsistency {
			t.Fatalf("expected default, got %+v", got)
		}
	})

	t.Run("compiled fallback when nothing stored", func(t *testing.T) {
		u := &SPKUsecase{metrics: &fakeSPKMetricRepo{}}
		got := u.ResolveMetricModel(context.Background(), userID)
		if got.Source != "default" || len(got.Indicators) != 5 {
			t.Fatalf("expected compiled fallback with 5 indicators, got %+v", got)
		}
	})

	t.Run("compiled fallback when the repo errors", func(t *testing.T) {
		u := &SPKUsecase{metrics: &fakeSPKMetricRepo{defErr: errors.New("boom")}}
		got := u.ResolveMetricModel(context.Background(), userID)
		if len(got.Indicators) != 5 {
			t.Fatalf("expected compiled fallback on error, got %+v", got)
		}
	})

	t.Run("compiled fallback when metrics repo is nil", func(t *testing.T) {
		u := &SPKUsecase{}
		got := u.ResolveMetricModel(context.Background(), userID)
		if len(got.Indicators) != 5 {
			t.Fatalf("expected compiled fallback with nil repo, got %+v", got)
		}
	})
}

func TestScoreIndicator(t *testing.T) {
	cases := []struct {
		name string
		ind  domain.SPKMetricIndicator
		raw  float64
		want float64
	}{
		{"higher_better at target", domain.SPKMetricIndicator{Target: 0.2, Direction: "higher_better"}, 0.2, 100},
		{"higher_better above target caps at 100", domain.SPKMetricIndicator{Target: 0.2, Direction: "higher_better"}, 0.5, 100},
		{"higher_better half target", domain.SPKMetricIndicator{Target: 0.2, Direction: "higher_better"}, 0.1, 50},
		{"higher_better zero", domain.SPKMetricIndicator{Target: 0.2, Direction: "higher_better"}, 0, 0},
		{"lower_better at target", domain.SPKMetricIndicator{Target: 0.1, Direction: "lower_better"}, 0.1, 100},
		{"lower_better below target", domain.SPKMetricIndicator{Target: 0.1, Direction: "lower_better"}, 0.05, 100},
		{"lower_better double target", domain.SPKMetricIndicator{Target: 0.1, Direction: "lower_better"}, 0.2, 50},
		{"lower_better zero", domain.SPKMetricIndicator{Target: 0.1, Direction: "lower_better"}, 0, 100},
		{"zero target is safe", domain.SPKMetricIndicator{Target: 0}, 5, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scoreIndicator(tc.ind, tc.raw); got != tc.want {
				t.Fatalf("scoreIndicator = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetermineStatus_UsesModelThresholds(t *testing.T) {
	th := []domain.SPKStatusThreshold{
		{Min: 90, Label: "A"},
		{Min: 70, Label: "B"},
		{Min: 0, Label: "C"},
	}
	cases := []struct {
		score float64
		want  string
	}{{95, "A"}, {90, "A"}, {89.9, "B"}, {70, "B"}, {10, "C"}, {0, "C"}}
	for _, tc := range cases {
		if got := string(determineStatus(th, tc.score)); got != tc.want {
			t.Errorf("score %v => %q, want %q", tc.score, got, tc.want)
		}
	}
}

func TestDetermineStatus_FallsBackWhenNoThresholds(t *testing.T) {
	// The compiled default: >=80 is "Sangat Baik".
	if got := string(determineStatus(nil, 85)); got != string(domain.SPKVeryGood) {
		t.Fatalf("expected fallback threshold, got %q", got)
	}
}

func TestNeedsSet_UsesModelCategories(t *testing.T) {
	u := &SPKUsecase{}
	custom := domain.SPKMetricModel{
		Indicators: []domain.SPKMetricIndicator{
			{Type: domain.MetricNeedsRatio, Categories: []string{"Sewa", "Listrik"}},
		},
	}
	set := u.needsSet(custom)
	if !set["Sewa"] || !set["Listrik"] || len(set) != 2 {
		t.Fatalf("expected custom needs set, got %+v", set)
	}

	// An empty category list falls back to the compiled default.
	fallback := u.needsSet(domain.SPKMetricModel{Indicators: []domain.SPKMetricIndicator{{Type: domain.MetricNeedsRatio}}})
	if !fallback["Makanan & Minuman"] {
		t.Fatalf("expected fallback needs set, got %+v", fallback)
	}
}
