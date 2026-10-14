package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/infrastructure/auth"
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
	UserID               uuid.UUID `json:"user_id"`
	AnonymizedEmail      string    `json:"anonymized_email"`
	ChatMessagesDeleted  int64     `json:"chat_messages_deleted"`
	ChatSessionsDeleted  int64     `json:"chat_sessions_deleted"`
	EmbeddingsDeleted    int64     `json:"embeddings_deleted"`
	RefreshTokensRevoked int64     `json:"refresh_tokens_revoked"`
	InvoicesRetained     int64     `json:"invoices_retained"`
	PaymentsRetained     int64     `json:"payments_retained"`
	AnonymizedAt         time.Time `json:"anonymized_at"`
}

// AnonymizeUserData executes atomic data erasure and pseudonimization:
// 1. Hard deletes AI chat messages, chat sessions, and vector embeddings (768d).
// 2. Revokes refresh tokens.
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

	// 4b. Delete notification inbox and preferences (personal data: titles and
	// bodies may contain amounts/invoice numbers).
	_, err = tx.Exec(ctx, `DELETE FROM notifications WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("delete notifications: %w", err)
	}
	_, err = tx.Exec(ctx, `DELETE FROM notification_preferences WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("delete notification_preferences: %w", err)
	}

	// 5. Cancel any active subscription so the anonymized user is no longer
	// counted in MRR or granted paid entitlements. The row is retained (not
	// deleted) so billing history and audit trails stay intact.
	_, err = tx.Exec(ctx, `
		UPDATE subscriptions
		SET status = 'canceled', canceled_at = NOW(), cancel_at_period_end = false, updated_at = NOW()
		WHERE user_id = $1 AND status NOT IN ('canceled', 'expired')`, userID)
	if err != nil {
		return nil, fmt.Errorf("cancel subscription: %w", err)
	}

	// 6. Pseudonymize users table
	anonymizedEmail := fmt.Sprintf("anonymized-%s@deleted.fintrack.internal", userID.String())
	res.AnonymizedEmail = anonymizedEmail

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET email = $2,
		    full_name = 'Eks Pengguna (Teranonimkan)',
		    phone = NULL,
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

// VerifyPassword checks if the provided password matches the user's password_hash.
func (s *AnonymizeService) VerifyPassword(ctx context.Context, userID uuid.UUID, password string) (bool, error) {
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if err != nil {
		return false, err
	}
	return auth.CheckPassword(hash, password), nil
}

type UserDataExport struct {
	ExportedAt   time.Time        `json:"exported_at"`
	Profile      map[string]any   `json:"profile"`
	Transactions []map[string]any `json:"transactions"`
	ChatSessions []map[string]any `json:"chat_sessions"`
	Invoices     []map[string]any `json:"invoices"`
	Payments     []map[string]any `json:"payments"`
}

// ExportUserData compiles all data associated with the user for Data Portability (UU PDP Pasal 21).
func (s *AnonymizeService) ExportUserData(ctx context.Context, userID uuid.UUID) (*UserDataExport, error) {
	export := &UserDataExport{
		ExportedAt:   time.Now().UTC(),
		Transactions: []map[string]any{},
		ChatSessions: []map[string]any{},
		Invoices:     []map[string]any{},
		Payments:     []map[string]any{},
	}

	// 1. Profile
	var email, fullName, role string
	var phone *string
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `SELECT email, full_name, phone, role, created_at FROM users WHERE id = $1`, userID).
		Scan(&email, &fullName, &phone, &role, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}
	export.Profile = map[string]any{
		"id":         userID.String(),
		"email":      email,
		"full_name":  fullName,
		"phone":      phone,
		"role":       role,
		"created_at": createdAt,
	}

	// 2. Transactions
	tRows, err := s.pool.Query(ctx, `
		SELECT id, type, amount, transaction_date, description, source, created_at
		FROM transactions WHERE user_id = $1 AND is_deleted = false
		ORDER BY transaction_date DESC`, userID)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var id uuid.UUID
			var tType, source string
			var amount float64
			var date time.Time
			var desc *string
			var cAt time.Time
			if err := tRows.Scan(&id, &tType, &amount, &date, &desc, &source, &cAt); err == nil {
				item := map[string]any{
					"id":               id.String(),
					"type":             tType,
					"amount":           amount,
					"transaction_date": date.Format("2006-01-02"),
					"source":           source,
					"created_at":       cAt,
				}
				if desc != nil {
					item["description"] = *desc
				}
				export.Transactions = append(export.Transactions, item)
			}
		}
	}

	// 3. Chat Sessions & Messages
	sRows, err := s.pool.Query(ctx, `
		SELECT id, title, created_at FROM chat_sessions WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err == nil {
		defer sRows.Close()
		for sRows.Next() {
			var sid uuid.UUID
			var title string
			var sCreatedAt time.Time
			if err := sRows.Scan(&sid, &title, &sCreatedAt); err == nil {
				sessionItem := map[string]any{
					"id":         sid.String(),
					"title":      title,
					"created_at": sCreatedAt,
					"messages":   []map[string]any{},
				}
				mRows, merr := s.pool.Query(ctx, `
					SELECT id, role, content, created_at FROM chat_messages WHERE session_id = $1 ORDER BY created_at ASC`, sid)
				if merr == nil {
					var msgs []map[string]any
					for mRows.Next() {
						var mid uuid.UUID
						var role, content string
						var mCreatedAt time.Time
						if err := mRows.Scan(&mid, &role, &content, &mCreatedAt); err == nil {
							msgs = append(msgs, map[string]any{
								"id":         mid.String(),
								"role":       role,
								"content":    content,
								"created_at": mCreatedAt,
							})
						}
					}
					mRows.Close()
					sessionItem["messages"] = msgs
				}
				export.ChatSessions = append(export.ChatSessions, sessionItem)
			}
		}
	}

	// 4. Invoices & Payments
	iRows, err := s.pool.Query(ctx, `
		SELECT id, invoice_number, subtotal_idr, tax_idr, total_idr, status, issued_at
		FROM invoices WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err == nil {
		defer iRows.Close()
		for iRows.Next() {
			var iid uuid.UUID
			var invNum, status string
			var subtotal, tax, total int64
			var issuedAt time.Time
			if err := iRows.Scan(&iid, &invNum, &subtotal, &tax, &total, &status, &issuedAt); err == nil {
				export.Invoices = append(export.Invoices, map[string]any{
					"id":             iid.String(),
					"invoice_number": invNum,
					"subtotal_idr":   subtotal,
					"tax_idr":        tax,
					"total_idr":      total,
					"status":         status,
					"issued_at":      issuedAt,
				})
			}
		}
	}

	pRows, err := s.pool.Query(ctx, `
		SELECT id, gateway, amount_idr, status, created_at
		FROM payments WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var pid uuid.UUID
			var gw, status string
			var amount int64
			var pCreatedAt time.Time
			if err := pRows.Scan(&pid, &gw, &amount, &status, &pCreatedAt); err == nil {
				export.Payments = append(export.Payments, map[string]any{
					"id":         pid.String(),
					"gateway":    gw,
					"amount_idr": amount,
					"status":     status,
					"created_at": pCreatedAt,
				})
			}
		}
	}

	return export, nil
}
