// Package metering extracts token usage from upstream responses and, when the
// provider omits usage, falls back to a deterministic local estimate. The
// resulting domain.TokenUsage feeds the billing ledger (P04).
package metering

import (
	"sync"

	"github.com/fintrack/backend/internal/domain"
)

// Source labels recorded on TokenUsage.
const (
	SourceProvider  = "provider_reported"
	SourceHeuristic = "heuristic_estimated"
)

// Default heuristic constants.
const (
	DefaultCharsPerToken       = 4
	DefaultSafetyBufferPct     = 10
	DefaultImageTokens         = 800
	DefaultTokensPerSecond     = 25
	DefaultAudioBytesPerSecond = 4000
	// Deprecated aliases kept for backwards compatibility.
	CharsPerToken   = DefaultCharsPerToken
	SafetyBufferPct = DefaultSafetyBufferPct
)

// Config tunes the local heuristic token estimation.
type Config struct {
	CharsPerToken       int `json:"chars_per_token"`
	SafetyBufferPct     int `json:"safety_buffer_pct"`
	ImageTokens         int `json:"image_tokens"`
	TokensPerSecond     int `json:"tokens_per_second"`
	AudioBytesPerSecond int `json:"audio_bytes_per_second"`
}

// DefaultConfig returns the standard heuristic parameters.
func DefaultConfig() Config {
	return Config{
		CharsPerToken:       DefaultCharsPerToken,
		SafetyBufferPct:     DefaultSafetyBufferPct,
		ImageTokens:         DefaultImageTokens,
		TokensPerSecond:     DefaultTokensPerSecond,
		AudioBytesPerSecond: DefaultAudioBytesPerSecond,
	}
}

// Estimator computes token usage heuristics using configurable parameters.
type Estimator struct {
	mu  sync.RWMutex
	cfg Config
}

// New creates an Estimator with the given configuration.
func New(cfg Config) *Estimator {
	if cfg.CharsPerToken <= 0 {
		cfg.CharsPerToken = DefaultCharsPerToken
	}
	if cfg.SafetyBufferPct < 0 {
		cfg.SafetyBufferPct = DefaultSafetyBufferPct
	}
	if cfg.ImageTokens <= 0 {
		cfg.ImageTokens = DefaultImageTokens
	}
	if cfg.TokensPerSecond <= 0 {
		cfg.TokensPerSecond = DefaultTokensPerSecond
	}
	if cfg.AudioBytesPerSecond <= 0 {
		cfg.AudioBytesPerSecond = DefaultAudioBytesPerSecond
	}
	return &Estimator{cfg: cfg}
}

// SetConfig updates tuning parameters at runtime without dropping state.
func (e *Estimator) SetConfig(cfg Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if cfg.CharsPerToken > 0 {
		e.cfg.CharsPerToken = cfg.CharsPerToken
	}
	if cfg.SafetyBufferPct >= 0 {
		e.cfg.SafetyBufferPct = cfg.SafetyBufferPct
	}
	if cfg.ImageTokens > 0 {
		e.cfg.ImageTokens = cfg.ImageTokens
	}
	if cfg.TokensPerSecond > 0 {
		e.cfg.TokensPerSecond = cfg.TokensPerSecond
	}
	if cfg.AudioBytesPerSecond > 0 {
		e.cfg.AudioBytesPerSecond = cfg.AudioBytesPerSecond
	}
}

// Config returns a snapshot of the current tuning parameters.
func (e *Estimator) Config() Config {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cfg
}

// defaultEstimator is the process-wide estimator used by package-level helpers.
var defaultEstimator = New(DefaultConfig())

// SetConfig updates the package-level default estimator parameters at runtime.
func SetConfig(cfg Config) {
	defaultEstimator.SetConfig(cfg)
}

// CurrentConfig returns the package-level default estimator parameters.
func CurrentConfig() Config {
	return defaultEstimator.Config()
}

// FromProvider builds a TokenUsage from provider-reported values. When all
// values are zero it falls back to the heuristic estimate over the given texts.
func (e *Estimator) FromProvider(prompt, completion, total int, modality domain.UseCase, promptText, responseText string) domain.TokenUsage {
	if total == 0 {
		total = prompt + completion
	}
	if total > 0 {
		return domain.TokenUsage{
			PromptTokens:     prompt,
			CompletionTokens: completion,
			TotalTokens:      total,
			Source:           SourceProvider,
			Modality:         string(modality),
		}
	}
	return e.Estimate(promptText, responseText, modality)
}

// Estimate computes a heuristic token usage from raw text lengths.
func (e *Estimator) Estimate(promptText, responseText string, modality domain.UseCase) domain.TokenUsage {
	prompt := e.estimateText(promptText)
	completion := e.estimateText(responseText)
	total := prompt + completion

	return domain.TokenUsage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
		Source:           SourceHeuristic,
		Modality:         string(modality),
	}
}

// EstimateVision estimates usage for a vision request: text tokens plus a flat
// per-image cost.
func (e *Estimator) EstimateVision(promptText, responseText string) domain.TokenUsage {
	u := e.Estimate(promptText, responseText, domain.UseCaseVision)
	cfg := e.Config()
	u.PromptTokens += cfg.ImageTokens
	u.TotalTokens += cfg.ImageTokens
	return u
}

// EstimateAudio estimates usage for audio: duration * tokensPerSecond.
func (e *Estimator) EstimateAudio(seconds float64, responseText string) domain.TokenUsage {
	cfg := e.Config()
	prompt := int(float64(cfg.TokensPerSecond) * seconds)
	completion := e.estimateText(responseText)
	return domain.TokenUsage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      prompt + completion,
		Source:           SourceHeuristic,
		Modality:         string(domain.UseCaseVoice),
	}
}

// EstimateAudioFromBytes converts raw audio bytes into seconds using AudioBytesPerSecond,
// then computes token usage.
func (e *Estimator) EstimateAudioFromBytes(audioBytes int, responseText string) domain.TokenUsage {
	cfg := e.Config()
	bps := float64(cfg.AudioBytesPerSecond)
	if bps <= 0 {
		bps = float64(DefaultAudioBytesPerSecond)
	}
	seconds := float64(audioBytes) / bps
	return e.EstimateAudio(seconds, responseText)
}

// estimateText applies ceil(len/charsPerToken) with a safety buffer.
func (e *Estimator) estimateText(s string) int {
	if s == "" {
		return 0
	}
	cfg := e.Config()
	cpt := cfg.CharsPerToken
	if cpt <= 0 {
		cpt = DefaultCharsPerToken
	}
	n := len([]rune(s))
	base := (n + cpt - 1) / cpt // ceil division
	buffered := base + (base*cfg.SafetyBufferPct+99)/100
	if buffered < 1 {
		buffered = 1
	}
	return buffered
}

// Package-level helpers delegating to defaultEstimator:

// FromProvider builds a TokenUsage from provider-reported values or falls back to heuristic estimate.
func FromProvider(prompt, completion, total int, modality domain.UseCase, promptText, responseText string) domain.TokenUsage {
	return defaultEstimator.FromProvider(prompt, completion, total, modality, promptText, responseText)
}

// Estimate computes a heuristic token usage from raw text lengths.
func Estimate(promptText, responseText string, modality domain.UseCase) domain.TokenUsage {
	return defaultEstimator.Estimate(promptText, responseText, modality)
}

// EstimateVision estimates usage for a vision request: text tokens plus a flat per-image cost.
func EstimateVision(promptText, responseText string) domain.TokenUsage {
	return defaultEstimator.EstimateVision(promptText, responseText)
}

// EstimateAudio estimates usage for audio: duration * tokensPerSecond.
func EstimateAudio(seconds float64, responseText string) domain.TokenUsage {
	return defaultEstimator.EstimateAudio(seconds, responseText)
}

// EstimateAudioFromBytes converts raw audio bytes into seconds using the default estimator.
func EstimateAudioFromBytes(audioBytes int, responseText string) domain.TokenUsage {
	return defaultEstimator.EstimateAudioFromBytes(audioBytes, responseText)
}
