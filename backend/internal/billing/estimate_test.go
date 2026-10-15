package billing

import (
	"context"
	"strconv"
	"testing"
)

type stubSettings struct {
	values  map[string]int64
	strings map[string]string
}

func (s stubSettings) GetInt(_ context.Context, key string) (int64, error) {
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	if raw, ok := s.strings[key]; ok {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return n, nil
		}
	}
	return 0, nil
}

func (s stubSettings) GetString(_ context.Context, key string) (string, error) {
	if v, ok := s.strings[key]; ok {
		return v, nil
	}
	return "", nil
}

func TestForChat_ScalesWithPromptLength(t *testing.T) {
	store := NewEstimateStore(stubSettings{values: map[string]int64{"ai_estimate_chat": 2000}})
	ctx := context.Background()

	short := store.ForChat(ctx, ModalityChat, 20)
	long := store.ForChat(ctx, ModalityChat, 8000)

	if long <= short {
		t.Fatalf("long prompt must reserve more than a short one: short=%d long=%d", short, long)
	}
	// The fixed overhead alone must exceed the old flat default, otherwise one
	// chat can still empty a Free cycle.
	if short < chatOverheadTokens {
		t.Fatalf("reserve must cover the fixed prompt overhead: got %d want >= %d", short, chatOverheadTokens)
	}
}

func TestForChat_NeverBelowConfiguredFloor(t *testing.T) {
	store := NewEstimateStore(stubSettings{values: map[string]int64{"ai_estimate_chat": 50000}})
	got := store.ForChat(context.Background(), ModalityChat, 10)
	if got != 50000 {
		t.Fatalf("configured estimate acts as the floor: got %d want 50000", got)
	}
}

func TestForChat_ReasoningUsesItsOwnFloor(t *testing.T) {
	store := NewEstimateStore(stubSettings{values: map[string]int64{
		"ai_estimate_chat":      2000,
		"ai_estimate_reasoning": 4000,
	}})
	ctx := context.Background()
	chat := store.ForChat(ctx, ModalityChat, 0)
	reasoning := store.ForChat(ctx, ModalityReasoning, 0)
	if reasoning <= chat {
		t.Fatalf("reasoning must reserve more than chat: chat=%d reasoning=%d", chat, reasoning)
	}
}

func TestForChat_NegativeCharsIsSafe(t *testing.T) {
	store := NewEstimateStore(stubSettings{})
	got := store.ForChat(context.Background(), ModalityChat, -100)
	if got < 1 {
		t.Fatalf("reserve must stay positive: got %d", got)
	}
}

func TestForVoice_ScalesWithAudioLength(t *testing.T) {
	store := NewEstimateStore(stubSettings{})
	ctx := context.Background()

	// 2 seconds of audio (8,000 bytes at 4000 B/s) -> 2 * 25 + 50 + 150 = 250 tokens
	short := store.ForVoice(ctx, 8000)
	// 30 seconds of audio (120,000 bytes at 4000 B/s) -> 30 * 25 + 50 + 150 = 950 tokens
	long := store.ForVoice(ctx, 120000)

	if long <= short {
		t.Fatalf("long audio must reserve more than short audio: short=%d long=%d", short, long)
	}
	if short > 500 {
		t.Fatalf("short 2s audio should reserve ~250 tokens, not flat 2000: got %d", short)
	}
}

func TestForVoice_NegativeBytesIsSafe(t *testing.T) {
	store := NewEstimateStore(stubSettings{})
	got := store.ForVoice(context.Background(), -500)
	if got < 1 {
		t.Fatalf("voice reserve must stay positive: got %d", got)
	}
}

func TestForVoice_RespectsConfiguredFloor(t *testing.T) {
	// An operator raising `ai_estimate_voice` must raise the minimum hold even
	// for a tiny clip, mirroring ForChat's floor semantics.
	store := NewEstimateStore(stubSettings{strings: map[string]string{"ai_estimate_voice": "9000"}})
	got := store.ForVoice(context.Background(), 4000) // 1s -> ~225 tokens
	if got != 9000 {
		t.Fatalf("configured floor must win for a small clip: got %d want 9000", got)
	}
}

func TestForVision_RespectsConfiguredFloor(t *testing.T) {
	store := NewEstimateStore(stubSettings{strings: map[string]string{"ai_estimate_vision": "12000"}})
	got := store.ForVision(context.Background(), 1000, 10)
	if got != 12000 {
		t.Fatalf("configured floor must win: got %d want 12000", got)
	}
}

func TestForVision_ScalesWithPromptAndSize(t *testing.T) {
	store := NewEstimateStore(stubSettings{})
	ctx := context.Background()

	// Normal receipt: 200KB image, 400 char prompt (100 tokens + 800 image + 400 headroom = 1300)
	normal := store.ForVision(ctx, 200000, 400)
	// Large high-res receipt: 1MB image, 400 char prompt (+400 extra large image tokens = 1700)
	large := store.ForVision(ctx, 1000000, 400)

	if large <= normal {
		t.Fatalf("large image must reserve more tokens than normal image: normal=%d large=%d", normal, large)
	}

	// Many categories in prompt (2000 chars -> 500 tokens)
	manyCats := store.ForVision(ctx, 200000, 2000)
	if manyCats <= normal {
		t.Fatalf("longer prompt with more categories must reserve more tokens: normal=%d manyCats=%d", normal, manyCats)
	}
}

func TestForVision_NegativeInputsIsSafe(t *testing.T) {
	store := NewEstimateStore(stubSettings{})
	got := store.ForVision(context.Background(), -100, -50)
	if got < 100 {
		t.Fatalf("vision reserve must stay >= floor: got %d", got)
	}
}
