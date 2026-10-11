package embedding

import (
	"context"
	"testing"

	"github.com/fintrack/backend/internal/domain"
)

// fakeGateway implements just enough of domain.AIGatewayService for embedder tests.
type fakeGateway struct {
	vec []float32
	err error
}

func (f *fakeGateway) ChatStream(context.Context, *domain.AIChatRequest) (<-chan domain.AIChatStreamChunk, error) {
	return nil, nil
}
func (f *fakeGateway) ChatSync(context.Context, *domain.AIChatRequest) (string, domain.TokenUsage, error) {
	return "", domain.TokenUsage{}, nil
}
func (f *fakeGateway) CreateEmbedding(context.Context, *domain.AIEmbeddingRequest) ([]float32, domain.TokenUsage, error) {
	return f.vec, domain.TokenUsage{TotalTokens: 4}, f.err
}
func (f *fakeGateway) TranscribeAudio(context.Context, *domain.AIAudioTranscriptionRequest) (string, domain.TokenUsage, error) {
	return "", domain.TokenUsage{}, nil
}
func (f *fakeGateway) AnalyzeVision(context.Context, *domain.AIVisionRequest) (string, domain.TokenUsage, error) {
	return "", domain.TokenUsage{}, nil
}

// TestGatewayEmbedder_Accepts768 verifies a valid vector passes.
func TestGatewayEmbedder_Accepts768(t *testing.T) {
	vec := make([]float32, 768)
	e := NewGatewayEmbedder(&fakeGateway{vec: vec})
	if !e.Enabled() {
		t.Fatal("expected enabled")
	}
	got, err := e.Embed(context.Background(), "halo")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 768 {
		t.Fatalf("expected 768 dims, got %d", len(got))
	}
}

// TestGatewayEmbedder_RejectsWrongDimension verifies the 768 lock.
func TestGatewayEmbedder_RejectsWrongDimension(t *testing.T) {
	e := NewGatewayEmbedder(&fakeGateway{vec: make([]float32, 512)})
	_, err := e.Embed(context.Background(), "halo")
	if err != ErrInvalidDimension {
		t.Fatalf("expected ErrInvalidDimension, got %v", err)
	}
}

// TestGatewayEmbedder_Disabled verifies disabled behaviour.
func TestGatewayEmbedder_Disabled(t *testing.T) {
	e := NewGatewayEmbedder(nil)
	if e.Enabled() {
		t.Fatal("expected disabled")
	}
	if _, err := e.Embed(context.Background(), "x"); err != ErrDisabled {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}

// TestValidateDimensions verifies the helper.
func TestValidateDimensions(t *testing.T) {
	if err := validateDimensions(make([]float32, 768)); err != nil {
		t.Fatalf("768 should pass, got %v", err)
	}
	if err := validateDimensions(make([]float32, 767)); err != ErrInvalidDimension {
		t.Fatalf("767 should fail, got %v", err)
	}
}

// TestExpectedDimensionsLock asserts the constant.
func TestExpectedDimensionsLock(t *testing.T) {
	if ExpectedDimensions != 768 {
		t.Fatalf("expected 768, got %d", ExpectedDimensions)
	}
}
