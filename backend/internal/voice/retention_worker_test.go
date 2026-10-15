package voice

import (
	"context"
	"errors"
	"testing"
	"time"
)

// capturePruner records the cutoff it was called with and returns a canned count.
type capturePruner struct {
	called     bool
	cutoff     time.Time
	removed    int64
	err        error
	prunedDay  time.Time
	dailyCalls int
	dailyErr   error
}

func (c *capturePruner) DeleteOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	c.called = true
	c.cutoff = cutoff
	return c.removed, c.err
}

func (c *capturePruner) PruneDailyUsageBefore(_ context.Context, beforeDay time.Time) (int64, error) {
	c.dailyCalls++
	c.prunedDay = beforeDay
	return 0, c.dailyErr
}

func TestRetentionWorker_PrunesOlderThanWindow(t *testing.T) {
	pruner := &capturePruner{removed: 7}
	w := NewRetentionWorker(pruner, func(context.Context) int { return 90 }, RetentionConfig{}).
		WithClock(func() time.Time { return time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC) })

	w.RunOnce(context.Background())

	if !pruner.called {
		t.Fatal("pruner was not called")
	}
	want := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) // 90 days before 2026-10-15
	if !pruner.cutoff.Equal(want) {
		t.Errorf("cutoff = %v, want %v", pruner.cutoff, want)
	}
	// The daily-counter table must be pruned on the same sweep.
	if pruner.dailyCalls != 1 {
		t.Errorf("daily prune called %d times, want 1", pruner.dailyCalls)
	}
	if !pruner.prunedDay.Equal(want) {
		t.Errorf("daily prune cutoff = %v, want %v", pruner.prunedDay, want)
	}
}

func TestRetentionWorker_SkipsNonPositiveWindow(t *testing.T) {
	pruner := &capturePruner{}
	w := NewRetentionWorker(pruner, func(context.Context) int { return 0 }, RetentionConfig{})
	w.RunOnce(context.Background())
	if pruner.called {
		t.Error("pruner must NOT run on a non-positive window (would wipe the table)")
	}
	if pruner.dailyCalls != 0 {
		t.Error("daily prune must NOT run on a non-positive window")
	}
}

func TestRetentionWorker_NilDepsAreSafe(t *testing.T) {
	// Must not panic when constructed without a pruner/resolver.
	NewRetentionWorker(nil, nil, RetentionConfig{}).RunOnce(context.Background())
}

func TestRetentionWorker_SweepErrorDoesNotPanic(t *testing.T) {
	pruner := &capturePruner{err: errors.New("boom")}
	w := NewRetentionWorker(pruner, func(context.Context) int { return 30 }, RetentionConfig{})
	w.RunOnce(context.Background()) // must swallow the error
	if !pruner.called {
		t.Error("pruner should have been attempted")
	}
}

func TestRetentionWorker_DefaultInterval(t *testing.T) {
	w := NewRetentionWorker(&capturePruner{}, func(context.Context) int { return 1 }, RetentionConfig{})
	if w.interval != 6*time.Hour {
		t.Errorf("default interval = %v, want 6h", w.interval)
	}
}
