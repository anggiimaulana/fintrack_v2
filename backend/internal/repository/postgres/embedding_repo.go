package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/fintrack/backend/internal/domain"
)

// EmbeddingRepo stores and searches transaction embeddings in pgvector.
type EmbeddingRepo struct {
	pool *pgxpool.Pool
}

func NewEmbeddingRepo(pool *pgxpool.Pool) *EmbeddingRepo {
	return &EmbeddingRepo{pool: pool}
}

// Upsert inserts or updates the embedding for a transaction.
func (r *EmbeddingRepo) Upsert(ctx context.Context, userID, transactionID uuid.UUID, content string, embedding []float32, metadata map[string]interface{}) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	_, err = q(ctx, r.pool).Exec(ctx, `
		INSERT INTO transaction_embeddings (user_id, transaction_id, content, embedding, metadata)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (transaction_id)
		DO UPDATE SET content = $3, embedding = $4, metadata = $5, user_id = $1, updated_at = now()`,
		userID, transactionID, content, pgvector.NewVector(embedding), metaJSON)
	return err
}

// Search returns the topK most similar embeddings for a user (cosine).
func (r *EmbeddingRepo) Search(ctx context.Context, userID uuid.UUID, queryEmbedding []float32, topK int) ([]domain.SearchResult, error) {
	rows, err := q(ctx, r.pool).Query(ctx, `
		SELECT content, metadata, 1 - (embedding <=> $1) AS similarity
		FROM transaction_embeddings
		WHERE user_id = $2
		ORDER BY embedding <=> $1
		LIMIT $3`,
		pgvector.NewVector(queryEmbedding), userID, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search embeddings: %w", err)
	}
	defer rows.Close()

	var results []domain.SearchResult
	for rows.Next() {
		var res domain.SearchResult
		var metaBytes []byte
		if err := rows.Scan(&res.Content, &metaBytes, &res.Similarity); err != nil {
			return nil, fmt.Errorf("failed to scan embedding: %w", err)
		}
		if len(metaBytes) > 0 {
			_ = json.Unmarshal(metaBytes, &res.Metadata)
		}
		results = append(results, res)
	}
	return results, nil
}

func (r *EmbeddingRepo) DeleteByTransactionID(ctx context.Context, transactionID uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx, `DELETE FROM transaction_embeddings WHERE transaction_id = $1`, transactionID)
	return err
}

// ListMissingTransactions returns non-deleted transactions that have no
// embedding yet (used by the backfill command).
func (r *EmbeddingRepo) ListMissingTransactions(ctx context.Context, limit int) ([]domain.Transaction, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.user_id, t.type, t.amount, t.currency_code, t.amount_base,
			t.income_category_id, t.expense_category_id,
			t.savings_category_id, t.subcategory_id, t.description, t.notes, t.tags,
			t.transaction_date, t.is_recurring, t.recurring_interval, t.source,
			t.is_deleted, t.deleted_at, t.created_at, t.updated_at
		FROM transactions t
		LEFT JOIN transaction_embeddings e ON e.transaction_id = t.id
		WHERE t.is_deleted = false AND e.id IS NULL
		ORDER BY t.transaction_date DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list missing embeddings: %w", err)
	}
	defer rows.Close()

	var txs []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Type, &t.Amount, &t.CurrencyCode, &t.AmountBase, &t.IncomeCategoryID,
			&t.ExpenseCategoryID, &t.SavingsCategoryID, &t.SubcategoryID, &t.Description, &t.Notes,
			&t.Tags, &t.TransactionDate, &t.IsRecurring, &t.RecurringInterval,
			&t.Source, &t.IsDeleted, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, nil
}
