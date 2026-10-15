package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

// seedUser inserts a throwaway user and returns its id (voice rows FK to users).
func seedUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	email := "voice-it-" + id.String() + "@example.com"
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, full_name, role) VALUES ($1, $2, $3, 'user')`,
		id, email, "Voice IT")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func TestVoiceRepo_CreateListCountDelete(t *testing.T) {
	pool := newTestPool(t)
	repo := NewVoiceTranscriptionRepo(pool)
	ctx := t.Context()

	userA := seedUser(t, ctx, pool)
	userB := seedUser(t, ctx, pool)

	// Two rows today for A, one for B.
	for range 2 {
		if err := repo.Create(ctx, &domain.VoiceTranscription{
			UserID: userA, Transcript: "halo", Language: "id", Mode: domain.VoiceModeServer,
		}); err != nil {
			t.Fatalf("create A: %v", err)
		}
	}
	if err := repo.Create(ctx, &domain.VoiceTranscription{
		UserID: userB, Transcript: "hai", Language: "id", Mode: domain.VoiceModeBrowser,
	}); err != nil {
		t.Fatalf("create B: %v", err)
	}

	// Isolation: each user lists only their own rows.
	listA, totalA, err := repo.ListByUser(ctx, userA, 1, 20)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	if totalA != 2 || len(listA) != 2 {
		t.Fatalf("user A: total=%d len=%d, want 2/2", totalA, len(listA))
	}
	for _, row := range listA {
		if row.UserID != userA {
			t.Fatalf("user A list leaked row owned by %s", row.UserID)
		}
	}

	// Daily count uses a lower bound.
	since := time.Now().UTC().Add(-time.Hour)
	countA, err := repo.CountSince(ctx, userA, since)
	if err != nil {
		t.Fatalf("count A: %v", err)
	}
	if countA != 2 {
		t.Fatalf("count A = %d, want 2", countA)
	}

	// A cannot delete B's row.
	if err := repo.DeleteOne(ctx, listA[0].ID, userB); err == nil {
		t.Fatal("user B should NOT be able to delete user A's row")
	}

	// A can delete its own row.
	if err := repo.DeleteOne(ctx, listA[0].ID, userA); err != nil {
		t.Fatalf("delete own row: %v", err)
	}

	// Clear removes the rest.
	if err := repo.ClearByUser(ctx, userA); err != nil {
		t.Fatalf("clear: %v", err)
	}
	_, totalAfter, err := repo.ListByUser(ctx, userA, 1, 20)
	if err != nil {
		t.Fatalf("list after clear: %v", err)
	}
	if totalAfter != 0 {
		t.Fatalf("after clear total = %d, want 0", totalAfter)
	}
}

func TestVoiceRepo_DailyCounterIsAppendOnly(t *testing.T) {
	pool := newTestPool(t)
	repo := NewVoiceTranscriptionRepo(pool)
	ctx := t.Context()

	user := seedUser(t, ctx, pool)
	day := time.Now().UTC()

	// No row yet → zero.
	got, err := repo.DailyUsage(ctx, user, day)
	if err != nil {
		t.Fatalf("initial daily usage: %v", err)
	}
	if got != 0 {
		t.Fatalf("initial usage = %d, want 0", got)
	}

	// Two increments → counter reaches 2.
	for want := int64(1); want <= 2; want++ {
		got, err = repo.IncrementDailyUsage(ctx, user, day)
		if err != nil {
			t.Fatalf("increment: %v", err)
		}
		if got != want {
			t.Fatalf("increment returned %d, want %d", got, want)
		}
	}

	// Deleting history must NOT refund the day (R-14): the counter is separate.
	if err := repo.Create(ctx, &domain.VoiceTranscription{
		UserID: user, Transcript: "x", Language: "id", Mode: domain.VoiceModeBrowser,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.ClearByUser(ctx, user); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, err = repo.DailyUsage(ctx, user, day)
	if err != nil {
		t.Fatalf("daily usage after clear: %v", err)
	}
	if got != 2 {
		t.Fatalf("after clearing history usage = %d, want 2 (append-only)", got)
	}

	// A different day starts at zero.
	otherDay := day.AddDate(0, 0, -1)
	got, err = repo.DailyUsage(ctx, user, otherDay)
	if err != nil {
		t.Fatalf("other day usage: %v", err)
	}
	if got != 0 {
		t.Fatalf("other day usage = %d, want 0", got)
	}
}

func TestVoiceRepo_PruneDailyUsageBefore(t *testing.T) {
	pool := newTestPool(t)
	repo := NewVoiceTranscriptionRepo(pool)
	ctx := t.Context()

	user := seedUser(t, ctx, pool)
	today := time.Now().UTC()
	yesterday := today.AddDate(0, 0, -1)
	old := today.AddDate(0, 0, -100)

	// Counters for today, yesterday, and long ago.
	for _, day := range []time.Time{today, yesterday, old} {
		if _, err := repo.IncrementDailyUsage(ctx, user, day); err != nil {
			t.Fatalf("increment %v: %v", day, err)
		}
	}

	// Prune everything older than 30 days: today and yesterday must survive.
	removed, err := repo.PruneDailyUsageBefore(ctx, today.AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if removed < 1 {
		t.Fatalf("pruned %d rows, want >= 1", removed)
	}

	// The current day's counter must be untouched (the gate reads it).
	got, err := repo.DailyUsage(ctx, user, today)
	if err != nil {
		t.Fatalf("today usage: %v", err)
	}
	if got != 1 {
		t.Fatalf("today usage = %d, want 1 (must not be pruned)", got)
	}
	// The old counter is gone.
	gotOld, err := repo.DailyUsage(ctx, user, old)
	if err != nil {
		t.Fatalf("old usage: %v", err)
	}
	if gotOld != 0 {
		t.Fatalf("old usage = %d, want 0 (pruned)", gotOld)
	}
}

func TestVoiceRepo_ConsumeDailyUsageIsRaceSafe(t *testing.T) {
	pool := newTestPool(t)
	repo := NewVoiceTranscriptionRepo(pool)
	ctx := t.Context()

	user := seedUser(t, ctx, pool)
	day := time.Now().UTC()
	const limit = 5
	const attempts = 40

	// Fire many concurrent claims against a small cap. Exactly `limit` must be
	// granted — no more — proving the conditional increment closes the TOCTOU
	// window that a separate check-then-increment would leave open.
	var granted int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _, err := repo.ConsumeDailyUsage(ctx, user, day, limit)
			if err != nil {
				t.Errorf("consume: %v", err)
				return
			}
			if ok {
				mu.Lock()
				granted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if granted != limit {
		t.Fatalf("granted %d slots, want exactly %d (cap must not be exceeded)", granted, limit)
	}
	used, err := repo.DailyUsage(ctx, user, day)
	if err != nil {
		t.Fatalf("daily usage: %v", err)
	}
	if used != limit {
		t.Fatalf("counter = %d, want %d", used, limit)
	}
}

func TestVoiceRepo_ConsumeDailyUsageRejectsNonPositiveLimit(t *testing.T) {
	pool := newTestPool(t)
	repo := NewVoiceTranscriptionRepo(pool)
	ctx := t.Context()
	user := seedUser(t, ctx, pool)

	ok, _, err := repo.ConsumeDailyUsage(ctx, user, time.Now().UTC(), 0)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if ok {
		t.Fatal("a non-positive limit must grant nothing")
	}
}

func TestVoiceRepo_DeleteOlderThan(t *testing.T) {
	pool := newTestPool(t)
	repo := NewVoiceTranscriptionRepo(pool)
	ctx := t.Context()

	user := seedUser(t, ctx, pool)
	old := &domain.VoiceTranscription{UserID: user, Transcript: "old", Language: "id", Mode: domain.VoiceModeServer}
	fresh := &domain.VoiceTranscription{UserID: user, Transcript: "fresh", Language: "id", Mode: domain.VoiceModeBrowser}
	if err := repo.Create(ctx, old); err != nil {
		t.Fatalf("create old: %v", err)
	}
	if err := repo.Create(ctx, fresh); err != nil {
		t.Fatalf("create fresh: %v", err)
	}
	// Backdate the first row well beyond any retention window.
	if _, err := pool.Exec(ctx,
		`UPDATE voice_transcriptions SET created_at = now() - interval '400 days' WHERE id = $1`, old.ID); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	removed, err := repo.DeleteOlderThan(ctx, time.Now().UTC().AddDate(0, 0, -90))
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if removed < 1 {
		t.Fatalf("pruned %d rows, want >= 1", removed)
	}

	list, total, err := repo.ListByUser(ctx, user, 1, 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].Transcript != "fresh" {
		t.Fatalf("after prune: total=%d rows=%v, want only 'fresh'", total, list)
	}
}
