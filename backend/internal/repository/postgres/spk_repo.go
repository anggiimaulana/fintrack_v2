package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

type SPKRepo struct {
	pool *pgxpool.Pool
}

func NewSPKRepo(pool *pgxpool.Pool) *SPKRepo {
	return &SPKRepo{pool: pool}
}

func (r *SPKRepo) Create(ctx context.Context, result *domain.SPKResult) error {
	breakdownJSON, err := json.Marshal(result.Breakdown)
	if err != nil {
		return fmt.Errorf("failed to marshal breakdown: %w", err)
	}

	query := `
		INSERT INTO spk_results (user_id, score, status, breakdown, period_start, period_end)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, calculated_at`
	return q(ctx, r.pool).QueryRow(ctx, query,
		result.UserID, result.Score, result.Status, breakdownJSON,
		result.PeriodStart, result.PeriodEnd,
	).Scan(&result.ID, &result.CalculatedAt)
}

func (r *SPKRepo) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.SPKResult, error) {
	query := `
		SELECT id, user_id, score, status, breakdown, period_start, period_end, calculated_at
		FROM spk_results
		WHERE user_id = $1
		ORDER BY calculated_at DESC
		LIMIT 1`

	result := &domain.SPKResult{}
	var breakdownJSON []byte
	err := q(ctx, r.pool).QueryRow(ctx, query, userID).Scan(
		&result.ID, &result.UserID, &result.Score, &result.Status,
		&breakdownJSON, &result.PeriodStart, &result.PeriodEnd, &result.CalculatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest SPK: %w", err)
	}

	if len(breakdownJSON) > 0 {
		if err := json.Unmarshal(breakdownJSON, &result.Breakdown); err != nil {
			return nil, fmt.Errorf("failed to unmarshal breakdown: %w", err)
		}
	}
	return result, nil
}

func (r *SPKRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM spk_results WHERE id = $1 AND user_id = $2`
	_, err := q(ctx, r.pool).Exec(ctx, query, id, userID)
	return err
}

func (r *SPKRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]domain.SPKResult, error) {
	if limit < 1 || limit > 100 {
		limit = 10
	}

	query := `
		SELECT id, user_id, score, status, breakdown, period_start, period_end, calculated_at
		FROM spk_results
		WHERE user_id = $1
		ORDER BY calculated_at DESC
		LIMIT $2`

	rows, err := q(ctx, r.pool).Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list SPK results: %w", err)
	}
	defer rows.Close()

	var results []domain.SPKResult
	for rows.Next() {
		var result domain.SPKResult
		var breakdownJSON []byte
		if err := rows.Scan(
			&result.ID, &result.UserID, &result.Score, &result.Status,
			&breakdownJSON, &result.PeriodStart, &result.PeriodEnd, &result.CalculatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan SPK result: %w", err)
		}
		if len(breakdownJSON) > 0 {
			if err := json.Unmarshal(breakdownJSON, &result.Breakdown); err != nil {
				return nil, fmt.Errorf("failed to unmarshal breakdown: %w", err)
			}
		}
		results = append(results, result)
	}
	return results, nil
}
