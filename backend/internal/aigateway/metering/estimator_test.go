package metering

import (
	"testing"

	"github.com/fintrack/backend/internal/domain"
)

// TestFromProvider_UsesReportedValues asserts provider values pass through.
func TestFromProvider_UsesReportedValues(t *testing.T) {
	u := FromProvider(100, 50, 150, domain.UseCaseChat, "prompt", "response")
	if u.TotalTokens != 150 || u.PromptTokens != 100 || u.CompletionTokens != 50 {
		t.Fatalf("unexpected usage: %+v", u)
	}
	if u.Source != SourceProvider {
		t.Fatalf("expected source %s, got %s", SourceProvider, u.Source)
	}
	if u.Modality != string(domain.UseCaseChat) {
		t.Fatalf("expected modality chat, got %s", u.Modality)
	}
}

// TestFromProvider_FallsBackToEstimate asserts zero usage triggers heuristic.
func TestFromProvider_FallsBackToEstimate(t *testing.T) {
	u := FromProvider(0, 0, 0, domain.UseCaseChat, "ini adalah prompt panjang", "ini respons")
	if u.Source != SourceHeuristic {
		t.Fatalf("expected heuristic source, got %s", u.Source)
	}
	if u.TotalTokens <= 0 {
		t.Fatalf("expected positive total, got %d", u.TotalTokens)
	}
}

// TestFromProvider_DerivesTotal asserts total is derived when only parts exist.
func TestFromProvider_DerivesTotal(t *testing.T) {
	u := FromProvider(30, 20, 0, domain.UseCaseChat, "", "")
	if u.TotalTokens != 50 {
		t.Fatalf("expected derived total 50, got %d", u.TotalTokens)
	}
	if u.Source != SourceProvider {
		t.Fatalf("expected provider source, got %s", u.Source)
	}
}

// TestEstimate_PositiveForNonEmpty asserts non-empty text yields > 0 tokens.
func TestEstimate_PositiveForNonEmpty(t *testing.T) {
	u := Estimate("halo dunia", "hai", domain.UseCaseChat)
	if u.TotalTokens <= 0 {
		t.Fatalf("expected positive estimate, got %d", u.TotalTokens)
	}
}

// TestEstimate_EmptyIsZero asserts empty input yields zero (not negative).
func TestEstimate_EmptyIsZero(t *testing.T) {
	u := Estimate("", "", domain.UseCaseChat)
	if u.TotalTokens != 0 {
		t.Fatalf("expected 0 for empty, got %d", u.TotalTokens)
	}
}

// TestEstimate_AppliesBuffer asserts the 10% buffer is applied.
func TestEstimate_AppliesBuffer(t *testing.T) {
	// 40 chars -> ceil(40/4)=10 base -> +10% -> 11.
	text := "abcdefghijklmnopqrstuvwxyz0123456789abcd" // 40 chars
	u := Estimate(text, "", domain.UseCaseChat)
	if u.PromptTokens != 11 {
		t.Fatalf("expected 11 prompt tokens (10 + 10%% buffer), got %d", u.PromptTokens)
	}
}

// TestEstimateVision_AddsImageCost asserts vision adds flat image tokens.
func TestEstimateVision_AddsImageCost(t *testing.T) {
	text := Estimate("prompt", "res", domain.UseCaseVision)
	vis := EstimateVision("prompt", "res")
	if vis.PromptTokens <= text.PromptTokens {
		t.Fatalf("vision should add image tokens: %d vs %d", vis.PromptTokens, text.PromptTokens)
	}
	if vis.Modality != string(domain.UseCaseVision) {
		t.Fatalf("expected vision modality, got %s", vis.Modality)
	}
}

// TestEstimateAudio_UsesDuration asserts audio tokens scale with duration.
func TestEstimateAudio_UsesDuration(t *testing.T) {
	u := EstimateAudio(10, "transkripsi")
	// 10s * 25 = 250 prompt tokens + completion.
	if u.PromptTokens != 250 {
		t.Fatalf("expected 250 prompt tokens, got %d", u.PromptTokens)
	}
	if u.Modality != string(domain.UseCaseVoice) {
		t.Fatalf("expected voice modality, got %s", u.Modality)
	}
}

// TestEstimator_DynamicConfig asserts an Estimator reacts to runtime parameter changes.
func TestEstimator_DynamicConfig(t *testing.T) {
	est := New(DefaultConfig())

	// Base case with default CharsPerToken = 4, buffer = 10%
	text := "12345678" // 8 chars -> ceil(8/4) = 2 base -> +10% (ceil(0.2) = 1) -> 3
	u1 := est.Estimate(text, "", domain.UseCaseChat)
	if u1.PromptTokens != 3 {
		t.Fatalf("expected 3 prompt tokens with charsPerToken=4, got %d", u1.PromptTokens)
	}

	// Retune CharsPerToken to 2 while keeping buffer at 10%
	cfg := est.Config()
	cfg.CharsPerToken = 2
	est.SetConfig(cfg)
	if got := est.Config().CharsPerToken; got != 2 {
		t.Fatalf("expected CharsPerToken=2, got %d", got)
	}
	// 8 chars -> ceil(8/2) = 4 base -> +10% (ceil(0.4) = 1) -> 5
	u2 := est.Estimate(text, "", domain.UseCaseChat)
	if u2.PromptTokens != 5 {
		t.Fatalf("expected 5 prompt tokens with charsPerToken=2, got %d", u2.PromptTokens)
	}

	// Retune SafetyBufferPct to 0% -> 4 base + 0% = 4
	cfg.SafetyBufferPct = 0
	est.SetConfig(cfg)
	u3 := est.Estimate(text, "", domain.UseCaseChat)
	if u3.PromptTokens != 4 {
		t.Fatalf("expected 4 prompt tokens with buffer=0%%, got %d", u3.PromptTokens)
	}

	// Retune ImageTokens from 85 to 200
	cfg.ImageTokens = 200
	est.SetConfig(cfg)
	vis := est.EstimateVision("hello", "world")
	baseVis := est.Estimate("hello", "world", domain.UseCaseVision)
	if vis.PromptTokens != baseVis.PromptTokens+200 {
		t.Fatalf("expected vision image tokens to add 200, got diff=%d", vis.PromptTokens-baseVis.PromptTokens)
	}

	// Retune TokensPerSecond from 25 to 50
	cfg.TokensPerSecond = 50
	est.SetConfig(cfg)
	aud := est.EstimateAudio(2.0, "")
	if aud.PromptTokens != 100 {
		t.Fatalf("expected 2s * 50 = 100 prompt tokens, got %d", aud.PromptTokens)
	}

	// Retune AudioBytesPerSecond from 4000 to 8000: 8000 bytes should now be 1s
	cfg.AudioBytesPerSecond = 8000
	est.SetConfig(cfg)
	fromBytes := est.EstimateAudioFromBytes(8000, "")
	// 1s * 50 tokens/s = 50 prompt tokens
	if fromBytes.PromptTokens != 50 {
		t.Fatalf("expected 8000 bytes @8000 B/s = 1s -> 50 tokens, got %d", fromBytes.PromptTokens)
	}
	if est.Config().AudioBytesPerSecond != 8000 {
		t.Fatalf("SetConfig dropped AudioBytesPerSecond: got %d", est.Config().AudioBytesPerSecond)
	}
}

// TestSetConfig_AppliesAudioBytesPerSecond guards the regression where
// SetConfig silently dropped the field, so a Super Admin change never applied.
func TestSetConfig_AppliesAudioBytesPerSecond(t *testing.T) {
	est := New(DefaultConfig())
	if est.Config().AudioBytesPerSecond != DefaultAudioBytesPerSecond {
		t.Fatalf("default not seeded: got %d", est.Config().AudioBytesPerSecond)
	}
	cfg := est.Config()
	cfg.AudioBytesPerSecond = 16000
	est.SetConfig(cfg)
	if got := est.Config().AudioBytesPerSecond; got != 16000 {
		t.Fatalf("SetConfig must apply AudioBytesPerSecond: got %d want 16000", got)
	}
	// A zero value must not clobber the current setting (guard semantics).
	cfg.AudioBytesPerSecond = 0
	est.SetConfig(cfg)
	if got := est.Config().AudioBytesPerSecond; got != 16000 {
		t.Fatalf("zero must be ignored, not applied: got %d", got)
	}
}

// TestPackage_SetConfig asserts process-wide package functions reflect SetConfig.
func TestPackage_SetConfig(t *testing.T) {
	orig := CurrentConfig()
	defer SetConfig(orig)

	SetConfig(Config{
		CharsPerToken:   2,
		ImageTokens:     150,
		TokensPerSecond: 10,
	})

	cur := CurrentConfig()
	if cur.CharsPerToken != 2 || cur.ImageTokens != 150 || cur.TokensPerSecond != 10 {
		t.Fatalf("package config not updated: %+v", cur)
	}

	vis := EstimateVision("", "")
	if vis.PromptTokens != 150 {
		t.Fatalf("expected EstimateVision prompt tokens = 150, got %d", vis.PromptTokens)
	}
}
