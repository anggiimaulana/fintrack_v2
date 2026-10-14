package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeDowngradeLister struct {
	due []uuid.UUID
	err error
}

func (f *fakeDowngradeLister) ListDueScheduledDowngrades(_ context.Context, _ time.Time) ([]uuid.UUID, error) {
	return f.due, f.err
}

// TestDowngradeWorker_AppliesDue verifies every due user is applied — F3/T-DWN-02.
func TestDowngradeWorker_AppliesDue(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	applied := map[uuid.UUID]int{}
	w := NewDowngradeWorker(
		&fakeDowngradeLister{due: []uuid.UUID{a, b}},
		func(_ context.Context, id uuid.UUID) error { applied[id]++; return nil },
		DowngradeConfig{},
	)
	w.RunOnce(context.Background())

	if applied[a] != 1 || applied[b] != 1 {
		t.Fatalf("expected each user applied once, got %v", applied)
	}
}

// TestDowngradeWorker_ContinuesAfterFailure verifies one failure does not abort
// the sweep (best-effort batch) — F3.
func TestDowngradeWorker_ContinuesAfterFailure(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	applied := map[uuid.UUID]int{}
	w := NewDowngradeWorker(
		&fakeDowngradeLister{due: []uuid.UUID{a, b}},
		func(_ context.Context, id uuid.UUID) error {
			applied[id]++
			if id == a {
				return errors.New("boom")
			}
			return nil
		},
		DowngradeConfig{},
	)
	w.RunOnce(context.Background())

	if applied[a] != 1 || applied[b] != 1 {
		t.Fatalf("expected both attempted, got %v", applied)
	}
}

// TestDowngradeWorker_ListErrorIsSafe verifies a list failure is a no-op.
func TestDowngradeWorker_ListErrorIsSafe(t *testing.T) {
	called := false
	w := NewDowngradeWorker(
		&fakeDowngradeLister{err: errors.New("db down")},
		func(_ context.Context, _ uuid.UUID) error { called = true; return nil },
		DowngradeConfig{},
	)
	w.RunOnce(context.Background())
	if called {
		t.Fatal("apply must not be called when listing fails")
	}
}

// TestDowngradeWorker_StartStopIsClean verifies the loop starts and stops without
// hanging (the default interval is long, so only RunOnce fires at startup).
func TestDowngradeWorker_StartStopIsClean(t *testing.T) {
	w := NewDowngradeWorker(
		&fakeDowngradeLister{},
		func(_ context.Context, _ uuid.UUID) error { return nil },
		DowngradeConfig{Interval: time.Hour},
	)
	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)
	cancel()
	w.Stop()
}
