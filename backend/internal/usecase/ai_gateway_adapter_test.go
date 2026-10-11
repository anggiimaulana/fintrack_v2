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

// TestForwardGatewayAttempt_SuccessStreamsContent covers the happy path.
func TestForwardGatewayAttempt_SuccessStreamsContent(t *testing.T) {
	out := make(chan domain.ChatStreamEvent, 8)
	sentAny, err := forwardGatewayAttempt(func(ch chan domain.ChatStreamEvent) error {
		ch <- domain.ChatStreamEvent{Type: "content", Content: "halo"}
		ch <- domain.ChatStreamEvent{Type: "done", Done: true}
		return nil
	}, out)

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !sentAny {
		t.Fatal("expected sentAny=true once content was forwarded")
	}
	if got := <-out; got.Content != "halo" {
		t.Fatalf("content not relayed: %+v", got)
	}
}

// TestForwardGatewayAttempt_ErrorBeforeContentAllowsFallback is the core guard:
// a gateway failure with nothing streamed must be reported so the caller can fall
// back to BYOK.
func TestForwardGatewayAttempt_ErrorBeforeContentAllowsFallback(t *testing.T) {
	out := make(chan domain.ChatStreamEvent, 8)
	sentAny, err := forwardGatewayAttempt(func(chan domain.ChatStreamEvent) error {
		return domain.ErrAllEndpointsExhausted
	}, out)

	if err == nil {
		t.Fatal("expected the gateway error to be reported")
	}
	if sentAny {
		t.Fatal("nothing was streamed, so sentAny must be false (fallback allowed)")
	}
}

// TestForwardGatewayAttempt_ErrorAfterContentBlocksFallback verifies the rule
// that matters for UX: once the user has seen output, we must not restart on
// another transport (it would duplicate/contradict what is on screen).
func TestForwardGatewayAttempt_ErrorAfterContentBlocksFallback(t *testing.T) {
	out := make(chan domain.ChatStreamEvent, 8)
	sentAny, err := forwardGatewayAttempt(func(ch chan domain.ChatStreamEvent) error {
		ch <- domain.ChatStreamEvent{Type: "content", Content: "sebagian"}
		return domain.ErrAllEndpointsExhausted
	}, out)

	if err == nil {
		t.Fatal("expected the error to be reported")
	}
	if !sentAny {
		t.Fatal("content was streamed, so sentAny must be true (no fallback)")
	}
}

// TestForwardGatewayAttempt_EmptySuccessIsTreatedAsFailure verifies that a
// gateway returning "success" with no content still triggers the fallback — an
// empty answer is a failure to the user.
func TestForwardGatewayAttempt_EmptySuccessIsTreatedAsFailure(t *testing.T) {
	out := make(chan domain.ChatStreamEvent, 8)
	sentAny, err := forwardGatewayAttempt(func(ch chan domain.ChatStreamEvent) error {
		ch <- domain.ChatStreamEvent{Type: "done", Done: true} // no content
		return nil
	}, out)

	if sentAny {
		t.Fatal("no content was produced")
	}
	if err != errGatewayEmpty {
		t.Fatalf("expected errGatewayEmpty, got %v", err)
	}
}

// TestForwardGatewayAttempt_IgnoresNonContentEvents verifies that tool/step
// events do not count as user-visible output (so the fallback still runs).
func TestForwardGatewayAttempt_IgnoresNonContentEvents(t *testing.T) {
	out := make(chan domain.ChatStreamEvent, 8)
	sentAny, err := forwardGatewayAttempt(func(ch chan domain.ChatStreamEvent) error {
		ch <- domain.ChatStreamEvent{Type: "agent_step", StepLabel: "Menganalisis..."}
		return domain.ErrAllEndpointsExhausted
	}, out)

	if sentAny {
		t.Fatal("agent_step is not user content; fallback must stay allowed")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}
