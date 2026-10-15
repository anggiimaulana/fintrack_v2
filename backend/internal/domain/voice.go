package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Voice transcription modes. `server` = transcribed via the AI gateway
// (/ai/voice); `browser` = transcribed on-device via the Web Speech API.
const (
	VoiceModeServer  = "server"
	VoiceModeBrowser = "browser"
)

// VoiceTranscription is one recorded transcription, owned by a user.
//
// Transcript holds the ORIGINAL text produced by the transcriber, before any
// user edit (an edited message is a chat message, not a voice record). This is
// what the daily-limit counter and the audit trail are based on.
type VoiceTranscription struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Transcript string    `json:"transcript" db:"transcript"`
	Language   string    `json:"language" db:"language"`
	Mode       string    `json:"mode" db:"mode"`
	DurationMS int       `json:"duration_ms" db:"duration_ms"`
	TokensUsed int64     `json:"tokens_used" db:"tokens_used"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// RecordVoiceTranscriptionRequest is the write shape for a browser-mode
// transcription (server-mode rows are written by the /ai/voice handler).
type RecordVoiceTranscriptionRequest struct {
	Transcript string `json:"transcript" binding:"required"`
	Language   string `json:"language"`
	Mode       string `json:"mode"`
	DurationMS int    `json:"duration_ms"`
}

// VoiceTranscriptionRepository persists per-user voice history. Every method is
// scoped by user_id so one user can never read or mutate another's history.
type VoiceTranscriptionRepository interface {
	// Create appends a transcription for the user.
	Create(ctx context.Context, t *VoiceTranscription) error
	// ListByUser returns one page of the user's history, newest first, plus the
	// total count.
	ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]VoiceTranscription, int, error)
	// CountSince returns how many transcriptions the user has created at or
	// after `since` (used for the daily limit).
	CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error)
	// DeleteOne removes one transcription owned by the user.
	DeleteOne(ctx context.Context, id, userID uuid.UUID) error
	// ClearByUser removes every transcription owned by the user.
	ClearByUser(ctx context.Context, userID uuid.UUID) error
	// DeleteOlderThan prunes rows created before `cutoff` across all users
	// (retention worker; not user-scoped by design).
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)

	// IncrementDailyUsage atomically bumps the user's counter for the given day
	// and returns the new total. It is append-only: the user cannot reset it.
	IncrementDailyUsage(ctx context.Context, userID uuid.UUID, day time.Time) (int64, error)
	// DailyUsage returns the user's counter for the given day (0 when absent).
	DailyUsage(ctx context.Context, userID uuid.UUID, day time.Time) (int64, error)
	// PruneDailyUsageBefore deletes counter rows for days strictly before
	// `beforeDay` across all users (housekeeping; keeps the table bounded).
	PruneDailyUsageBefore(ctx context.Context, beforeDay time.Time) (int64, error)
	// ConsumeDailyUsage atomically claims one daily slot iff used < limit,
	// returning (granted, newValue). Closes the check-then-increment race.
	ConsumeDailyUsage(ctx context.Context, userID uuid.UUID, day time.Time, limit int64) (bool, int64, error)
}
