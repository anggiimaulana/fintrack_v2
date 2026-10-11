package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/fintrack/backend/internal/domain"
)

// fakeGateway implements domain.AIGatewayService for adapter tests.
type fakeGateway struct {
	chatResponse string
	chatErr      error

	visionResponse string
	visionErr      error

	voiceResponse string
	voiceErr      error

	embedding []float32
	embedErr  error

	lastVisionReq *domain.AIVisionRequest
	lastVoiceReq  *domain.AIAudioTranscriptionRequest
	lastChatReq   *domain.AIChatRequest
}

func (f *fakeGateway) ChatStream(_ context.Context, req *domain.AIChatRequest) (<-chan domain.AIChatStreamChunk, error) {
	f.lastChatReq = req
	if f.chatErr != nil {
		return nil, f.chatErr
	}
	ch := make(chan domain.AIChatStreamChunk, 2)
	go func() {
		defer close(ch)
		ch <- domain.AIChatStreamChunk{Content: f.chatResponse}
		ch <- domain.AIChatStreamChunk{Done: true, Usage: domain.TokenUsage{TotalTokens: 5, Source: "provider_reported"}}
	}()
	return ch, nil
}

func (f *fakeGateway) ChatSync(_ context.Context, req *domain.AIChatRequest) (string, domain.TokenUsage, error) {
	f.lastChatReq = req
	return f.chatResponse, domain.TokenUsage{TotalTokens: 5, Source: "provider_reported"}, f.chatErr
}

func (f *fakeGateway) CreateEmbedding(context.Context, *domain.AIEmbeddingRequest) ([]float32, domain.TokenUsage, error) {
	return f.embedding, domain.TokenUsage{TotalTokens: 3}, f.embedErr
}

func (f *fakeGateway) TranscribeAudio(_ context.Context, req *domain.AIAudioTranscriptionRequest) (string, domain.TokenUsage, error) {
	f.lastVoiceReq = req
	return f.voiceResponse, domain.TokenUsage{TotalTokens: 10}, f.voiceErr
}

func (f *fakeGateway) AnalyzeVision(_ context.Context, req *domain.AIVisionRequest) (string, domain.TokenUsage, error) {
	f.lastVisionReq = req
	return f.visionResponse, domain.TokenUsage{TotalTokens: 20}, f.visionErr
}

// TestAnalyzeReceiptViaGateway_ParsesJSON verifies vision via gateway parses
// the model's JSON into the receipt result map.
func TestAnalyzeReceiptViaGateway_ParsesJSON(t *testing.T) {
	gw := &fakeGateway{visionResponse: `{"merchant":"Toko A","total":50000,"category":"Makanan","type":"expense","items":[]}`}
	u := (&AIUsecase{}).WithGateway(gw)

	res, err := u.analyzeReceiptViaGateway(context.Background(), []byte("fakeimg"), "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res["success"] != true {
		t.Fatalf("expected success true, got %v", res["success"])
	}
	if res["merchant"] != "Toko A" {
		t.Fatalf("expected merchant Toko A, got %v", res["merchant"])
	}
	if gw.lastVisionReq == nil || gw.lastVisionReq.ImageMIME != "image/jpeg" {
		t.Fatalf("expected vision request with MIME, got %+v", gw.lastVisionReq)
	}
	if gw.lastVisionReq.ImageBase64 == "" {
		t.Fatal("expected base64 image attached")
	}
}

// TestAnalyzeReceiptViaGateway_ErrorFriendly verifies gateway errors map to a
// friendly result (never a raw error to the user).
func TestAnalyzeReceiptViaGateway_ErrorFriendly(t *testing.T) {
	gw := &fakeGateway{visionErr: domain.ErrAllEndpointsExhausted}
	u := (&AIUsecase{}).WithGateway(gw)

	res, err := u.analyzeReceiptViaGateway(context.Background(), []byte("img"), "image/jpeg")
	if err != nil {
		t.Fatalf("adapter must swallow upstream errors, got %v", err)
	}
	if res["success"] != false {
		t.Fatalf("expected success false on error, got %v", res["success"])
	}
}

// TestProcessVoiceViaGateway_Success verifies transcription via gateway.
func TestProcessVoiceViaGateway_Success(t *testing.T) {
	gw := &fakeGateway{voiceResponse: "  halo dunia  "}
	u := (&AIUsecase{}).WithGateway(gw)

	res, err := u.processVoiceViaGateway(context.Background(), "aGVsbG8=") // base64 "hello"
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res["success"] != true {
		t.Fatalf("expected success, got %v", res["success"])
	}
	if res["transcript"] != "halo dunia" {
		t.Fatalf("expected trimmed transcript, got %q", res["transcript"])
	}
	if gw.lastVoiceReq == nil || len(gw.lastVoiceReq.Audio) == 0 {
		t.Fatal("expected audio bytes forwarded to gateway")
	}
}

// TestProcessVoiceViaGateway_InvalidBase64 verifies bad input is handled.
func TestProcessVoiceViaGateway_InvalidBase64(t *testing.T) {
	gw := &fakeGateway{}
	u := (&AIUsecase{}).WithGateway(gw)
	res, err := u.processVoiceViaGateway(context.Background(), "!!!not-base64!!!")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res["success"] != false {
		t.Fatalf("expected failure on invalid base64, got %v", res["success"])
	}
}

// TestUseGatewayFlag verifies the gateway flag helpers.
func TestUseGatewayFlag(t *testing.T) {
	u := &AIUsecase{}
	if u.UseGateway() {
		t.Fatal("expected gateway disabled by default")
	}
	u.WithGateway(&fakeGateway{})
	if !u.UseGateway() {
		t.Fatal("expected gateway enabled after WithGateway")
	}
}

// TestGatewayFriendlyError verifies a generic, non-leaky error message.
func TestGatewayFriendlyError(t *testing.T) {
	msg := gatewayFriendlyError(domain.ErrAllEndpointsExhausted)
	if msg == "" || strings.Contains(msg, "exhausted") {
		t.Fatalf("expected friendly message without internal detail, got %q", msg)
	}
}
