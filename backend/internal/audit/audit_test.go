package audit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeRepo captures inserted batches for assertions.
type fakeRepo struct {
	mu      sync.Mutex
	batches [][]Event
	delay   time.Duration
}

func (f *fakeRepo) InsertBatch(_ context.Context, events []Event) error {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]Event, len(events))
	copy(cp, events)
	f.batches = append(f.batches, cp)
	return nil
}

func (f *fakeRepo) all() []Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Event
	for _, b := range f.batches {
		out = append(out, b...)
	}
	return out
}

// TestService_LogIsNonBlocking asserts Log returns quickly even with a slow repo.
func TestService_LogIsNonBlocking(t *testing.T) {
	repo := &fakeRepo{delay: 50 * time.Millisecond}
	svc := NewService(repo, Config{Buffer: 64, BatchSize: 100, FlushInterval: 10 * time.Millisecond})
	defer svc.Close()

	start := time.Now()
	for range 10 {
		svc.Log(context.Background(), Event{Action: "create", Entity: "plan"})
	}
	elapsed := time.Since(start)
	if elapsed > 10*time.Millisecond {
		t.Fatalf("Log should be non-blocking, took %v", elapsed)
	}
}

// TestService_FlushesOnClose asserts events are persisted before Close returns.
func TestService_FlushesOnClose(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, Config{Buffer: 16, BatchSize: 100, FlushInterval: time.Hour})

	for range 5 {
		svc.Log(context.Background(), Event{Action: "update", Entity: "user"})
	}
	svc.Close()

	got := repo.all()
	if len(got) != 5 {
		t.Fatalf("expected 5 persisted events, got %d", len(got))
	}
}

// TestService_Batches asserts events are grouped into batches of BatchSize.
func TestService_Batches(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, Config{Buffer: 64, BatchSize: 3, FlushInterval: 5 * time.Millisecond})

	for range 7 {
		svc.Log(context.Background(), Event{Action: "a", Entity: "e"})
	}
	svc.Close()

	got := repo.all()
	if len(got) != 7 {
		t.Fatalf("expected 7 persisted events, got %d", len(got))
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	// At least one batch should be exactly the batch size (3).
	foundFull := false
	for _, b := range repo.batches {
		if len(b) >= 3 {
			foundFull = true
		}
	}
	if !foundFull {
		t.Fatal("expected at least one batch of size >= 3")
	}
}

// TestService_MasksOnLog asserts masking is applied before persistence.
func TestService_MasksOnLog(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, Config{Buffer: 8, BatchSize: 1, FlushInterval: time.Hour})

	svc.Log(context.Background(), Event{
		Action: "create",
		Entity: "gemini_key",
		After:  map[string]any{"alias": "k1", "api_key": "sk-secret"},
	})
	svc.Close()

	got := repo.all()
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].After["api_key"] != redacted {
		t.Fatalf("api_key should be masked, got %v", got[0].After["api_key"])
	}
	if got[0].After["alias"] != "k1" {
		t.Fatalf("alias should be preserved, got %v", got[0].After["alias"])
	}
	if got[0].CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
}

// TestService_ActorIDPassthrough sanity-checks uuid handling.
func TestService_ActorIDPassthrough(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, Config{Buffer: 4, BatchSize: 1, FlushInterval: time.Hour})
	id := uuid.New()
	svc.Log(context.Background(), Event{ActorID: &id, Action: "x", Entity: "y"})
	svc.Close()
	got := repo.all()
	if len(got) != 1 || got[0].ActorID == nil || *got[0].ActorID != id {
		t.Fatal("actor id should be preserved")
	}
}
