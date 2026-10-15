package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

// VoiceTranscriptionRepo persists per-user voice history.
type VoiceTranscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewVoiceTranscriptionRepo(pool *pgxpool.Pool) *VoiceTranscriptionRepo {
	return &VoiceTranscriptionRepo{pool: pool}
}

// Create appends a transcription.
func (r *VoiceTranscriptionRepo) Create(ctx context.Context, t *domain.VoiceTranscription) error {
	if t.Language == "" {
		t.Language = "id"
	}
	return q(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO voice_transcriptions (user_id, transcript, language, mode, duration_ms, tokens_used)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		t.UserID, t.Transcript, t.Language, t.Mode, t.DurationMS, t.TokensUsed).
		Scan(&t.ID, &t.CreatedAt)
}

// ListByUser returns one page of the user's history plus the total count.
func (r *VoiceTranscriptionRepo) ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]domain.VoiceTranscription, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	var total int
	if err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT COUNT(*) FROM voice_transcriptions WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count voice transcriptions: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := q(ctx, r.pool).Query(ctx, `
		SELECT id, user_id, transcript, language, mode, duration_ms, tokens_used, created_at
		FROM voice_transcriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, userID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list voice transcriptions: %w", err)
	}
	defer rows.Close()

	out := []domain.VoiceTranscription{}
	for rows.Next() {
		var t domain.VoiceTranscription
		if err := rows.Scan(&t.ID, &t.UserID, &t.Transcript, &t.Language, &t.Mode,
			&t.DurationMS, &t.TokensUsed, &t.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan voice transcription: %w", err)
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

// CountSince counts the user's transcriptions created at or after `since`.
//
// This is the daily-limit counter. It counts ROWS THAT EXIST, so a user who
// deletes their history would appear to have used less — which is why the
// caller computes `since` from the start of the day and the counter is treated
// as a best-effort guard: deleting history does not refund the day (the row was
// already created today). See the R-14 note in the design docs.
func (r *VoiceTranscriptionRepo) CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	var n int64
	err := q(ctx, r.pool).QueryRow(ctx, `
		SELECT COUNT(*) FROM voice_transcriptions
		WHERE user_id = $1 AND created_at >= $2`, userID, since).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count voice transcriptions since: %w", err)
	}
	return n, nil
}

// DeleteOne removes one transcription owned by the user.
func (r *VoiceTranscriptionRepo) DeleteOne(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := q(ctx, r.pool).Exec(ctx,
		`DELETE FROM voice_transcriptions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete voice transcription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ClearByUser removes every transcription owned by the user.
func (r *VoiceTranscriptionRepo) ClearByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`DELETE FROM voice_transcriptions WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("clear voice transcriptions: %w", err)
	}
	return nil
}

// DeleteOlderThan prunes rows created before cutoff across all users. Used by
// the retention worker; returns the number of rows removed.
func (r *VoiceTranscriptionRepo) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := q(ctx, r.pool).Exec(ctx,
		`DELETE FROM voice_transcriptions WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune voice transcriptions: %w", err)
	}
	return tag.RowsAffected(), nil
}

// IncrementDailyUsage atomically bumps the user's counter for `day` and returns
// the new value. The counter is separate from the history rows so deleting
// history cannot refund the day (R-14).
func (r *VoiceTranscriptionRepo) IncrementDailyUsage(ctx context.Context, userID uuid.UUID, day time.Time) (int64, error) {
	var used int
	err := q(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO voice_daily_usage (user_id, usage_day, used, updated_at)
		VALUES ($1, $2, 1, now())
		ON CONFLICT (user_id, usage_day) DO UPDATE
		SET used = voice_daily_usage.used + 1, updated_at = now()
		RETURNING used`,
		userID, day.Format("2006-01-02")).Scan(&used)
	if err != nil {
		return 0, fmt.Errorf("increment voice daily usage: %w", err)
	}
	return int64(used), nil
}

// DailyUsage returns the user's counter for `day` (0 when no row exists).
func (r *VoiceTranscriptionRepo) DailyUsage(ctx context.Context, userID uuid.UUID, day time.Time) (int64, error) {
	var used int64
	err := q(ctx, r.pool).QueryRow(ctx, `
		SELECT used FROM voice_daily_usage WHERE user_id = $1 AND usage_day = $2`,
		userID, day.Format("2006-01-02")).Scan(&used)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("read voice daily usage: %w", err)
	}
	return used, nil
}

// PruneDailyUsageBefore deletes counter rows for days strictly before
// `beforeDay`. Housekeeping only: the current day is never affected, so the
// daily limit stays correct. Keeps the table from growing without bound.
func (r *VoiceTranscriptionRepo) PruneDailyUsageBefore(ctx context.Context, beforeDay time.Time) (int64, error) {
	tag, err := q(ctx, r.pool).Exec(ctx,
		`DELETE FROM voice_daily_usage WHERE usage_day < $1`, beforeDay.Format("2006-01-02"))
	if err != nil {
		return 0, fmt.Errorf("prune voice daily usage: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ConsumeDailyUsage atomically increments the counter ONLY when it is still
// below `limit`, returning whether the slot was granted and the new value.
//
// This closes the TOCTOU race of a separate "check then increment": two
// concurrent requests can both pass a pre-check, but only `limit` conditional
// increments succeed, so the cap can never be exceeded (defense in depth on top
// of the middleware gate). A non-positive limit is treated as "no slots".
func (r *VoiceTranscriptionRepo) ConsumeDailyUsage(ctx context.Context, userID uuid.UUID, day time.Time, limit int64) (bool, int64, error) {
	if limit <= 0 {
		return false, 0, nil
	}
	var used int64
	err := q(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO voice_daily_usage (user_id, usage_day, used, updated_at)
		VALUES ($1, $2, 1, now())
		ON CONFLICT (user_id, usage_day) DO UPDATE
		SET used = voice_daily_usage.used + 1, updated_at = now()
		WHERE voice_daily_usage.used < $3
		RETURNING used`,
		userID, day.Format("2006-01-02"), limit).Scan(&used)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// The conditional update matched nothing → the cap is already reached.
			return false, limit, nil
		}
		return false, 0, fmt.Errorf("consume voice daily usage: %w", err)
	}
	return true, used, nil
}

// compile-time assertion.
var _ domain.VoiceTranscriptionRepository = (*VoiceTranscriptionRepo)(nil)
