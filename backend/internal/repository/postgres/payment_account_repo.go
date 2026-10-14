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

// PaymentAccountRepo stores manual payment destinations.
type PaymentAccountRepo struct {
	pool *pgxpool.Pool
}

func NewPaymentAccountRepo(pool *pgxpool.Pool) *PaymentAccountRepo {
	return &PaymentAccountRepo{pool: pool}
}

const paymentAccountColumns = `id, kind, name, account_number, account_holder, qris_url, sort_order, is_active, created_at, updated_at`

func scanPaymentAccount(row pgx.Row) (*domain.PaymentAccount, error) {
	var a domain.PaymentAccount
	err := row.Scan(&a.ID, &a.Kind, &a.Name, &a.AccountNumber, &a.AccountHolder,
		&a.QRISURL, &a.SortOrder, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *PaymentAccountRepo) ListActive(ctx context.Context) ([]domain.PaymentAccount, error) {
	return r.list(ctx, true)
}

func (r *PaymentAccountRepo) ListAll(ctx context.Context) ([]domain.PaymentAccount, error) {
	return r.list(ctx, false)
}

func (r *PaymentAccountRepo) list(ctx context.Context, activeOnly bool) ([]domain.PaymentAccount, error) {
	query := `SELECT ` + paymentAccountColumns + ` FROM payment_accounts`
	if activeOnly {
		query += ` WHERE is_active`
	}
	query += ` ORDER BY sort_order, name`

	rows, err := q(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list payment accounts: %w", err)
	}
	defer rows.Close()

	out := []domain.PaymentAccount{}
	for rows.Next() {
		a, err := scanPaymentAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment account: %w", err)
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *PaymentAccountRepo) Create(ctx context.Context, req domain.UpsertPaymentAccountRequest) (*domain.PaymentAccount, error) {
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	query := `
		INSERT INTO payment_accounts (kind, name, account_number, account_holder, qris_url, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + paymentAccountColumns

	acc, err := scanPaymentAccount(q(ctx, r.pool).QueryRow(ctx, query,
		req.Kind, req.Name, req.AccountNumber, req.AccountHolder, req.QRISURL, req.SortOrder, active))
	if err != nil {
		return nil, fmt.Errorf("create payment account: %w", err)
	}
	return acc, nil
}

func (r *PaymentAccountRepo) Update(ctx context.Context, id uuid.UUID, req domain.UpsertPaymentAccountRequest) (*domain.PaymentAccount, error) {
	// is_active is only changed when the caller supplies it, so a partial edit
	// (e.g. fixing a typo in the holder name) never silently reactivates an
	// account the operator had turned off.
	query := `
		UPDATE payment_accounts SET
			kind = $2,
			name = $3,
			account_number = $4,
			account_holder = $5,
			qris_url = $6,
			sort_order = $7,
			is_active = COALESCE($8, is_active),
			updated_at = now()
		WHERE id = $1
		RETURNING ` + paymentAccountColumns

	acc, err := scanPaymentAccount(q(ctx, r.pool).QueryRow(ctx, query,
		id, req.Kind, req.Name, req.AccountNumber, req.AccountHolder, req.QRISURL, req.SortOrder, req.IsActive))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentAccountNotFound
		}
		return nil, fmt.Errorf("update payment account: %w", err)
	}
	return acc, nil
}

func (r *PaymentAccountRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := q(ctx, r.pool).Exec(ctx, `DELETE FROM payment_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete payment account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPaymentAccountNotFound
	}
	return nil
}
