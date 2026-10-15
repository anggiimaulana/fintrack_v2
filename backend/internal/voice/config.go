// Package voice resolves the runtime configuration for the voice-input feature
// and computes the daily-limit window.
//
// Every value is dynamic: an explicit `settings` row wins, otherwise the env
// default supplies it (see runtimeconfig.Resolver). Nothing here is hardcoded
// except the documented fallback defaults, so a Super Admin can retune the
// feature without a redeploy (K-VOI-19).
package voice

import (
	"context"
	"strings"
	"time"
)

// Defaults used when neither a DB setting nor an env var is present.
const (
	DefaultMaxDurationSeconds  = 60
	DefaultLanguageMode        = "id_only"
	DefaultLanguage            = "id"
	DefaultHistoryRetentionDay = 90
	DefaultResetTimezone       = "Asia/Jakarta"
	DefaultAllowEditBeforeSend = true
	DefaultVUMeterEnabled      = true
	DefaultFallbackToBrowser   = true
	DefaultServerEnabled       = true

	// LanguageModeIDOnly forces the default language; LanguageModeMulti lets the
	// user choose per recording.
	LanguageModeIDOnly = "id_only"
	LanguageModeMulti  = "multi"
)

// Setting keys (must match the writableSettings allowlist in internal/admin).
const (
	KeyMaxDurationSeconds  = "voice_max_duration_seconds"
	KeyLanguageMode        = "voice_language_mode"
	KeyDefaultLanguage     = "voice_default_language"
	KeyLanguages           = "voice_languages"
	KeyHistoryRetentionDay = "voice_history_retention_days"
	KeyResetTimezone       = "voice_daily_reset_timezone"
	KeyAllowEditBeforeSend = "voice_allow_edit_before_send"
	KeyVUMeterEnabled      = "voice_vu_meter_enabled"
	KeyFallbackToBrowser   = "voice_fallback_to_browser"
	KeyServerEnabled       = "voice_server_enabled"
)

// ConfigReader resolves one setting with DB-over-env precedence. Satisfied by
// *runtimeconfig.Resolver; kept as an interface so this package is testable
// without a database.
type ConfigReader interface {
	String(ctx context.Context, settingKey, envKey, def string) string
	Bool(ctx context.Context, settingKey, envKey string, def bool) bool
	Int(ctx context.Context, settingKey, envKey string, def int) int
}

// Config is the effective voice configuration for a request.
type Config struct {
	MaxDurationSeconds  int
	LanguageMode        string
	DefaultLanguage     string
	Languages           []string
	HistoryRetentionDay int
	ResetTimezone       string
	AllowEditBeforeSend bool
	VUMeterEnabled      bool
	FallbackToBrowser   bool
	ServerEnabled       bool
}

// Load resolves the effective configuration. A nil reader yields the defaults,
// so callers never have to guard.
func Load(ctx context.Context, r ConfigReader) Config {
	if r == nil {
		return defaults()
	}
	cfg := Config{
		MaxDurationSeconds:  r.Int(ctx, KeyMaxDurationSeconds, "VOICE_MAX_DURATION_SECONDS", DefaultMaxDurationSeconds),
		LanguageMode:        r.String(ctx, KeyLanguageMode, "VOICE_LANGUAGE_MODE", DefaultLanguageMode),
		DefaultLanguage:     r.String(ctx, KeyDefaultLanguage, "VOICE_DEFAULT_LANGUAGE", DefaultLanguage),
		Languages:           splitCSV(r.String(ctx, KeyLanguages, "VOICE_LANGUAGES", DefaultLanguage)),
		HistoryRetentionDay: r.Int(ctx, KeyHistoryRetentionDay, "VOICE_HISTORY_RETENTION_DAYS", DefaultHistoryRetentionDay),
		ResetTimezone:       r.String(ctx, KeyResetTimezone, "VOICE_DAILY_RESET_TIMEZONE", DefaultResetTimezone),
		AllowEditBeforeSend: r.Bool(ctx, KeyAllowEditBeforeSend, "VOICE_ALLOW_EDIT_BEFORE_SEND", DefaultAllowEditBeforeSend),
		VUMeterEnabled:      r.Bool(ctx, KeyVUMeterEnabled, "VOICE_VU_METER_ENABLED", DefaultVUMeterEnabled),
		FallbackToBrowser:   r.Bool(ctx, KeyFallbackToBrowser, "VOICE_FALLBACK_TO_BROWSER", DefaultFallbackToBrowser),
		ServerEnabled:       r.Bool(ctx, KeyServerEnabled, "VOICE_SERVER_ENABLED", DefaultServerEnabled),
	}
	return cfg.normalize()
}

func defaults() Config {
	return Config{
		MaxDurationSeconds:  DefaultMaxDurationSeconds,
		LanguageMode:        DefaultLanguageMode,
		DefaultLanguage:     DefaultLanguage,
		Languages:           []string{DefaultLanguage},
		HistoryRetentionDay: DefaultHistoryRetentionDay,
		ResetTimezone:       DefaultResetTimezone,
		AllowEditBeforeSend: DefaultAllowEditBeforeSend,
		VUMeterEnabled:      DefaultVUMeterEnabled,
		FallbackToBrowser:   DefaultFallbackToBrowser,
		ServerEnabled:       DefaultServerEnabled,
	}
}

// normalize clamps values to sane ranges and guarantees the language list is
// consistent with the mode (a multi-mode config always advertises at least the
// default language).
func (c Config) normalize() Config {
	if c.MaxDurationSeconds < 1 {
		c.MaxDurationSeconds = DefaultMaxDurationSeconds
	}
	if c.MaxDurationSeconds > 600 {
		c.MaxDurationSeconds = 600
	}
	if c.HistoryRetentionDay < 1 {
		c.HistoryRetentionDay = DefaultHistoryRetentionDay
	}
	if c.LanguageMode != LanguageModeMulti {
		c.LanguageMode = LanguageModeIDOnly
	}
	if strings.TrimSpace(c.DefaultLanguage) == "" {
		c.DefaultLanguage = DefaultLanguage
	}
	if c.LanguageMode == LanguageModeIDOnly {
		// id_only ignores any configured list.
		c.Languages = []string{c.DefaultLanguage}
	} else if len(c.Languages) == 0 {
		c.Languages = []string{c.DefaultLanguage}
	}
	if _, err := time.LoadLocation(c.ResetTimezone); err != nil {
		c.ResetTimezone = DefaultResetTimezone
	}
	return c
}

// splitCSV trims and drops empty entries from a comma-separated list.
func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// DayWindow returns the [start, end) bounds of "today" in the configured
// timezone, at the instant `now`. The end is the start of the next day.
//
// The daily limit counts rows created within this window, so the reset happens
// at local midnight in the configured zone (N-03).
func DayWindow(now time.Time, tz string) (start, end time.Time) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	start = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	end = start.AddDate(0, 0, 1)
	return start, end
}

// ResetsAt returns the RFC3339 timestamp of the next daily reset, for the
// client to display ("resets in X hours").
func ResetsAt(now time.Time, tz string) string {
	_, end := DayWindow(now, tz)
	return end.Format(time.RFC3339)
}
