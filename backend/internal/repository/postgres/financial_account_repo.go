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

type FinancialAccountRepo struct {
	pool *pgxpool.Pool
}

func NewFinancialAccountRepo(pool *pgxpool.Pool) *FinancialAccountRepo {
	return &FinancialAccountRepo{pool: pool}
}

const financialAccountColumns = `id, user_id, name, account_type, institution_code, account_number_mask, currency_code, current_balance, initial_balance, color_hex, icon_key, is_active, created_at, updated_at`

func scanFinancialAccount(row pgx.Row) (*domain.FinancialAccount, error) {
	var a domain.FinancialAccount
	err := row.Scan(
		&a.ID,
		&a.UserID,
		&a.Name,
		&a.AccountType,
		&a.InstitutionCode,
		&a.AccountNumberMask,
		&a.CurrencyCode,
		&a.CurrentBalance,
		&a.InitialBalance,
		&a.ColorHex,
		&a.IconKey,
		&a.IsActive,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrFinancialAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *FinancialAccountRepo) Create(ctx context.Context, a *domain.FinancialAccount) error {
	query := `INSERT INTO financial_accounts (
		id, user_id, name, account_type, institution_code, account_number_mask,
		currency_code, current_balance, initial_balance, color_hex, icon_key, is_active
	) VALUES (
		COALESCE(NULLIF($1, '00000000-0000-0000-0000-000000000000'::uuid), gen_random_uuid()),
		$2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
	) RETURNING ` + financialAccountColumns

	row := q(ctx, r.pool).QueryRow(ctx, query,
		a.ID, a.UserID, a.Name, a.AccountType, a.InstitutionCode, a.AccountNumberMask,
		a.CurrencyCode, a.CurrentBalance, a.InitialBalance, a.ColorHex, a.IconKey, a.IsActive,
	)
	created, err := scanFinancialAccount(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrFinancialAccountDuplicateName
		}
		return fmt.Errorf("create financial account: %w", err)
	}
	*a = *created
	return nil
}

func (r *FinancialAccountRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.FinancialAccount, error) {
	query := `SELECT ` + financialAccountColumns + ` FROM financial_accounts WHERE id = $1 AND user_id = $2`
	row := q(ctx, r.pool).QueryRow(ctx, query, id, userID)
	return scanFinancialAccount(row)
}

func (r *FinancialAccountRepo) GetByIDForUpdate(ctx context.Context, id, userID uuid.UUID) (*domain.FinancialAccount, error) {
	query := `SELECT ` + financialAccountColumns + ` FROM financial_accounts WHERE id = $1 AND user_id = $2 FOR UPDATE`
	row := q(ctx, r.pool).QueryRow(ctx, query, id, userID)
	return scanFinancialAccount(row)
}

func (r *FinancialAccountRepo) ListByUser(ctx context.Context, userID uuid.UUID, onlyActive bool) ([]domain.FinancialAccount, error) {
	query := `SELECT ` + financialAccountColumns + ` FROM financial_accounts WHERE user_id = $1`
	if onlyActive {
		query += ` AND is_active = true`
	}
	query += ` ORDER BY name ASC`

	rows, err := q(ctx, r.pool).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list financial accounts: %w", err)
	}
	defer rows.Close()

	var out []domain.FinancialAccount
	for rows.Next() {
		a, err := scanFinancialAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *FinancialAccountRepo) Update(ctx context.Context, a *domain.FinancialAccount) error {
	query := `UPDATE financial_accounts SET
		name = $1, account_type = $2, institution_code = $3, account_number_mask = $4,
		currency_code = $5, initial_balance = $6, color_hex = $7, icon_key = $8, is_active = $9,
		updated_at = now()
	WHERE id = $10 AND user_id = $11
	RETURNING ` + financialAccountColumns

	row := q(ctx, r.pool).QueryRow(ctx, query,
		a.Name, a.AccountType, a.InstitutionCode, a.AccountNumberMask,
		a.CurrencyCode, a.InitialBalance, a.ColorHex, a.IconKey, a.IsActive,
		a.ID, a.UserID,
	)
	updated, err := scanFinancialAccount(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrFinancialAccountDuplicateName
		}
		return fmt.Errorf("update financial account: %w", err)
	}
	*a = *updated
	return nil
}

func (r *FinancialAccountRepo) UpdateBalance(ctx context.Context, id, userID uuid.UUID, newBalance float64) error {
	query := `UPDATE financial_accounts SET current_balance = $1, updated_at = now() WHERE id = $2 AND user_id = $3`
	cmd, err := q(ctx, r.pool).Exec(ctx, query, newBalance, id, userID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrFinancialAccountNotFound
	}
	return nil
}

func (r *FinancialAccountRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM financial_accounts WHERE id = $1 AND user_id = $2`
	cmd, err := q(ctx, r.pool).Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete financial account: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrFinancialAccountNotFound
	}
	return nil
}

func (r *FinancialAccountRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := q(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM financial_accounts WHERE user_id = $1 AND is_active = true`, userID).Scan(&count)
	return count, err
}
