package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnonymizeService implements data subject rights under UU PDP No. 27/2022
// (Right to Erasure / Pemusnahan Data Pribadi) while preserving financial ledger
// records for 10-year statutory tax retention under UU KUP No. 28/2007 Pasal 28.
type AnonymizeService struct {
	pool *pgxpool.Pool
}

func NewAnonymizeService(pool *pgxpool.Pool) *AnonymizeService {
	return &AnonymizeService{pool: pool}
}

type AnonymizeResult struct {
	UserID              uuid.UUID `json:"user_id"`
	AnonymizedEmail     string    `json:"anonymized_email"`
	ChatMessagesDeleted int64     `json:"chat_messages_deleted"`
	ChatSessionsDeleted int64     `json:"chat_sessions_deleted"`
	EmbeddingsDeleted   int64     `json:"embeddings_deleted"`
	RefreshTokensRevoked int64    `json:"refresh_tokens_revoked"`
	InvoicesRetained    int64     `json:"invoices_retained"`
	PaymentsRetained    int64     `json:"payments_retained"`
	AnonymizedAt        time.Time `json:"anonymized_at"`
}

// AnonymizeUserData executes atomic data erasure and pseudonimization:
// 1. Hard deletes AI chat messages, chat sessions, and vector embeddings (768d).
// 2. Revokes refresh tokens and deletes BYOK keys.
// 3. Pseudonimizes users table: PII overwritten with anonymous identifier, password shredded.
// 4. Invoices, payments, and token_ledger records remain intact with original sums and dates.
// 5. Writes compliance entry to audit_logs.
func (s *AnonymizeService) AnonymizeUserData(ctx context.Context, userID uuid.UUID, actorID *uuid.UUID, reason string) (*AnonymizeResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	res := &AnonymizeResult{
		UserID:       userID,
		AnonymizedAt: time.Now().UTC(),
	}

	// 1. Delete chat messages
	cmd, err := tx.Exec(ctx, `DELETE FROM chat_messages WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("delete chat_messages: %w", err)
	}
	res.ChatMessagesDeleted = cmd.RowsAffected()

	// 2. Delete chat sessions
	cmd, err = tx.Exec(ctx, `DELETE FROM chat_sessions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("delete chat_sessions: %w", err)
	}
	res.ChatSessionsDeleted = cmd.RowsAffected()

	// 3. Delete vector embeddings
	cmd, err = tx.Exec(ctx, `DELETE FROM transaction_embeddings WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("delete transaction_embeddings: %w", err)
	}
	res.EmbeddingsDeleted = cmd.RowsAffected()

	// 4. Delete refresh tokens
	cmd, err = tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("delete refresh_tokens: %w", err)
	}
	res.RefreshTokensRevoked = cmd.RowsAffected()

	// 5. Delete BYOK keys
	_, err = tx.Exec(ctx, `DELETE FROM gemini_api_keys WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("delete gemini_api_keys: %w", err)
	}

	// 6. Pseudonymize users table
	anonymizedEmail := fmt.Sprintf("anonymized-%s@deleted.fintrack.internal", userID.String())
	res.AnonymizedEmail = anonymizedEmail

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET email = $2,
		    full_name = 'Eks Pengguna (Teranonimkan)',
		    password_hash = 'DELETED_CRYPTOGRAPHIC_SHREDDING',
		    avatar_url = NULL,
		    is_active = false,
		    updated_at = NOW()
		WHERE id = $1`, userID, anonymizedEmail)
	if err != nil {
		return nil, fmt.Errorf("pseudonymize users: %w", err)
	}

	// 7. Count retained tax and financial ledger documents
	err = tx.QueryRow(ctx, `SELECT count(*) FROM invoices WHERE user_id = $1`, userID).Scan(&res.InvoicesRetained)
	if err != nil {
		return nil, fmt.Errorf("count invoices: %w", err)
	}

	err = tx.QueryRow(ctx, `SELECT count(*) FROM payments WHERE user_id = $1`, userID).Scan(&res.PaymentsRetained)
	if err != nil {
		return nil, fmt.Errorf("count payments: %w", err)
	}

	// 8. Log compliance event to audit_logs
	if reason == "" {
		reason = "Permohonan Penghapusan Data Pribadi (UU PDP No. 27/2022 Pasal 43)"
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (id, actor_id, action, entity, entity_id, after, created_at)
		VALUES (gen_random_uuid(), $1, 'user.pdp_anonymize', 'users', $2, jsonb_build_object('reason', $3::text, 'anonymized_email', $4::text), NOW())`,
		actorID, userID.String(), reason, anonymizedEmail)
	if err != nil {
		return nil, fmt.Errorf("insert audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return res, nil
}
