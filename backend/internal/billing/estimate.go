package billing

import (
	"context"
	"strings"
	"sync"
	"time"
)

// QuotaEstimate holds the per-modality token amounts RESERVED before an AI call.
//
// The reserve is a hold, not a charge: the real cost is settled at commit with
// the provider-reported token count. An estimate that is too small means a large
// over-run at commit (which the ledger clamps), so these values should sit near
// the typical usage per call.
type QuotaEstimate struct {
	Chat      int64 `json:"chat"`
	Reasoning int64 `json:"reasoning"`
	Vision    int64 `json:"vision"`
	Voice     int64 `json:"voice"`
	Embedding int64 `json:"embedding"`
}

// DefaultQuotaEstimate mirrors the seed values in `settings` (ai_estimate_*).
// They are intentionally generous enough to cover a typical call.
func DefaultQuotaEstimate() QuotaEstimate {
	return QuotaEstimate{
		Chat:      2000,
		Reasoning: 4000,
		Vision:    5000,
		Voice:     2000,
		Embedding: 1000,
	}
}

// EstimateStore reads the configured estimates and caches them briefly so the
// hot path (every AI request) does not hit the database. Values are editable by
// a Super Admin via `settings` (keys: ai_estimate_chat, ai_estimate_vision, ...).
type EstimateStore struct {
	settings SettingsReader
	mu       sync.RWMutex
	current  QuotaEstimate
	expires  time.Time
	ttl      time.Duration
}

// SettingsReader is the narrow slice of the settings repository we need.
type SettingsReader interface {
	GetInt(ctx context.Context, key string) (int64, error)
}

// NewEstimateStore builds the store, seeding it with the defaults.
func NewEstimateStore(settings SettingsReader) *EstimateStore {
	return &EstimateStore{settings: settings, current: DefaultQuotaEstimate(), ttl: 30 * time.Second}
}

// For returns the reserve estimate for a modality, refreshing from settings when
// the cached copy has expired. A zero/absent setting keeps the default.
func (s *EstimateStore) For(ctx context.Context, modality Modality) int64 {
	e := s.load(ctx)
	switch modality {
	case ModalityVision:
		return e.Vision
	case ModalityVoice:
		return e.Voice
	case ModalityEmbedding:
		return e.Embedding
	case ModalityReasoning:
		return e.Reasoning
	default:
		return e.Chat
	}
}

// Snapshot returns the current estimates (for the admin UI).
func (s *EstimateStore) Snapshot(ctx context.Context) QuotaEstimate { return s.load(ctx) }

func (s *EstimateStore) load(ctx context.Context) QuotaEstimate {
	s.mu.RLock()
	if time.Now().Before(s.expires) {
		e := s.current
		s.mu.RUnlock()
		return e
	}
	s.mu.RUnlock()

	e := DefaultQuotaEstimate()
	if s.settings != nil {
		e.Chat = s.readKey(ctx, "ai_estimate_chat", e.Chat)
		e.Reasoning = s.readKey(ctx, "ai_estimate_reasoning", e.Reasoning)
		e.Vision = s.readKey(ctx, "ai_estimate_vision", e.Vision)
		e.Voice = s.readKey(ctx, "ai_estimate_voice", e.Voice)
		e.Embedding = s.readKey(ctx, "ai_estimate_embedding", e.Embedding)
	}

	s.mu.Lock()
	s.current = e
	s.expires = time.Now().Add(s.ttl)
	s.mu.Unlock()
	return e
}

// readKey returns the setting value when it is a positive integer, else def.
func (s *EstimateStore) readKey(ctx context.Context, key string, def int64) int64 {
	v, err := s.settings.GetInt(ctx, key)
	if err != nil {
		return def
	}
	if v <= 0 {
		return def
	}
	return v
}

// ParseEstimateKey reports whether key is an AI estimate setting and returns the
// modality name. Used by validation/logging.
func ParseEstimateKey(key string) (string, bool) {
	const prefix = "ai_estimate_"
	if !strings.HasPrefix(key, prefix) {
		return "", false
	}
	return strings.TrimPrefix(key, prefix), true
}

// EstimateSettingKeys lists the settings keys that hold AI reserve estimates.
func EstimateSettingKeys() []string {
	return []string{
		"ai_estimate_chat",
		"ai_estimate_reasoning",
		"ai_estimate_vision",
		"ai_estimate_voice",
		"ai_estimate_embedding",
	}
}
