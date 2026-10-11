package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

type RefreshTokenRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepo(pool *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{pool: pool}
}

const refreshTokenCols = "id, user_id, token_hash, expires_at, revoked_at, replaced_by, created_at, last_used_at, user_agent"

func scanRefreshToken(row interface {
	Scan(dest ...interface{}) error
}) (*domain.RefreshToken, error) {
	t := &domain.RefreshToken{}
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.ReplacedBy, &t.CreatedAt, &t.LastUsedAt, &t.UserAgent)
	return t, err
}

func (r *RefreshTokenRepo) Create(ctx context.Context, token *domain.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		token.UserID, token.TokenHash, token.ExpiresAt, token.UserAgent,
	).Scan(&token.ID, &token.CreatedAt)
}

func (r *RefreshTokenRepo) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	query := `SELECT ` + refreshTokenCols + ` FROM refresh_tokens WHERE token_hash = $1`
	t, err := scanRefreshToken(q(ctx, r.pool).QueryRow(ctx, query, hash))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return t, nil
}

// Rotate revokes the old token and inserts the new one in a single transaction.
// The old row is locked FOR UPDATE to serialize concurrent refreshes.
func (r *RefreshTokenRepo) Rotate(ctx context.Context, oldID uuid.UUID, newToken *domain.RefreshToken) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var revokedAt *time.Time
	if err := tx.QueryRow(ctx,
		`SELECT revoked_at FROM refresh_tokens WHERE id = $1 FOR UPDATE`, oldID,
	).Scan(&revokedAt); err != nil {
		return fmt.Errorf("failed to lock refresh token: %w", err)
	}

	if err := tx.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		newToken.UserID, newToken.TokenHash, newToken.ExpiresAt, newToken.UserAgent,
	).Scan(&newToken.ID, &newToken.CreatedAt); err != nil {
		return fmt.Errorf("failed to insert rotated token: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now(), replaced_by = $2, last_used_at = now() WHERE id = $1`,
		oldID, newToken.ID); err != nil {
		return fmt.Errorf("failed to revoke old token: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func (r *RefreshTokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := q(ctx, r.pool).Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < now()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
