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
