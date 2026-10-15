package voice

import (
	"context"
	"testing"
	"time"
)

// stubReader is a ConfigReader backed by a map, so tests exercise the
// resolution/normalisation logic without a database.
type stubReader struct {
	strings map[string]string
	bools   map[string]bool
	ints    map[string]int
}

func (s stubReader) String(_ context.Context, key, _ string, def string) string {
	if v, ok := s.strings[key]; ok {
		return v
	}
	return def
}

func (s stubReader) Bool(_ context.Context, key, _ string, def bool) bool {
	if v, ok := s.bools[key]; ok {
		return v
	}
	return def
}

func (s stubReader) Int(_ context.Context, key, _ string, def int) int {
	if v, ok := s.ints[key]; ok {
		return v
	}
	return def
}

func TestLoad_NilReaderUsesDefaults(t *testing.T) {
	cfg := Load(context.Background(), nil)
	if cfg.MaxDurationSeconds != DefaultMaxDurationSeconds {
		t.Errorf("max duration = %d, want %d", cfg.MaxDurationSeconds, DefaultMaxDurationSeconds)
	}
	if cfg.LanguageMode != LanguageModeIDOnly {
		t.Errorf("language mode = %q, want id_only", cfg.LanguageMode)
	}
	if len(cfg.Languages) != 1 || cfg.Languages[0] != DefaultLanguage {
		t.Errorf("languages = %v, want [%s]", cfg.Languages, DefaultLanguage)
	}
	if !cfg.AllowEditBeforeSend || !cfg.VUMeterEnabled || !cfg.FallbackToBrowser || !cfg.ServerEnabled {
		t.Error("boolean defaults should all be true")
	}
}

func TestLoad_MultiModeExposesLanguageList(t *testing.T) {
	cfg := Load(context.Background(), stubReader{
		strings: map[string]string{
			KeyLanguageMode:    LanguageModeMulti,
			KeyDefaultLanguage: "id",
			KeyLanguages:       "id, en ,",
		},
	})
	if cfg.LanguageMode != LanguageModeMulti {
		t.Fatalf("mode = %q, want multi", cfg.LanguageMode)
	}
	if len(cfg.Languages) != 2 || cfg.Languages[0] != "id" || cfg.Languages[1] != "en" {
		t.Errorf("languages = %v, want [id en] (CSV trimmed, empties dropped)", cfg.Languages)
	}
}

func TestLoad_IDOnlyForcesSingleLanguage(t *testing.T) {
	// Even with a configured list, id_only collapses to the default language.
	cfg := Load(context.Background(), stubReader{
		strings: map[string]string{
			KeyLanguageMode: LanguageModeIDOnly,
			KeyLanguages:    "id,en,ja",
		},
	})
	if len(cfg.Languages) != 1 || cfg.Languages[0] != cfg.DefaultLanguage {
		t.Errorf("languages = %v, want [%s] under id_only", cfg.Languages, cfg.DefaultLanguage)
	}
}

func TestLoad_ClampsOutOfRangeValues(t *testing.T) {
	cfg := Load(context.Background(), stubReader{
		ints: map[string]int{
			KeyMaxDurationSeconds:  9999,
			KeyHistoryRetentionDay: 0,
		},
	})
	if cfg.MaxDurationSeconds != 600 {
		t.Errorf("max duration = %d, want clamped 600", cfg.MaxDurationSeconds)
	}
	if cfg.HistoryRetentionDay != DefaultHistoryRetentionDay {
		t.Errorf("retention = %d, want default %d", cfg.HistoryRetentionDay, DefaultHistoryRetentionDay)
	}
}

func TestLoad_InvalidTimezoneFallsBack(t *testing.T) {
	cfg := Load(context.Background(), stubReader{
		strings: map[string]string{KeyResetTimezone: "Not/AZone"},
	})
	if cfg.ResetTimezone != DefaultResetTimezone {
		t.Errorf("timezone = %q, want fallback %q", cfg.ResetTimezone, DefaultResetTimezone)
	}
}

func TestDayWindow_RespectsTimezone(t *testing.T) {
	// 2026-10-15T18:30:00Z is 2026-10-16 01:30 in Jakarta (+7), so "today" there
	// starts at 2026-10-16 00:00 WIB (= 2026-10-15 17:00 UTC).
	now := time.Date(2026, 10, 15, 18, 30, 0, 0, time.UTC)
	start, end := DayWindow(now, "Asia/Jakarta")

	if start.Day() != 16 {
		t.Errorf("start day = %d, want 16 (Jakarta)", start.Day())
	}
	if got := end.Sub(start); got != 24*time.Hour {
		t.Errorf("window = %v, want 24h", got)
	}
	if !now.After(start) || !now.Before(end) {
		t.Error("now must fall inside its own day window")
	}
}

func TestDayWindow_UTCMidnight(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	start, _ := DayWindow(now, "UTC")
	if start.Day() != 15 || start.Hour() != 0 {
		t.Errorf("UTC start = %v, want 2026-10-15 00:00 UTC", start)
	}
}

func TestResetsAt_IsNextMidnight(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	got := ResetsAt(now, "UTC")
	if got != "2026-10-16T00:00:00Z" {
		t.Errorf("resets at = %q, want 2026-10-16T00:00:00Z", got)
	}
}
