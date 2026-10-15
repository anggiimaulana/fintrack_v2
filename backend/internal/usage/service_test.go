package usage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeRepo is a stub Repository returning canned points per kind.
type fakeRepo struct {
	points map[string][]SeriesPoint
	// captured granularity/window of the last call, per kind
	lastGran map[string]Granularity
}

func (f *fakeRepo) CountInBuckets(_ context.Context, _ uuid.UUID, kind string, g Granularity, _, _ time.Time) ([]SeriesPoint, error) {
	if f.lastGran == nil {
		f.lastGran = map[string]Granularity{}
	}
	f.lastGran[kind] = g
	return f.points[kind], nil
}

func (f *fakeRepo) History(context.Context, uuid.UUID, HistoryFilter) ([]HistoryEvent, int, error) {
	return nil, 0, nil
}

func newTestService(t *testing.T, now time.Time) *Service {
	t.Helper()
	s := NewService(&fakeRepo{})
	s.now = func() time.Time { return now }
	return s
}

// TestResolveWindow_Ranges asserts each preset maps to the expected granularity.
func TestResolveWindow_Ranges(t *testing.T) {
	now := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	s := newTestService(t, now)

	cases := []struct {
		rangeKey string
		want     Granularity
	}{
		{"24h", GranularityHour},
		{"7d", GranularityDay},
		{"30d", GranularityDay},
		{"3m", GranularityDay},
		{"1y", GranularityMonth},
		{"", GranularityDay}, // default = 7d
	}
	for _, tc := range cases {
		w, err := s.ResolveWindow(tc.rangeKey, "", "")
		if err != nil {
			t.Fatalf("range %q: unexpected error %v", tc.rangeKey, err)
		}
		if w.Granularity != tc.want {
			t.Fatalf("range %q: expected %s, got %s", tc.rangeKey, tc.want, w.Granularity)
		}
	}

	if _, err := s.ResolveWindow("nonsense", "", ""); err == nil {
		t.Fatal("expected error for unknown range")
	}
}

// TestResolveWindow_Custom asserts custom bounds pick a granularity by span.
func TestResolveWindow_Custom(t *testing.T) {
	now := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	s := newTestService(t, now)

	w, err := s.ResolveWindow("custom", "2026-03-14", "2026-03-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Granularity != GranularityHour {
		t.Fatalf("2-day span should be hourly, got %s", w.Granularity)
	}
	if !w.End.After(w.Start) {
		t.Fatalf("end must be after start: %v..%v", w.Start, w.End)
	}

	w, _ = s.ResolveWindow("custom", "2026-01-01", "2026-06-01")
	if w.Granularity != GranularityMonth {
		t.Fatalf("5-month span should be monthly, got %s", w.Granularity)
	}

	// Reversed bounds are swapped, not rejected.
	w, err = s.ResolveWindow("custom", "2026-06-01", "2026-01-01")
	if err != nil || !w.End.After(w.Start) {
		t.Fatalf("reversed bounds should be swapped, got %v..%v err=%v", w.Start, w.End, err)
	}
}

// TestTimeseries_ZeroFills asserts the label axis is dense and values land in
// the right bucket.
func TestTimeseries_ZeroFills(t *testing.T) {
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	s := newTestService(t, now)

	day1 := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	s.repo = &fakeRepo{points: map[string][]SeriesPoint{
		KindTransaction: {{Bucket: day1, Value: 2}, {Bucket: day3, Value: 1}},
		KindAIToken:     {{Bucket: day3, Value: 500}},
	}}

	w, err := s.ResolveWindow("7d", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ts, err := s.Timeseries(context.Background(), uuid.New(), w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ts.Labels) != 7 {
		t.Fatalf("expected 7 labels, got %d (%v)", len(ts.Labels), ts.Labels)
	}
	if len(ts.Series) != len(kindMeta) {
		t.Fatalf("expected %d series, got %d", len(kindMeta), len(ts.Series))
	}

	var txSeries, tokenSeries *Series
	for i := range ts.Series {
		switch ts.Series[i].Key {
		case KindTransaction:
			txSeries = &ts.Series[i]
		case KindAIToken:
			tokenSeries = &ts.Series[i]
		}
	}
	if txSeries == nil || tokenSeries == nil {
		t.Fatal("expected transaction and token series")
	}
	// day1 is index 4 (03-09..03-15), day3 is index 6.
	if txSeries.Data[4] != 2 || txSeries.Data[6] != 1 {
		t.Fatalf("unexpected transaction data: %v", txSeries.Data)
	}
	if txSeries.Data[0] != 0 || txSeries.Data[5] != 0 {
		t.Fatalf("expected zero-filled gaps, got %v", txSeries.Data)
	}
	if tokenSeries.Data[6] != 500 {
		t.Fatalf("unexpected token data: %v", tokenSeries.Data)
	}
	// Every series must be exactly as long as the label axis.
	for _, ser := range ts.Series {
		if len(ser.Data) != len(ts.Labels) {
			t.Fatalf("series %s length %d != labels %d", ser.Key, len(ser.Data), len(ts.Labels))
		}
	}
}

// TestBucketsFor asserts bucket counts for each granularity.
func TestBucketsFor(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 7, 23, 0, 0, 0, time.UTC)

	if got := len(bucketsFor(Window{Granularity: GranularityHour, Start: start, End: end})); got != 7*24 {
		t.Fatalf("expected 168 hourly buckets, got %d", got)
	}
	if got := len(bucketsFor(Window{Granularity: GranularityDay, Start: start, End: end})); got != 7 {
		t.Fatalf("expected 7 daily buckets, got %d", got)
	}
	if got := len(bucketsFor(Window{Granularity: GranularityMonth, Start: start, End: end})); got != 1 {
		t.Fatalf("expected 1 monthly bucket, got %d", got)
	}
}

// TestAutoGranularity asserts the span-to-bucket heuristic.
func TestAutoGranularity(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := autoGranularity(base, base.Add(36*time.Hour)); got != GranularityHour {
		t.Fatalf("36h should be hourly, got %s", got)
	}
	if got := autoGranularity(base, base.AddDate(0, 0, 30)); got != GranularityDay {
		t.Fatalf("30d should be daily, got %s", got)
	}
	if got := autoGranularity(base, base.AddDate(0, 6, 0)); got != GranularityMonth {
		t.Fatalf("6mo should be monthly, got %s", got)
	}
}
