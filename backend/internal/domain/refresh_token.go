package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RefreshToken is a persisted, rotatable refresh token. Only the sha256 hash of
// the raw token is stored (token_hash). A rotated token records revoked_at and
// replaced_by for reuse detection.
type RefreshToken struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash  string     `json:"-" db:"token_hash"`
	ExpiresAt  time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	ReplacedBy *uuid.UUID `json:"replaced_by,omitempty" db:"replaced_by"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	UserAgent  *string    `json:"user_agent,omitempty" db:"user_agent"`
}

// RefreshTokenRepository persists and rotates refresh tokens.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	// GetByHash returns the token row for the given hash, or nil if absent.
	GetByHash(ctx context.Context, hash string) (*RefreshToken, error)
	// Rotate marks oldID revoked, links it to the newly created token, and
	// persists newToken — all in a single transaction.
	Rotate(ctx context.Context, oldID uuid.UUID, newToken *RefreshToken) error
	// RevokeAllForUser revokes every active token for a user (reuse response).
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	// DeleteExpired removes tokens whose expiry has passed.
	DeleteExpired(ctx context.Context) (int64, error)
}
