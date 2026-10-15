package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Series is one named data series for a chart.
type Series struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Unit  string  `json:"unit"`
	Data  []int64 `json:"data"`
}

// Timeseries is the chart payload: a shared label axis plus one series per kind.
type Timeseries struct {
	Range       string   `json:"range"`
	Granularity string   `json:"granularity"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
	Labels      []string `json:"labels"`
	Series      []Series `json:"series"`
}

// kindMeta is the display metadata for each consumption kind.
var kindMeta = []struct {
	Key   string
	Label string
	Unit  string
}{
	{KindTransaction, "Transaksi", "transaksi"},
	{KindAIToken, "Token AI", "token"},
	{KindAIRequest, "Permintaan AI", "permintaan"},
	{KindOCR, "Pindai Struk", "pindai"},
	{KindVoice, "Input Suara", "permintaan"},
}

// Service builds chart payloads from the usage repository.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService builds the usage service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// ResolveWindow turns a range token (24h, 7d, 30d, 3m, 1y, custom) plus optional
// explicit bounds into a concrete window and bucket granularity.
func (s *Service) ResolveWindow(rangeKey, startRaw, endRaw string) (Window, error) {
	now := s.now()

	if rangeKey == "custom" || (startRaw != "" && endRaw != "") {
		start, err := parseBound(startRaw, false)
		if err != nil {
			return Window{}, fmt.Errorf("start tidak valid")
		}
		end, err := parseBound(endRaw, true)
		if err != nil {
			return Window{}, fmt.Errorf("end tidak valid")
		}
		if start.After(end) {
			start, end = end, start
		}
		return Window{Range: "custom", Granularity: autoGranularity(start, end), Start: start, End: end}, nil
	}

	switch rangeKey {
	case "", "7d":
		return Window{Range: "7d", Granularity: GranularityDay, Start: now.AddDate(0, 0, -6).Truncate(24 * time.Hour), End: now}, nil
	case "24h":
		return Window{Range: "24h", Granularity: GranularityHour, Start: now.Add(-23 * time.Hour).Truncate(time.Hour), End: now}, nil
	case "30d":
		return Window{Range: "30d", Granularity: GranularityDay, Start: now.AddDate(0, 0, -29).Truncate(24 * time.Hour), End: now}, nil
	case "3m":
		return Window{Range: "3m", Granularity: GranularityDay, Start: now.AddDate(0, 0, -89).Truncate(24 * time.Hour), End: now}, nil
	case "1y":
		return Window{Range: "1y", Granularity: GranularityMonth, Start: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -11, 0), End: now}, nil
	default:
		return Window{}, fmt.Errorf("range tidak dikenal")
	}
}

// Timeseries builds the full chart payload for a user and window.
func (s *Service) Timeseries(ctx context.Context, userID uuid.UUID, w Window) (Timeseries, error) {
	buckets := bucketsFor(w)
	labels := make([]string, len(buckets))
	for i, b := range buckets {
		labels[i] = b.Label
	}

	// Bucket index lookup keyed on the truncated timestamp.
	index := make(map[int64]int, len(buckets))
	for i, b := range buckets {
		index[b.Start.Unix()] = i
	}

	series := make([]Series, 0, len(kindMeta))
	for _, m := range kindMeta {
		points, err := s.repo.CountInBuckets(ctx, userID, m.Key, w.Granularity, w.Start, w.End)
		if err != nil {
			return Timeseries{}, err
		}
		data := make([]int64, len(buckets))
		for _, p := range points {
			if i, ok := index[truncate(p.Bucket, w.Granularity).Unix()]; ok {
				data[i] += p.Value
			}
		}
		series = append(series, Series{Key: m.Key, Label: m.Label, Unit: m.Unit, Data: data})
	}

	return Timeseries{
		Range:       w.Range,
		Granularity: string(w.Granularity),
		Start:       w.Start.Format(time.RFC3339),
		End:         w.End.Format(time.RFC3339),
		Labels:      labels,
		Series:      series,
	}, nil
}

// bucketsFor materialises every bucket (dense: empty buckets are zero-filled by
// the caller) between the window bounds at the window's granularity.
func bucketsFor(w Window) []Bucket {
	cur := truncate(w.Start, w.Granularity)
	out := make([]Bucket, 0, 32)
	for !cur.After(w.End) {
		out = append(out, Bucket{Start: cur, Label: bucketLabel(cur, w.Granularity)})
		switch w.Granularity {
		case GranularityHour:
			cur = cur.Add(time.Hour)
		case GranularityMonth:
			cur = cur.AddDate(0, 1, 0)
		default:
			cur = cur.AddDate(0, 0, 1)
		}
	}
	return out
}

func truncate(t time.Time, g Granularity) time.Time {
	switch g {
	case GranularityHour:
		return t.Truncate(time.Hour)
	case GranularityMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
}

var monthShort = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

func bucketLabel(t time.Time, g Granularity) string {
	switch g {
	case GranularityHour:
		return fmt.Sprintf("%02d:00", t.Hour())
	case GranularityMonth:
		return fmt.Sprintf("%s %02d", monthShort[int(t.Month())-1], t.Year()%100)
	default:
		return fmt.Sprintf("%d %s", t.Day(), monthShort[int(t.Month())-1])
	}
}

// autoGranularity picks a sensible bucket size for a custom span: hourly up to
// two days, daily up to ~3 months, monthly beyond.
func autoGranularity(start, end time.Time) Granularity {
	days := end.Sub(start).Hours() / 24
	switch {
	case days <= 2:
		return GranularityHour
	case days <= 92:
		return GranularityDay
	default:
		return GranularityMonth
	}
}

// parseBound parses an RFC3339 timestamp or a bare YYYY-MM-DD date. A bare end
// date is advanced by 24h so the end day is inclusive (half-open interval).
func parseBound(raw string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		t = t.Add(24 * time.Hour)
	}
	return t.UTC(), nil
}
