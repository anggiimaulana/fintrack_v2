package embedding

import (
	"context"
	"errors"
	"testing"
)

type stubEmbedder struct {
	calls   int
	enabled bool
}

func (s *stubEmbedder) Embed(context.Context, string) ([]float32, error) {
	s.calls++
	return make([]float32, ExpectedDimensions), nil
}

func (s *stubEmbedder) Enabled() bool { return s.enabled }

// TestKillSwitchEmbedder_FailsClosedWhileStopped covers the emergency stop on
// the direct (non-gateway) embedding transport.
func TestKillSwitchEmbedder_FailsClosedWhileStopped(t *testing.T) {
	inner := &stubEmbedder{enabled: true}
	stopped := true
	ks := NewKillSwitchEmbedder(inner, func() bool { return stopped })

	if ks.Enabled() {
		t.Fatal("expected Enabled() false while stopped")
	}
	if _, err := ks.Embed(context.Background(), "teks"); !errors.Is(err, ErrKillSwitch) {
		t.Fatalf("expected ErrKillSwitch, got %v", err)
	}
	if inner.calls != 0 {
		t.Fatalf("provider called %d times while stopped", inner.calls)
	}

	// Re-enabling must restore traffic immediately (live switch state).
	stopped = false
	if !ks.Enabled() {
		t.Fatal("expected Enabled() true after re-enable")
	}
	if _, err := ks.Embed(context.Background(), "teks"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", inner.calls)
	}
}
