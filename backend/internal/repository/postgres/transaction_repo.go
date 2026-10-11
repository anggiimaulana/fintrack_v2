package postgres

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

type TransactionRepo struct {
	pool *pgxpool.Pool
}

func NewTransactionRepo(pool *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{pool: pool}
}

func (r *TransactionRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	// Normalize NOT NULL columns on the struct so later updates stay valid.
	tx.Tags = emptyTags(tx.Tags)
	tx.Source = defaultSource(tx.Source)

	query := `
		INSERT INTO transactions (
			user_id, type, amount, income_category_id, expense_category_id, savings_category_id,
			subcategory_id, description, notes, tags, transaction_date,
			is_recurring, recurring_interval, source
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id, created_at, updated_at`

	return q(ctx, r.pool).QueryRow(
		ctx, query,
		tx.UserID, tx.Type, tx.Amount, tx.IncomeCategoryID,
		tx.ExpenseCategoryID, tx.SavingsCategoryID, tx.SubcategoryID, tx.Description,
		tx.Notes, emptyTags(tx.Tags), tx.TransactionDate,
		tx.IsRecurring, tx.RecurringInterval, defaultSource(tx.Source),
	).Scan(&tx.ID, &tx.CreatedAt, &tx.UpdatedAt)
}

// emptyTags ensures a non-nil slice so the NOT NULL `tags` column is satisfied.
func emptyTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

// defaultSource falls back to 'manual' when no source is set.
func defaultSource(s domain.TransactionSource) domain.TransactionSource {
	if s == "" {
		return domain.SourceManual
	}
	return s
}

func (r *TransactionRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Transaction, error) {
	query := `
		SELECT id, user_id, type, amount, income_category_id, expense_category_id, savings_category_id,
			subcategory_id, description, notes, tags, transaction_date,
			is_recurring, recurring_interval, source, is_deleted, deleted_at,
			created_at, updated_at
		FROM transactions
		WHERE id = $1 AND user_id = $2 AND is_deleted = false`

	tx := &domain.Transaction{}
	err := q(ctx, r.pool).QueryRow(ctx, query, id, userID).Scan(
		&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.IncomeCategoryID,
		&tx.ExpenseCategoryID, &tx.SavingsCategoryID, &tx.SubcategoryID, &tx.Description, &tx.Notes,
		&tx.Tags, &tx.TransactionDate, &tx.IsRecurring, &tx.RecurringInterval,
		&tx.Source, &tx.IsDeleted, &tx.DeletedAt, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return tx, nil
}

func (r *TransactionRepo) List(ctx context.Context, userID uuid.UUID, filter domain.TransactionFilter) ([]domain.Transaction, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
	args = append(args, userID)
	argIdx++

	conditions = append(conditions, "is_deleted = false")

	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *filter.Type)
		argIdx++
	}

	if filter.CategoryID != nil {
		conditions = append(conditions,
			fmt.Sprintf("(income_category_id = $%d OR expense_category_id = $%d OR savings_category_id = $%d)", argIdx, argIdx, argIdx))
		args = append(args, *filter.CategoryID)
		argIdx++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("transaction_date >= $%d", argIdx))
		args = append(args, *filter.DateFrom)
		argIdx++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("transaction_date <= $%d", argIdx))
		args = append(args, *filter.DateTo)
		argIdx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("description ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM transactions WHERE %s", whereClause)
	var total int
	err := q(ctx, r.pool).QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	offset := (filter.Page - 1) * filter.PerPage
	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, type, amount, income_category_id, expense_category_id, savings_category_id,
			subcategory_id, description, notes, tags, transaction_date,
			is_recurring, recurring_interval, source, is_deleted, deleted_at,
			created_at, updated_at
		FROM transactions
		WHERE %s
		ORDER BY transaction_date DESC, created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := q(ctx, r.pool).Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Type, &t.Amount, &t.IncomeCategoryID,
			&t.ExpenseCategoryID, &t.SavingsCategoryID, &t.SubcategoryID, &t.Description, &t.Notes,
			&t.Tags, &t.TransactionDate, &t.IsRecurring, &t.RecurringInterval,
			&t.Source, &t.IsDeleted, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}

	return transactions, total, nil
}

// Update applies the row. When expectedUpdatedAt is non-nil, it is used as an
// optimistic-lock guard: if no row matches, ErrConflict is returned when the
// row still exists, otherwise nil (already deleted / not owned).
func (r *TransactionRepo) Update(ctx context.Context, tx *domain.Transaction, expectedUpdatedAt *time.Time) error {
	query := `
		UPDATE transactions SET
			type = COALESCE($3, type),
			amount = COALESCE($4, amount),
			income_category_id = $5,
			expense_category_id = $6,
			savings_category_id = $7,
			subcategory_id = $8,
			description = COALESCE($9, description),
			notes = COALESCE($10, notes),
			tags = COALESCE($11, tags),
			transaction_date = COALESCE($12, transaction_date),
			is_recurring = COALESCE($13, is_recurring),
			recurring_interval = COALESCE($14, recurring_interval),
			source = COALESCE($15, source)
		WHERE id = $1 AND user_id = $2 AND is_deleted = false`
	args := []interface{}{
		tx.ID, tx.UserID, tx.Type, tx.Amount, tx.IncomeCategoryID,
		tx.ExpenseCategoryID, tx.SavingsCategoryID, tx.SubcategoryID, tx.Description,
		tx.Notes, emptyTags(tx.Tags), tx.TransactionDate, tx.IsRecurring, tx.RecurringInterval,
		defaultSource(tx.Source),
	}
	if expectedUpdatedAt != nil {
		query += ` AND updated_at = $16`
		args = append(args, *expectedUpdatedAt)
	}
	query += ` RETURNING updated_at`

	err := q(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&tx.UpdatedAt)
	if err == pgx.ErrNoRows && expectedUpdatedAt != nil {
		// Distinguish a stale write from a missing/soft-deleted row.
		exists, existsErr := r.rowExists(ctx, tx.ID, tx.UserID)
		if existsErr == nil && exists {
			return domain.ErrConflict
		}
		return nil
	}
	return err
}

// rowExists reports whether a non-deleted transaction exists for the user.
func (r *TransactionRepo) rowExists(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	var one int
	err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT 1 FROM transactions WHERE id = $1 AND user_id = $2 AND is_deleted = false`,
		id, userID).Scan(&one)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// HasTransactions reports whether the user has at least one non-deleted tx.
func (r *TransactionRepo) HasTransactions(ctx context.Context, userID uuid.UUID) (bool, error) {
	var one int
	err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT 1 FROM transactions WHERE user_id = $1 AND is_deleted = false LIMIT 1`, userID).Scan(&one)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *TransactionRepo) SoftDelete(ctx context.Context, id, userID uuid.UUID) error {
	query := `UPDATE transactions SET is_deleted = true, deleted_at = now() WHERE id = $1 AND user_id = $2`
	_, err := q(ctx, r.pool).Exec(ctx, query, id, userID)
	return err
}

func (r *TransactionRepo) GetStats(ctx context.Context, userID uuid.UUID, period domain.StatsPeriod) (*domain.TransactionStats, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expense,
			COALESCE(SUM(CASE WHEN type = 'savings' THEN amount ELSE 0 END), 0) as total_savings,
			COUNT(CASE WHEN type = 'income' THEN 1 END) as income_count,
			COUNT(CASE WHEN type = 'expense' THEN 1 END) as expense_count,
			COUNT(CASE WHEN type = 'savings' THEN 1 END) as savings_count
		FROM transactions
		WHERE user_id = $1
			AND is_deleted = false
			AND transaction_date >= $2
			AND transaction_date <= $3`

	stats := &domain.TransactionStats{}
	err := q(ctx, r.pool).QueryRow(ctx, query, userID, period.StartDate, period.EndDate).Scan(
		&stats.TotalIncome, &stats.TotalExpense, &stats.TotalSavings,
		&stats.IncomeCount, &stats.ExpenseCount, &stats.SavingsCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	stats.NetBalance = math.Round((stats.TotalIncome-stats.TotalExpense)*100) / 100
	return stats, nil
}

func (r *TransactionRepo) GetMonthlyExpense(ctx context.Context, userID uuid.UUID, year, month int) (float64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0) FROM transactions
		WHERE user_id = $1
			AND type = 'expense'
			AND is_deleted = false
			AND EXTRACT(YEAR FROM transaction_date) = $2
			AND EXTRACT(MONTH FROM transaction_date) = $3`

	var total float64
	err := q(ctx, r.pool).QueryRow(ctx, query, userID, year, month).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get monthly expense: %w", err)
	}
	return total, nil
}

func (r *TransactionRepo) GetCategoryExpensesByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.CategoryExpenseSummary, error) {
	query := `
		SELECT
			COALESCE(ec.name, 'Unknown') as category_name,
			SUM(t.amount) as total,
			COUNT(t.id) as count
		FROM transactions t
		LEFT JOIN expense_categories ec ON t.expense_category_id = ec.id
		WHERE t.user_id = $1
			AND t.type = 'expense'
			AND t.is_deleted = false
			AND t.transaction_date >= $2
			AND t.transaction_date <= $3
		GROUP BY ec.name
		ORDER BY total DESC`

	rows, err := q(ctx, r.pool).Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get category expenses: %w", err)
	}
	defer rows.Close()

	var summaries []domain.CategoryExpenseSummary
	for rows.Next() {
		var s domain.CategoryExpenseSummary
		if err := rows.Scan(&s.CategoryName, &s.Total, &s.Count); err != nil {
			return nil, fmt.Errorf("failed to scan category expense: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (r *TransactionRepo) GetCategoryIncomesByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.CategoryExpenseSummary, error) {
	query := `
		SELECT
			COALESCE(ic.name, 'Unknown') as category_name,
			SUM(t.amount) as total,
			COUNT(t.id) as count
		FROM transactions t
		LEFT JOIN income_categories ic ON t.income_category_id = ic.id
		WHERE t.user_id = $1
			AND t.type = 'income'
			AND t.is_deleted = false
			AND t.transaction_date >= $2
			AND t.transaction_date <= $3
		GROUP BY ic.name
		ORDER BY total DESC`

	rows, err := q(ctx, r.pool).Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get category incomes: %w", err)
	}
	defer rows.Close()

	var summaries []domain.CategoryExpenseSummary
	for rows.Next() {
		var s domain.CategoryExpenseSummary
		if err := rows.Scan(&s.CategoryName, &s.Total, &s.Count); err != nil {
			return nil, fmt.Errorf("failed to scan category income: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (r *TransactionRepo) GetCategorySavingsByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.CategoryExpenseSummary, error) {
	query := `
		SELECT
			COALESCE(sc.name, 'Unknown') as category_name,
			SUM(t.amount) as total,
			COUNT(t.id) as count
		FROM transactions t
		LEFT JOIN savings_categories sc ON t.savings_category_id = sc.id
		WHERE t.user_id = $1
			AND t.type = 'savings'
			AND t.is_deleted = false
			AND t.transaction_date >= $2
			AND t.transaction_date <= $3
		GROUP BY sc.name
		ORDER BY total DESC`

	rows, err := q(ctx, r.pool).Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get category savings: %w", err)
	}
	defer rows.Close()

	var summaries []domain.CategoryExpenseSummary
	for rows.Next() {
		var s domain.CategoryExpenseSummary
		if err := rows.Scan(&s.CategoryName, &s.Total, &s.Count); err != nil {
			return nil, fmt.Errorf("failed to scan category savings: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (r *TransactionRepo) GetDailyTransactionCount(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error) {
	query := `
		SELECT COUNT(DISTINCT transaction_date) FROM transactions
		WHERE user_id = $1 AND is_deleted = false
			AND transaction_date >= $2 AND transaction_date <= $3`

	var count int
	err := q(ctx, r.pool).QueryRow(ctx, query, userID, start, end).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active days: %w", err)
	}
	return count, nil
}

// CountActiveByUser returns the number of live (non-soft-deleted) transactions
// for a user. Used as the usage value for the `max_transactions` plan limit.
func (r *TransactionRepo) CountActiveByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := q(ctx, r.pool).QueryRow(ctx,
		`SELECT COUNT(*) FROM transactions WHERE user_id = $1 AND is_deleted = false`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}
	return count, nil
}

func (r *TransactionRepo) GetRecentByUser(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Transaction, error) {
	query := `
		SELECT id, user_id, type, amount, income_category_id, expense_category_id, savings_category_id,
			subcategory_id, description, notes, tags, transaction_date,
			is_recurring, recurring_interval, source, is_deleted, deleted_at,
			created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND is_deleted = false
		ORDER BY transaction_date DESC, created_at DESC
		LIMIT $2`

	rows, err := q(ctx, r.pool).Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent transactions: %w", err)
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Type, &t.Amount, &t.IncomeCategoryID,
			&t.ExpenseCategoryID, &t.SavingsCategoryID, &t.SubcategoryID, &t.Description, &t.Notes,
			&t.Tags, &t.TransactionDate, &t.IsRecurring, &t.RecurringInterval,
			&t.Source, &t.IsDeleted, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

func (r *TransactionRepo) GetCategoryTimeSeries(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.CategoryTimeSeriesRow, error) {
	query := `
		SELECT
			to_char(t.transaction_date, 'YYYY-MM') as period,
			COALESCE(ec.name, 'Unknown') as category_name,
			SUM(t.amount) as total
		FROM transactions t
		LEFT JOIN expense_categories ec ON t.expense_category_id = ec.id
		WHERE t.user_id = $1
			AND t.type = 'expense'
			AND t.is_deleted = false
			AND t.transaction_date >= $2
			AND t.transaction_date <= $3
		GROUP BY ec.name, to_char(t.transaction_date, 'YYYY-MM')
		ORDER BY ec.name, period`

	rows, err := q(ctx, r.pool).Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get category time series: %w", err)
	}
	defer rows.Close()

	var results []domain.CategoryTimeSeriesRow
	for rows.Next() {
		var row domain.CategoryTimeSeriesRow
		if err := rows.Scan(&row.Period, &row.CategoryName, &row.Total); err != nil {
			return nil, fmt.Errorf("failed to scan category time series: %w", err)
		}
		results = append(results, row)
	}
	return results, nil
}

func (r *TransactionRepo) GetTimeSeries(ctx context.Context, userID uuid.UUID, start, end time.Time, granularity string) ([]domain.TimeSeriesPoint, error) {
	var interval, datePart, periodFormat string

	switch granularity {
	case "day":
		interval = "1 day"
		datePart = "day"
		periodFormat = "to_char(d, 'YYYY-MM-DD')"
	case "week":
		interval = "1 week"
		datePart = "week"
		periodFormat = "to_char(d, 'IYYY-IW')"
	default: // month
		interval = "1 month"
		datePart = "month"
		periodFormat = "to_char(d, 'YYYY-MM')"
	}

	query := fmt.Sprintf(`
		WITH date_series AS (
			SELECT date_trunc('%s', d)::date AS bucket, %s AS period
			FROM generate_series($2::date, $3::date, '%s'::interval) d
		),
		tx_data AS (
			SELECT
				date_trunc('%s', transaction_date)::date AS bucket,
				COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
				COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense,
				COALESCE(SUM(CASE WHEN type = 'savings' THEN amount ELSE 0 END), 0) as savings
			FROM transactions
			WHERE user_id = $1
				AND is_deleted = false
				AND transaction_date >= $2
				AND transaction_date <= $3
			GROUP BY date_trunc('%s', transaction_date)::date
		)
		SELECT ds.period,
			COALESCE(td.income, 0) as income,
			COALESCE(td.expense, 0) as expense,
			COALESCE(td.savings, 0) as savings
		FROM date_series ds
		LEFT JOIN tx_data td ON ds.bucket = td.bucket
		ORDER BY ds.bucket`,
		datePart, periodFormat, interval, datePart, datePart)

	rows, err := q(ctx, r.pool).Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}
	defer rows.Close()

	var points []domain.TimeSeriesPoint
	for rows.Next() {
		var p domain.TimeSeriesPoint
		if err := rows.Scan(&p.Period, &p.Income, &p.Expense, &p.Savings); err != nil {
			return nil, fmt.Errorf("failed to scan time series: %w", err)
		}
		points = append(points, p)
	}
	return points, nil
}
