package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// drainChatEvents collects every event from a chat stream channel.
func drainChatEvents(ch <-chan domain.ChatStreamEvent) []domain.ChatStreamEvent {
	var out []domain.ChatStreamEvent
	for ev := range ch {
		out = append(out, ev)
	}
	return out
}

// TestKillSwitch_StopsChatBeforeAnyWork verifies the emergency stop short
// circuits chat before touching repositories, so a zero-value usecase (nil
// repos) must not panic.
func TestKillSwitch_StopsChatBeforeAnyWork(t *testing.T) {
	u := (&AIUsecase{}).WithKillSwitch(func(string) bool { return true })

	ch := make(chan domain.ChatStreamEvent, 4)
	if err := u.Chat(context.Background(), uuid.New(), domain.ChatRequest{Message: "halo"}, nil, "web", ch); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	events := drainChatEvents(ch)
	if len(events) != 2 {
		t.Fatalf("expected exactly 2 events while stopped, got %d: %+v", len(events), events)
	}
	if !strings.Contains(events[0].Content, "dinonaktifkan") {
		t.Fatalf("expected disabled notice, got %q", events[0].Content)
	}
	if !events[1].Done {
		t.Fatalf("expected terminal done event, got %+v", events[1])
	}
}

// TestKillSwitch_StopsVisionAndVoice verifies both non-streaming modalities
// fail closed without reaching the provider.
func TestKillSwitch_StopsVisionAndVoice(t *testing.T) {
	gw := &fakeGateway{visionResponse: `{"merchant":"X","total":1,"type":"expense"}`}
	u := (&AIUsecase{}).WithGateway(gw).WithKillSwitch(func(string) bool { return true })

	vision, err := u.AnalyzeReceipt(context.Background(), uuid.New(), []byte("img"), "image/jpeg")
	if err != nil {
		t.Fatalf("vision err: %v", err)
	}
	if vision["success"] != false || !strings.Contains(vision["message"].(string), "dinonaktifkan") {
		t.Fatalf("expected disabled vision result, got %+v", vision)
	}

	voice, err := u.ProcessVoice(context.Background(), uuid.New(), "YXVkaW8=")
	if err != nil {
		t.Fatalf("voice err: %v", err)
	}
	if voice["success"] != false || !strings.Contains(voice["message"].(string), "dinonaktifkan") {
		t.Fatalf("expected disabled voice result, got %+v", voice)
	}

	if gw.lastVisionReq != nil || gw.lastVoiceReq != nil {
		t.Fatalf("provider reached while stopped: vision=%+v voice=%+v", gw.lastVisionReq, gw.lastVoiceReq)
	}
}

// TestKillSwitch_OffLeavesTrafficAlone guards against the guard itself
// swallowing requests once an operator re-enables a modality.
func TestKillSwitch_OffLeavesTrafficAlone(t *testing.T) {
	gw := &fakeGateway{visionResponse: `{"merchant":"Toko B","total":9000,"type":"expense"}`}
	u := (&AIUsecase{}).WithGateway(gw).WithKillSwitch(func(string) bool { return false })

	res, err := u.AnalyzeReceipt(context.Background(), uuid.New(), []byte("img"), "image/jpeg")
	if err != nil {
		t.Fatalf("vision err: %v", err)
	}
	if res["success"] != true || res["merchant"] != "Toko B" {
		t.Fatalf("expected gateway result, got %+v", res)
	}
	if gw.lastVisionReq == nil {
		t.Fatal("expected provider call when switch is off")
	}
}
