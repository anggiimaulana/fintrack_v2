package usecase

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestSPKQueueDedupe enqueues the same user 20 times before workers start and
// asserts only a single job runs (dedupe per user).
func TestSPKQueueDedupe(t *testing.T) {
	uid := uuid.New()

	var mu sync.Mutex
	runs := 0
	q := NewSPKQueue(4, func(_ context.Context, got uuid.UUID) {
		if got != uid {
			t.Errorf("job for wrong user: %v", got)
		}
		mu.Lock()
		runs++
		mu.Unlock()
	})

	// All enqueues happen while the queue is not started, so `pending` is set
	// on the first call and every later call for the same user is skipped.
	for range 20 {
		q.Enqueue(uid)
	}

	q.Start(t.Context())

	// Give workers a moment to drain, then stop (Stop waits for completion).
	time.Sleep(100 * time.Millisecond)
	q.Stop()

	mu.Lock()
	defer mu.Unlock()
	if runs != 1 {
		t.Fatalf("expected exactly 1 run for 20 enqueues of the same user, got %d", runs)
	}
}

// TestSPKQueueDistinctUsers ensures different users are each processed once.
func TestSPKQueueDistinctUsers(t *testing.T) {
	var mu sync.Mutex
	seen := map[uuid.UUID]int{}
	q := NewSPKQueue(4, func(_ context.Context, uid uuid.UUID) {
		mu.Lock()
		seen[uid]++
		mu.Unlock()
	})

	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, id := range ids {
		q.Enqueue(id)
	}

	q.Start(t.Context())
	time.Sleep(100 * time.Millisecond)
	q.Stop()

	mu.Lock()
	defer mu.Unlock()
	for _, id := range ids {
		if seen[id] != 1 {
			t.Errorf("user %v processed %d times, want 1", id, seen[id])
		}
	}
}
