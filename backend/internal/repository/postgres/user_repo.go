package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

// UserRepo reads/writes the unified `users` table (replaces the old
// profiles + auth.users split).
type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

const userCols = "id, email, full_name, phone, role, avatar_url, is_active, created_at, updated_at"

func scanUser(row interface {
	Scan(dest ...interface{}) error
}) (*domain.Profile, error) {
	p := &domain.Profile{}
	err := row.Scan(&p.ID, &p.Email, &p.FullName, &p.Phone, &p.Role, &p.AvatarURL, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	query := `SELECT ` + userCols + ` FROM users WHERE id = $1`
	p, err := scanUser(q(ctx, r.pool).QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return p, nil
}

// GetByEmail returns the user and their bcrypt password hash (empty if NULL).
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.Profile, string, error) {
	query := `SELECT ` + userCols + `, COALESCE(password_hash, '') FROM users WHERE email = $1`
	p := &domain.Profile{}
	var hash string
	err := q(ctx, r.pool).QueryRow(ctx, query, email).Scan(
		&p.ID, &p.Email, &p.FullName, &p.Phone, &p.Role, &p.AvatarURL, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &hash,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("failed to get user by email: %w", err)
	}
	return p, hash, nil
}

// Create inserts a new user, returning the persisted row.
func (r *UserRepo) Create(ctx context.Context, p *domain.Profile, passwordHash string) (*domain.Profile, error) {
	query := `INSERT INTO users (email, password_hash, full_name, role, is_active)
		VALUES ($1, $2, $3, $4, true)
		RETURNING ` + userCols
	created, err := scanUser(q(ctx, r.pool).QueryRow(ctx, query, p.Email, passwordHash, p.FullName, p.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return created, nil
}

func (r *UserRepo) Update(ctx context.Context, p *domain.Profile) error {
	query := `UPDATE users SET
		full_name = $2,
		phone = $3,
		avatar_url = $4
		WHERE id = $1 RETURNING updated_at`
	return q(ctx, r.pool).QueryRow(ctx, query, p.ID, p.FullName, p.Phone, p.AvatarURL).Scan(&p.UpdatedAt)
}

// SetActive toggles a user's is_active flag.
func (r *UserRepo) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := q(ctx, r.pool).Exec(ctx, `UPDATE users SET is_active = $2 WHERE id = $1`, id, active)
	return err
}

// UpdatePasswordHash replaces the user's bcrypt hash.
func (r *UserRepo) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := q(ctx, r.pool).Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, passwordHash)
	return err
}

// GetByIDWithHash returns the profile plus its bcrypt hash. Mirrors GetByEmail
// but keyed by id, so a password change does not need the email.
func (r *UserRepo) GetByIDWithHash(ctx context.Context, id uuid.UUID) (*domain.Profile, string, error) {
	query := `SELECT ` + userCols + `, COALESCE(password_hash, '') FROM users WHERE id = $1`
	var p domain.Profile
	var hash string
	err := q(ctx, r.pool).QueryRow(ctx, query, id).
		Scan(&p.ID, &p.Email, &p.FullName, &p.Phone, &p.Role, &p.AvatarURL, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user by id: %w", err)
	}
	return &p, hash, nil
}

func (r *UserRepo) List(ctx context.Context) ([]domain.Profile, error) {
	query := `SELECT ` + userCols + ` FROM users ORDER BY created_at ASC`
	rows, err := q(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var profiles []domain.Profile
	for rows.Next() {
		p, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		profiles = append(profiles, *p)
	}
	return profiles, nil
}

// ListWithStats aggregates per-user transaction totals plus global admin stats.
func (r *UserRepo) ListWithStats(ctx context.Context) ([]domain.UserStats, *domain.AdminStats, error) {
	query := `
		SELECT
			u.id, u.full_name, u.email, u.role, u.is_active, u.created_at,
			COUNT(t.id) AS tx_count,
			COALESCE(SUM(CASE WHEN t.type = 'income'  AND t.is_deleted = false THEN t.amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN t.type = 'expense' AND t.is_deleted = false THEN t.amount ELSE 0 END), 0) AS total_expense,
			MAX(t.created_at) AS last_tx
		FROM users u
		LEFT JOIN transactions t ON t.user_id = u.id AND t.is_deleted = false
		GROUP BY u.id, u.full_name, u.email, u.role, u.is_active, u.created_at
		ORDER BY u.created_at ASC`

	rows, err := q(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	defer rows.Close()

	var userStats []domain.UserStats
	var adminStats domain.AdminStats

	for rows.Next() {
		var s domain.UserStats
		if err := rows.Scan(
			&s.ID, &s.FullName, &s.Email, &s.Role, &s.IsActive, &s.CreatedAt,
			&s.TransactionCount, &s.TotalIncome, &s.TotalExpense, &s.LastTransactionAt,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan user stats: %w", err)
		}
		userStats = append(userStats, s)
		adminStats.TotalUsers++
		if s.IsActive {
			adminStats.ActiveUsers++
		}
		adminStats.TotalTransactions += s.TransactionCount
		adminStats.TotalIncome += s.TotalIncome
		adminStats.TotalExpense += s.TotalExpense
	}

	return userStats, &adminStats, nil
}
