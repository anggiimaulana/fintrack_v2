package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

const chatSessionCols = "id, user_id, title, source, created_at, updated_at"

type ChatSessionRepo struct {
	pool *pgxpool.Pool
}

func NewChatSessionRepo(pool *pgxpool.Pool) *ChatSessionRepo {
	return &ChatSessionRepo{pool: pool}
}

type sessionScanner interface {
	Scan(dest ...interface{}) error
}

func scanChatSession(s sessionScanner) (domain.ChatSession, error) {
	var sess domain.ChatSession
	err := s.Scan(&sess.ID, &sess.UserID, &sess.Title, &sess.Source, &sess.CreatedAt, &sess.UpdatedAt)
	return sess, err
}

func (r *ChatSessionRepo) Create(ctx context.Context, s *domain.ChatSession) error {
	query := `INSERT INTO chat_sessions (user_id, title, source) VALUES ($1, $2, $3) RETURNING ` + chatSessionCols
	return q(ctx, r.pool).QueryRow(ctx, query, s.UserID, s.Title, s.Source).Scan(&s.ID, &s.UserID, &s.Title, &s.Source, &s.CreatedAt, &s.UpdatedAt)
}

func (r *ChatSessionRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.ChatSession, error) {
	query := `SELECT ` + chatSessionCols + ` FROM chat_sessions WHERE id = $1 AND user_id = $2`
	s, err := scanChatSession(q(ctx, r.pool).QueryRow(ctx, query, id, userID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get chat session: %w", err)
	}
	return &s, nil
}

func (r *ChatSessionRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.ChatSession, error) {
	query := `SELECT ` + chatSessionCols + ` FROM chat_sessions WHERE user_id = $1 ORDER BY updated_at DESC`
	rows, err := q(ctx, r.pool).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list chat sessions: %w", err)
	}
	defer rows.Close()

	var sessions []domain.ChatSession
	for rows.Next() {
		s, err := scanChatSession(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *ChatSessionRepo) ListByUserAndSource(ctx context.Context, userID uuid.UUID, source string) ([]domain.ChatSession, error) {
	query := `SELECT ` + chatSessionCols + ` FROM chat_sessions WHERE user_id = $1 AND source = $2 ORDER BY updated_at DESC`
	rows, err := q(ctx, r.pool).Query(ctx, query, userID, source)
	if err != nil {
		return nil, fmt.Errorf("failed to list chat sessions by source: %w", err)
	}
	defer rows.Close()

	var sessions []domain.ChatSession
	for rows.Next() {
		s, err := scanChatSession(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *ChatSessionRepo) Update(ctx context.Context, s *domain.ChatSession) error {
	query := `UPDATE chat_sessions SET title = $1 WHERE id = $2 AND user_id = $3 RETURNING updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query, s.Title, s.ID, s.UserID).Scan(&s.UpdatedAt)
}

func (r *ChatSessionRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := q(ctx, r.pool).Exec(ctx, `DELETE FROM chat_sessions WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}
