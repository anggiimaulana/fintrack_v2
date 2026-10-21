package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

type ImportBatchRepo struct {
	pool *pgxpool.Pool
}

func NewImportBatchRepo(pool *pgxpool.Pool) *ImportBatchRepo {
	return &ImportBatchRepo{pool: pool}
}

const importBatchColumns = `id, user_id, account_id, file_name, file_size_bytes, file_hash_sha256, total_rows, imported_rows, skipped_duplicates, status, error_message, metadata, created_at`

func scanImportBatch(row pgx.Row) (*domain.ImportBatch, error) {
	var b domain.ImportBatch
	var metaBytes []byte
	err := row.Scan(
		&b.ID,
		&b.UserID,
		&b.AccountID,
		&b.FileName,
		&b.FileSizeBytes,
		&b.FileHashSHA256,
		&b.TotalRows,
		&b.ImportedRows,
		&b.SkippedDuplicates,
		&b.Status,
		&b.ErrorMessage,
		&metaBytes,
		&b.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrImportBatchNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(metaBytes) > 0 {
		_ = json.Unmarshal(metaBytes, &b.Metadata)
	}
	if b.Metadata == nil {
		b.Metadata = make(map[string]any)
	}
	return &b, nil
}

func (r *ImportBatchRepo) Create(ctx context.Context, b *domain.ImportBatch) error {
	query := `INSERT INTO import_batches (
		id, user_id, account_id, file_name, file_size_bytes, file_hash_sha256,
		total_rows, imported_rows, skipped_duplicates, status, error_message, metadata
	) VALUES (
		COALESCE(NULLIF($1, '00000000-0000-0000-0000-000000000000'::uuid), gen_random_uuid()),
		$2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
	) RETURNING ` + importBatchColumns

	metaBytes, _ := json.Marshal(b.Metadata)
	row := q(ctx, r.pool).QueryRow(ctx, query,
		b.ID, b.UserID, b.AccountID, b.FileName, b.FileSizeBytes, b.FileHashSHA256,
		b.TotalRows, b.ImportedRows, b.SkippedDuplicates, b.Status, b.ErrorMessage, metaBytes,
	)
	created, err := scanImportBatch(row)
	if err != nil {
		return fmt.Errorf("create import batch: %w", err)
	}
	*b = *created
	return nil
}

func (r *ImportBatchRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.ImportBatch, error) {
	query := `SELECT ` + importBatchColumns + ` FROM import_batches WHERE id = $1 AND user_id = $2`
	row := q(ctx, r.pool).QueryRow(ctx, query, id, userID)
	return scanImportBatch(row)
}

func (r *ImportBatchRepo) ListByUser(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID, limit, offset int) ([]domain.ImportBatch, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `SELECT ` + importBatchColumns + ` FROM import_batches WHERE user_id = $1`
	args := []any{userID}
	if accountID != nil {
		query += fmt.Sprintf(` AND account_id = $%d`, len(args)+1)
		args = append(args, *accountID)
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := q(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list import batches: %w", err)
	}
	defer rows.Close()

	var out []domain.ImportBatch
	for rows.Next() {
		b, err := scanImportBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (r *ImportBatchRepo) UpdateStatus(ctx context.Context, id, userID uuid.UUID, status domain.ImportBatchStatus, errMessage *string) error {
	query := `UPDATE import_batches SET status = $1, error_message = $2 WHERE id = $3 AND user_id = $4`
	cmd, err := q(ctx, r.pool).Exec(ctx, query, status, errMessage, id, userID)
	if err != nil {
		return fmt.Errorf("update import batch status: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrImportBatchNotFound
	}
	return nil
}

func (r *ImportBatchRepo) GetLatestByFileHash(ctx context.Context, userID, accountID uuid.UUID, fileHash string) (*domain.ImportBatch, error) {
	query := `SELECT ` + importBatchColumns + ` FROM import_batches WHERE user_id = $1 AND account_id = $2 AND file_hash_sha256 = $3 ORDER BY created_at DESC LIMIT 1`
	row := q(ctx, r.pool).QueryRow(ctx, query, userID, accountID, fileHash)
	return scanImportBatch(row)
}
