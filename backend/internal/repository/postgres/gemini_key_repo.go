package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

type GeminiKeyRepo struct {
	pool *pgxpool.Pool
}

func NewGeminiKeyRepo(pool *pgxpool.Pool) *GeminiKeyRepo {
	return &GeminiKeyRepo{pool: pool}
}

const geminiKeyCols = "id, user_id, alias, provider, model, encrypted_key, key_preview, priority, status, supports_text, supports_audio, supports_vision, last_used_at, last_error, created_at, updated_at"

func scanGeminiKey(row interface {
	Scan(dest ...interface{}) error
}) (domain.GeminiAPIKey, error) {
	var k domain.GeminiAPIKey
	err := row.Scan(
		&k.ID, &k.UserID, &k.Alias, &k.Provider, &k.Model, &k.EncryptedKey, &k.KeyPreview,
		&k.Priority, &k.Status, &k.SupportsText, &k.SupportsAudio, &k.SupportsVision,
		&k.LastUsedAt, &k.LastError, &k.CreatedAt, &k.UpdatedAt,
	)
	return k, err
}

func (r *GeminiKeyRepo) Create(ctx context.Context, key *domain.GeminiAPIKey) error {
	query := `
		INSERT INTO gemini_api_keys (user_id, alias, provider, model, encrypted_key, key_preview, priority, status, supports_text, supports_audio, supports_vision)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING ` + geminiKeyCols
	k, err := scanGeminiKey(q(ctx, r.pool).QueryRow(ctx, query,
		key.UserID, key.Alias, key.Provider, key.Model, key.EncryptedKey, key.KeyPreview, key.Priority, key.Status,
		key.SupportsText, key.SupportsAudio, key.SupportsVision,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: %s", domain.ErrDuplicateAlias, key.Alias)
		}
		return err
	}
	*key = k
	return nil
}

func (r *GeminiKeyRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.GeminiAPIKey, error) {
	query := `SELECT ` + geminiKeyCols + ` FROM gemini_api_keys WHERE id = $1 AND user_id = $2`
	k, err := scanGeminiKey(q(ctx, r.pool).QueryRow(ctx, query, id, userID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get gemini key: %w", err)
	}
	return &k, nil
}

func (r *GeminiKeyRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.GeminiAPIKey, error) {
	query := `SELECT ` + geminiKeyCols + ` FROM gemini_api_keys WHERE user_id = $1 ORDER BY priority ASC, created_at ASC`
	rows, err := q(ctx, r.pool).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list gemini keys: %w", err)
	}
	defer rows.Close()

	var keys []domain.GeminiAPIKey
	for rows.Next() {
		k, err := scanGeminiKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan gemini key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *GeminiKeyRepo) Update(ctx context.Context, key *domain.GeminiAPIKey) error {
	query := `UPDATE gemini_api_keys SET
		alias = COALESCE($3, alias),
		provider = COALESCE($4, provider),
		model = COALESCE($5, model),
		encrypted_key = COALESCE($6, encrypted_key),
		key_preview = COALESCE($7, key_preview),
		priority = COALESCE($8, priority),
		status = COALESCE($9, status),
		supports_text = COALESCE($10, supports_text),
		supports_audio = COALESCE($11, supports_audio),
		supports_vision = COALESCE($12, supports_vision),
		last_error = $13
		WHERE id = $1 AND user_id = $2
		RETURNING ` + geminiKeyCols
	k, err := scanGeminiKey(q(ctx, r.pool).QueryRow(ctx, query,
		key.ID, key.UserID, key.Alias, key.Provider, key.Model, key.EncryptedKey, key.KeyPreview,
		key.Priority, key.Status,
		key.SupportsText, key.SupportsAudio, key.SupportsVision,
		key.LastError,
	))
	if err != nil {
		return err
	}
	*key = k
	return nil
}

func (r *GeminiKeyRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`DELETE FROM gemini_api_keys WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *GeminiKeyRepo) GetActiveKeysByPriority(ctx context.Context, userID uuid.UUID) ([]domain.GeminiAPIKey, error) {
	query := `SELECT ` + geminiKeyCols + ` FROM gemini_api_keys WHERE user_id = $1 AND status = 'active' ORDER BY priority ASC, created_at ASC`
	rows, err := q(ctx, r.pool).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active keys: %w", err)
	}
	defer rows.Close()

	var keys []domain.GeminiAPIKey
	for rows.Next() {
		k, err := scanGeminiKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan gemini key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *GeminiKeyRepo) GetActiveKeysByModality(ctx context.Context, userID uuid.UUID, modality string) ([]domain.GeminiAPIKey, error) {
	var col string
	switch modality {
	case domain.ModalityAudio:
		col = "supports_audio"
	case domain.ModalityVision:
		col = "supports_vision"
	default:
		col = "supports_text"
	}
	query := fmt.Sprintf(`SELECT `+geminiKeyCols+` FROM gemini_api_keys WHERE user_id = $1 AND status = 'active' AND %s = true ORDER BY priority ASC, created_at ASC`, col)
	rows, err := q(ctx, r.pool).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active keys by modality: %w", err)
	}
	defer rows.Close()

	var keys []domain.GeminiAPIKey
	for rows.Next() {
		k, err := scanGeminiKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan gemini key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}
