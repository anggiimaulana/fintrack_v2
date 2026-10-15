package billing

import (
	"context"
	"math"
	"strconv"
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
//
// Chat/Reasoning stay flat because their reserve is computed dynamically by
// ForChat (this value is only the floor). Vision/Voice are ALSO computed
// dynamically now (ForVision/ForVoice), so their flat default is a small safety
// minimum rather than the whole reserve — a large flat floor would defeat the
// dynamic calculation entirely.
func DefaultQuotaEstimate() QuotaEstimate {
	return QuotaEstimate{
		Chat:      2000,
		Reasoning: 4000,
		Vision:    500,
		Voice:     200,
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

// Fixed overhead (in tokens) every chat request carries regardless of the user's
// message: the system prompt, the tool instructions, the RAG block and the last
// 20 history messages. Reserve only the flat default and this overhead alone can
// exceed a Free user's whole cycle, which is what made one chat empty the quota.
const (
	chatOverheadTokens       = 3000
	chatOutputHeadroomTokens = 800
	// charsPerToken approximates an Indonesian/English mix. It is deliberately
	// conservative (a real token averages ~4 chars) so the reserve errs high.
	charsPerToken = 4
)

// Default constants for voice dynamic estimation.
const (
	defaultVoiceBytesPerSec  = 4000 // WebM Opus ~32 kbps (~4 KB/s)
	defaultVoiceTokensPerSec = 25   // Provider tokens per second of audio
	defaultVoiceOverhead     = 50   // Base processing overhead
	defaultVoiceHeadroom     = 150  // Expected transcript completion tokens
)

// Default constants for vision dynamic estimation.
const (
	defaultVisionImageTokens    = 800    // Flat token cost for a standard receipt image
	defaultVisionOutputHeadroom = 400    // Expected JSON receipt extraction tokens
	defaultVisionLargeFileBytes = 500000 // 500 KB threshold for high-resolution images
	defaultVisionLargeFileExtra = 400    // Extra tokens for large images (extra tiles)
)

// SettingsReader is the narrow slice of the settings repository we need.
type SettingsReader interface {
	GetInt(ctx context.Context, key string) (int64, error)
	GetString(ctx context.Context, key string) (string, error)
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

// ForChat returns the reserve for one chat turn, scaled to the request itself
// rather than a flat constant.
//
// A flat reserve is wrong in both directions: too small and the commit over-runs
// (which the ledger clamps, silently consuming the rest of the cycle); too large
// and a request that would fit is refused. Scaling by promptChars keeps the hold
// close to the real cost, and the fixed overhead accounts for the system prompt,
// tool instructions, RAG block and history that every turn carries.
//
// The configured `ai_estimate_chat` (or reasoning) value acts as the floor, so a
// Super Admin can still raise the minimum hold.
func (s *EstimateStore) ForChat(ctx context.Context, modality Modality, promptChars int64) int64 {
	base := s.For(ctx, modality)
	if base < 1 {
		base = 1
	}
	if promptChars < 0 {
		promptChars = 0
	}
	// The overhead and chars-per-token are OPERATOR-CONFIGURABLE settings
	// (P-13); the constants above are only the fallback when unset. This keeps
	// the estimate tunable without a deploy — and lets a deployment (or a test)
	// set them to 0 when the provider already reports exact usage.
	overhead := s.settingInt(ctx, "ai_estimate_overhead_tokens", chatOverheadTokens)
	headroom := s.settingInt(ctx, "ai_estimate_output_headroom", chatOutputHeadroomTokens)
	charsPer := s.settingInt(ctx, "ai_estimate_chars_per_token", charsPerToken)
	if charsPer < 1 {
		charsPer = 1
	}
	estimated := overhead + (promptChars/charsPer) + headroom
	if estimated < base {
		return base
	}
	return estimated
}

// ForVoice returns the reserve for one voice transcription call, dynamically
// scaled by audio byte length. A 2-second voice memo reserves only ~200 tokens
// rather than locking a flat 2000 tokens (which unfairly rejects low-balance
// users), while longer audio clips receive a proportional hold that prevents
// over-run clamping.
func (s *EstimateStore) ForVoice(ctx context.Context, audioBytes int64) int64 {
	// The configured `ai_estimate_voice` acts as a floor, exactly like
	// ForChat's base — an operator can raise the minimum hold to be safe.
	base := s.For(ctx, ModalityVoice)
	if base < 1 {
		base = 1
	}
	if audioBytes < 0 {
		audioBytes = 0
	}
	bytesPerSec := s.settingInt(ctx, "ai_estimate_voice_bytes_per_sec", defaultVoiceBytesPerSec)
	if bytesPerSec < 100 {
		bytesPerSec = defaultVoiceBytesPerSec
	}
	tokensPerSec := s.settingInt(ctx, "ai_estimate_voice_tokens_per_sec", defaultVoiceTokensPerSec)
	if tokensPerSec < 1 {
		tokensPerSec = 1
	}
	overhead := s.settingInt(ctx, "ai_estimate_voice_overhead", defaultVoiceOverhead)
	headroom := s.settingInt(ctx, "ai_estimate_voice_output_headroom", defaultVoiceHeadroom)

	// Approximate duration in seconds (rounded up to avoid zero-length clip bias)
	sec := float64(audioBytes) / float64(bytesPerSec)
	promptTokens := int64(math.Ceil(sec * float64(tokensPerSec)))
	estimated := promptTokens + overhead + headroom

	if estimated < base {
		return base
	}
	return estimated
}

// ForVision returns the reserve for one receipt OCR call, dynamically scaled
// by image file size and prompt length (which varies as categories are added).
func (s *EstimateStore) ForVision(ctx context.Context, imageBytes int64, promptChars int64) int64 {
	// The configured `ai_estimate_vision` acts as a floor, exactly like
	// ForChat's base — an operator can raise the minimum hold to be safe.
	base := s.For(ctx, ModalityVision)
	if base < 1 {
		base = 1
	}
	if imageBytes < 0 {
		imageBytes = 0
	}
	if promptChars < 0 {
		promptChars = 0
	}
	charsPer := s.settingInt(ctx, "ai_estimate_chars_per_token", charsPerToken)
	if charsPer < 1 {
		charsPer = 1
	}
	imageTokens := s.settingInt(ctx, "ai_estimate_vision_image_tokens", defaultVisionImageTokens)
	headroom := s.settingInt(ctx, "ai_estimate_vision_output_headroom", defaultVisionOutputHeadroom)
	largeThreshold := s.settingInt(ctx, "ai_estimate_vision_large_file_bytes", defaultVisionLargeFileBytes)
	largeExtra := s.settingInt(ctx, "ai_estimate_vision_large_file_extra", defaultVisionLargeFileExtra)

	promptTokens := promptChars / charsPer
	totalImage := imageTokens
	if imageBytes > largeThreshold {
		totalImage += largeExtra
	}
	estimated := promptTokens + totalImage + headroom

	if estimated < base {
		return base
	}
	return estimated
}

// settingInt reads an integer setting, falling back to `def` when the setting
// is ABSENT. A configured 0 is honoured (it is a legitimate value: it disables
// that component of the estimate), which is why this reads the raw string
// rather than GetInt — GetInt cannot distinguish "absent" from "0".
func (s *EstimateStore) settingInt(ctx context.Context, key string, def int64) int64 {
	if s.settings == nil {
		return def
	}
	raw, err := s.settings.GetString(ctx, key)
	if err != nil || raw == "" {
		return def
	}
	v, perr := strconv.ParseInt(raw, 10, 64)
	if perr != nil || v < 0 {
		return def
	}
	return v
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
