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
	// servedModel is stamped on the terminal chunk, mirroring a real upstream
	// reporting which model produced the reply.
	servedModel string
	// reasoningErr, when set, fails only requests tagged with the reasoning use
	// case — used to exercise the chat-pool fallback.
	reasoningErr error

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
	if f.reasoningErr != nil && req.UseCase == domain.UseCaseReasoning {
		return nil, f.reasoningErr
	}
	if f.chatErr != nil {
		return nil, f.chatErr
	}
	ch := make(chan domain.AIChatStreamChunk, 2)
	go func() {
		defer close(ch)
		ch <- domain.AIChatStreamChunk{Content: f.chatResponse}
		ch <- domain.AIChatStreamChunk{Done: true, Model: f.servedModel, Usage: domain.TokenUsage{TotalTokens: 5, Source: "provider_reported"}}
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

	res, _, err := u.analyzeReceiptViaGateway(context.Background(), []byte("fakeimg"), "image/jpeg")
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

	res, _, err := u.analyzeReceiptViaGateway(context.Background(), []byte("img"), "image/jpeg")
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

	res, _, err := u.processVoiceViaGateway(context.Background(), "aGVsbG8=") // base64 "hello"
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
	res, _, err := u.processVoiceViaGateway(context.Background(), "!!!not-base64!!!")
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

// TestGatewayResultFailed pins the fallback trigger for vision/voice, whose
// adapters swallow upstream errors into a map rather than returning an error.
func TestGatewayResultFailed(t *testing.T) {
	cases := []struct {
		name string
		res  map[string]interface{}
		want bool
	}{
		{"nil map counts as failure", nil, true},
		{"success true passes", map[string]interface{}{"success": true}, false},
		{"success false fails", map[string]interface{}{"success": false}, true},
		{"missing success fails closed", map[string]interface{}{"message": "x"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := gatewayResultFailed(tc.res); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAnalyzeReceiptViaGateway_ErrorIsDetectable verifies that a gateway vision
// failure surfaces as success=false, which is what the caller uses to decide to
// fall back to BYOK.
func TestAnalyzeReceiptViaGateway_ErrorIsDetectable(t *testing.T) {
	gw := &fakeGateway{visionErr: domain.ErrAllEndpointsExhausted}
	u := (&AIUsecase{}).WithGateway(gw)

	res, _, err := u.analyzeReceiptViaGateway(context.Background(), []byte("img"), "image/jpeg")
	if err != nil {
		t.Fatalf("adapter should not return a raw error, got %v", err)
	}
	if !gatewayResultFailed(res) {
		t.Fatalf("a failed gateway vision result must be detectable for fallback: %+v", res)
	}
}

// TestProcessVoiceViaGateway_ErrorIsDetectable is the voice counterpart.
func TestProcessVoiceViaGateway_ErrorIsDetectable(t *testing.T) {
	gw := &fakeGateway{voiceErr: domain.ErrAllEndpointsExhausted}
	u := (&AIUsecase{}).WithGateway(gw)

	res, _, err := u.processVoiceViaGateway(context.Background(), "aGVsbG8=")
	if err != nil {
		t.Fatalf("adapter should not return a raw error, got %v", err)
	}
	if !gatewayResultFailed(res) {
		t.Fatalf("a failed gateway voice result must be detectable for fallback: %+v", res)
	}
}

// TestGatewayVisionSuccessIsNotTreatedAsFailure guards against the fallback
// firing on a healthy gateway response.
func TestGatewayVisionSuccessIsNotTreatedAsFailure(t *testing.T) {
	gw := &fakeGateway{visionResponse: `{"merchant":"Toko A","total":50000,"category":"Makanan","type":"expense","items":[]}`}
	u := (&AIUsecase{}).WithGateway(gw)

	res, _, err := u.analyzeReceiptViaGateway(context.Background(), []byte("img"), "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gatewayResultFailed(res) {
		t.Fatalf("a successful gateway response must NOT trigger the fallback: %+v", res)
	}
}

// ─── Mode routing (fast vs reasoning) ──────────────────────────

// TestGatewayUseCaseForMode pins the mapping that makes the user's CEPAT /
// PENALARAN choice take effect: reasoning must hit the reasoning route pool,
// everything else the standard chat pool.
func TestGatewayUseCaseForMode(t *testing.T) {
	cases := []struct {
		mode domain.LLMMode
		want domain.UseCase
	}{
		{domain.LLMReasoning, domain.UseCaseReasoning},
		{domain.LLMFast, domain.UseCaseChat},
		{"", domain.UseCaseChat},
		{"unknown", domain.UseCaseChat},
	}
	for _, tc := range cases {
		if got := gatewayUseCaseForMode(tc.mode); got != tc.want {
			t.Errorf("gatewayUseCaseForMode(%q) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

// TestReasoningEffortForMode_OnlyForReasoning verifies that the extra
// `reasoning_effort` field is only ever attached to reasoning requests, and only
// when explicitly opted in via the environment — so providers that reject the
// field keep working for fast mode.
func TestReasoningEffortForMode_OnlyForReasoning(t *testing.T) {
	t.Setenv("AI_REASONING_EFFORT", "")

	if got := reasoningEffortForMode(domain.LLMReasoning); got != "" {
		t.Fatalf("expected empty effort when env unset, got %q", got)
	}
	if got := reasoningEffortForMode(domain.LLMFast); got != "" {
		t.Fatalf("fast mode must never send reasoning_effort, got %q", got)
	}

	t.Setenv("AI_REASONING_EFFORT", "high")
	if got := reasoningEffortForMode(domain.LLMReasoning); got != "high" {
		t.Fatalf("expected opt-in effort for reasoning, got %q", got)
	}
	if got := reasoningEffortForMode(domain.LLMFast); got != "" {
		t.Fatalf("fast mode must stay empty even when env set, got %q", got)
	}
}

// TestGatewayStream_RoutesReasoningAndCapturesModel is the end-to-end adapter
// check: a reasoning request must reach the gateway tagged as the reasoning use
// case, and the model reported on the terminal chunk must be captured.
func TestGatewayStream_RoutesReasoningAndCapturesModel(t *testing.T) {
	t.Setenv("AI_REASONING_EFFORT", "medium")
	gw := &fakeGateway{chatResponse: "jawaban", servedModel: "deepseek-r1"}
	u := (&AIUsecase{}).WithGateway(gw)

	var served string
	got, err := u.gatewayStream(context.Background(), domain.UseCaseReasoning, reasoningEffortForMode(domain.LLMReasoning),
		"sys", nil, "hai", func(string) {}, &served, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "jawaban" {
		t.Fatalf("expected streamed text, got %q", got)
	}
	if served != "deepseek-r1" {
		t.Fatalf("expected served model captured, got %q", served)
	}
	if gw.lastChatReq == nil || gw.lastChatReq.UseCase != domain.UseCaseReasoning {
		t.Fatalf("expected reasoning use case forwarded, got %+v", gw.lastChatReq)
	}
	if gw.lastChatReq.ReasoningEffort != "medium" {
		t.Fatalf("expected reasoning effort forwarded, got %q", gw.lastChatReq.ReasoningEffort)
	}
}

// TestGatewayStream_FastModeSendsNoReasoningEffort guards the fast path against
// regressions that would leak reasoning parameters into ordinary chat.
func TestGatewayStream_FastModeSendsNoReasoningEffort(t *testing.T) {
	t.Setenv("AI_REASONING_EFFORT", "high")
	gw := &fakeGateway{chatResponse: "ok", servedModel: "gemini-2.5-flash"}
	u := (&AIUsecase{}).WithGateway(gw)

	var served string
	if _, err := u.gatewayStream(context.Background(), domain.UseCaseChat, reasoningEffortForMode(domain.LLMFast),
		"sys", nil, "hai", func(string) {}, &served, nil); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gw.lastChatReq == nil || gw.lastChatReq.UseCase != domain.UseCaseChat {
		t.Fatalf("expected chat use case, got %+v", gw.lastChatReq)
	}
	if gw.lastChatReq.ReasoningEffort != "" {
		t.Fatalf("fast mode must not forward reasoning effort, got %q", gw.lastChatReq.ReasoningEffort)
	}
}

// TestGatewayStream_ReasoningFallsBackToChatPool verifies graceful degradation:
// if the reasoning pool cannot serve the request, the turn still completes via
// the standard chat pool (and without leaking reasoning_effort).
func TestGatewayStream_ReasoningFallsBackToChatPool(t *testing.T) {
	t.Setenv("AI_REASONING_EFFORT", "high")
	gw := &fakeGateway{chatResponse: "fallback ok", servedModel: "gemini-2.5-flash", reasoningErr: domain.ErrNoRoute}
	u := (&AIUsecase{}).WithGateway(gw)

	var served string
	got, err := u.gatewayStream(context.Background(), domain.UseCaseReasoning, reasoningEffortForMode(domain.LLMReasoning),
		"sys", nil, "hai", func(string) {}, &served, nil)
	if err != nil {
		t.Fatalf("expected graceful fallback, got err: %v", err)
	}
	if got != "fallback ok" {
		t.Fatalf("expected chat-pool answer, got %q", got)
	}
	// The fallback call must be tagged as chat and must not carry reasoning effort.
	if gw.lastChatReq == nil || gw.lastChatReq.UseCase != domain.UseCaseChat {
		t.Fatalf("expected fallback to chat use case, got %+v", gw.lastChatReq)
	}
	if gw.lastChatReq.ReasoningEffort != "" {
		t.Fatalf("fallback must drop reasoning_effort, got %q", gw.lastChatReq.ReasoningEffort)
	}
}
