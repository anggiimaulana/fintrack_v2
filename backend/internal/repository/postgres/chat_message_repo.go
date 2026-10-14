package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

type ChatMessageRepo struct {
	pool *pgxpool.Pool
}

func NewChatMessageRepo(pool *pgxpool.Pool) *ChatMessageRepo {
	return &ChatMessageRepo{pool: pool}
}

func (r *ChatMessageRepo) Create(ctx context.Context, msg *domain.ChatMessage) error {
	query := `INSERT INTO chat_messages (session_id, user_id, role, content, model_used, mode, tokens_used) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at`
	return q(ctx, r.pool).QueryRow(ctx, query, msg.SessionID, msg.UserID, msg.Role, msg.Content, msg.ModelUsed, msg.Mode, msg.TokensUsed).Scan(&msg.ID, &msg.CreatedAt)
}

func (r *ChatMessageRepo) ListBySession(ctx context.Context, sessionID, userID uuid.UUID) ([]domain.ChatMessage, error) {
	query := `SELECT id, session_id, user_id, role, content, model_used, mode, tokens_used, created_at FROM chat_messages WHERE session_id = $1 AND user_id = $2 ORDER BY created_at ASC`
	rows, err := q(ctx, r.pool).Query(ctx, query, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var msgs []domain.ChatMessage
	for rows.Next() {
		var m domain.ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.UserID, &m.Role, &m.Content, &m.ModelUsed, &m.Mode, &m.TokensUsed, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (r *ChatMessageRepo) DeleteBySession(ctx context.Context, sessionID, userID uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx, `DELETE FROM chat_messages WHERE session_id = $1 AND user_id = $2`, sessionID, userID)
	return err
}
