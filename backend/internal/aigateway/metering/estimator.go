// Package metering extracts token usage from upstream responses and, when the
// provider omits usage, falls back to a deterministic local estimate. The
// resulting domain.TokenUsage feeds the billing ledger (P04).
package metering

import "github.com/fintrack/backend/internal/domain"

// Source labels recorded on TokenUsage.
const (
	SourceProvider  = "provider_reported"
	SourceHeuristic = "heuristic_estimated"
)

// CharsPerToken is the heuristic ratio for latin/Indonesian text.
const CharsPerToken = 4

// SafetyBufferPct adds headroom to heuristic estimates (10%).
const SafetyBufferPct = 10

// FromProvider builds a TokenUsage from provider-reported values. When all
// values are zero it falls back to the heuristic estimate over the given texts.
func FromProvider(prompt, completion, total int, modality domain.UseCase, promptText, responseText string) domain.TokenUsage {
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
	return Estimate(promptText, responseText, modality)
}

// Estimate computes a heuristic token usage from raw text lengths. It applies a
// 10% safety buffer and never returns a zero total for non-empty input.
func Estimate(promptText, responseText string, modality domain.UseCase) domain.TokenUsage {
	prompt := estimateText(promptText)
	completion := estimateText(responseText)
	total := prompt + completion

	// Multimodal/audio adjustments are applied by callers via Estimate* helpers.
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
func EstimateVision(promptText, responseText string) domain.TokenUsage {
	u := Estimate(promptText, responseText, domain.UseCaseVision)
	// Flat estimate per image (low-res patch approximation).
	const imageTokens = 85
	u.PromptTokens += imageTokens
	u.TotalTokens += imageTokens
	return u
}

// EstimateAudio estimates usage for audio: 1 second ≈ 25 tokens.
func EstimateAudio(seconds float64, responseText string) domain.TokenUsage {
	const tokensPerSecond = 25
	prompt := int(float64(tokensPerSecond) * seconds)
	completion := estimateText(responseText)
	return domain.TokenUsage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      prompt + completion,
		Source:           SourceHeuristic,
		Modality:         string(domain.UseCaseVoice),
	}
}

// estimateText applies ceil(len/CharsPerToken) with a safety buffer.
func estimateText(s string) int {
	if s == "" {
		return 0
	}
	// Count runes to be fair to multi-byte input; fall back to bytes for latin.
	n := len([]rune(s))
	base := (n + CharsPerToken - 1) / CharsPerToken // ceil division
	buffered := base + (base*SafetyBufferPct+99)/100
	if buffered < 1 {
		buffered = 1
	}
	return buffered
}
