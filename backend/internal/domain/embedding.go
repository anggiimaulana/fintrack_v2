package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TransactionEmbedding is a RAG vector row. transaction_id is NOT NULL + UNIQUE
// in the final schema, enabling ON CONFLICT (transaction_id) upserts.
type TransactionEmbedding struct {
	ID            uuid.UUID              `json:"id"`
	UserID        uuid.UUID              `json:"user_id"`
	TransactionID uuid.UUID              `json:"transaction_id"`
	Content       string                 `json:"content"`
	Embedding     []float32              `json:"-"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type SearchResult struct {
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	Similarity float64                `json:"similarity"`
}

// EmbeddingRepository stores and searches transaction embeddings. The vector is
// passed as []float32; the pgvector conversion happens in the repo layer.
type EmbeddingRepository interface {
	Upsert(ctx context.Context, userID, transactionID uuid.UUID, content string, embedding []float32, metadata map[string]interface{}) error
	Search(ctx context.Context, userID uuid.UUID, queryEmbedding []float32, topK int) ([]SearchResult, error)
	DeleteByTransactionID(ctx context.Context, transactionID uuid.UUID) error
	ListMissingTransactions(ctx context.Context, limit int) ([]Transaction, error)
}
