package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/usage"
)

// UsageRepo reads a user's own consumption from the append-only token_ledger
// and the transactions table.
type UsageRepo struct {
	pool *pgxpool.Pool
}

// NewUsageRepo builds the usage repository.
func NewUsageRepo(pool *pgxpool.Pool) *UsageRepo {
	return &UsageRepo{pool: pool}
}

// truncUnit maps a granularity to a Postgres date_trunc field. Only the three
// known values are ever returned, so the fragment is safe to interpolate.
func truncUnit(g usage.Granularity) string {
	switch g {
	case usage.GranularityHour:
		return "hour"
	case usage.GranularityMonth:
		return "month"
	default:
		return "day"
	}
}

// CountInBuckets aggregates one consumption kind into time buckets. The
// granularity is an enum (never user text), so the date_trunc field is safe to
// bind as a parameter.
func (r *UsageRepo) CountInBuckets(
	ctx context.Context,
	userID uuid.UUID,
	kind string,
	g usage.Granularity,
	start, end time.Time,
) ([]usage.SeriesPoint, error) {
	unit := truncUnit(g)

	var sql string
	var args []any
	switch kind {
	case usage.KindTransaction:
		// transaction_date is a plain date; hourly buckets collapse to midnight,
		// which is correct for date-granular data.
		sql = `SELECT date_trunc($2, transaction_date::timestamptz) AS bucket, COUNT(*)::bigint
		       FROM transactions
		       WHERE user_id = $1 AND is_deleted = false
		         AND transaction_date >= $3 AND transaction_date < $4
		       GROUP BY 1 ORDER BY 1`
		args = []any{userID, unit, start, end}
	case usage.KindAIToken:
		sql = `SELECT date_trunc($2, created_at) AS bucket, COALESCE(SUM(GREATEST(raw_tokens, ABS(amount))), 0)::bigint
		       FROM token_ledger
		       WHERE user_id = $1 AND entry_type = 'commit'
		         AND created_at >= $3 AND created_at < $4
		       GROUP BY 1 ORDER BY 1`
		args = []any{userID, unit, start, end}
	case usage.KindAIRequest:
		sql = `SELECT date_trunc($2, created_at) AS bucket, COUNT(*)::bigint
		       FROM token_ledger
		       WHERE user_id = $1 AND entry_type = 'reserve'
		         AND modality NOT IN ('vision', 'voice')
		         AND created_at >= $3 AND created_at < $4
		       GROUP BY 1 ORDER BY 1`
		args = []any{userID, unit, start, end}
	case usage.KindOCR:
		sql = `SELECT date_trunc($2, created_at) AS bucket, COUNT(*)::bigint
		       FROM token_ledger
		       WHERE user_id = $1 AND entry_type = 'reserve' AND modality = 'vision'
		         AND created_at >= $3 AND created_at < $4
		       GROUP BY 1 ORDER BY 1`
		args = []any{userID, unit, start, end}
	case usage.KindVoice:
		sql = `SELECT date_trunc($2, created_at) AS bucket, COUNT(*)::bigint
		       FROM token_ledger
		       WHERE user_id = $1 AND entry_type = 'reserve' AND modality = 'voice'
		         AND created_at >= $3 AND created_at < $4
		       GROUP BY 1 ORDER BY 1`
		args = []any{userID, unit, start, end}
	default:
		return nil, nil
	}

	rows, err := q(ctx, r.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("usage timeseries (%s): %w", kind, err)
	}
	defer rows.Close()

	out := make([]usage.SeriesPoint, 0, 32)
	for rows.Next() {
		var p usage.SeriesPoint
		if err := rows.Scan(&p.Bucket, &p.Value); err != nil {
			return nil, fmt.Errorf("scan usage timeseries: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// historySortColumns whitelists the sortable columns of the history union.
var historySortColumns = map[string]string{
	"created_at": "created_at",
	"kind":       "kind",
	"amount":     "amount",
	"cost_idr":   "cost_idr",
}

// buildOrderBy turns the requested sort list into a safe ORDER BY fragment. Each
// column is validated against the whitelist; unknown columns are dropped. A
// stable `created_at DESC` tiebreaker is always appended so pagination is
// deterministic across pages.
func buildOrderBy(sorts []usage.SortField) string {
	parts := make([]string, 0, len(sorts)+1)
	for _, s := range sorts {
		col, ok := historySortColumns[strings.ToLower(s.Column)]
		if !ok {
			continue
		}
		dir := "DESC"
		if strings.EqualFold(s.Order, "asc") {
			dir = "ASC"
		}
		parts = append(parts, col+" "+dir)
	}
	parts = append(parts, "created_at DESC")
	return strings.Join(parts, ", ")
}

// HistoryEventsSQL is the union of every consumption source, normalised to one
// shape. Kept as a const so the count and page queries share it verbatim.
const historyEventsSQL = `
	SELECT t.id::text AS id,
	       'transaction' AS kind,
	       'Transaksi' AS kind_label,
	       COALESCE(NULLIF(t.description, ''), 'Transaksi ' || t.type) AS description,
	       1::bigint AS amount,
	       'transaksi' AS unit,
	       0::bigint AS cost_idr,
	       t.created_at AS created_at
	FROM transactions t
	WHERE t.user_id = $1 AND t.is_deleted = false
	UNION ALL
	SELECT l.id::text,
	       CASE WHEN l.modality = 'vision' THEN 'ocr'
	            WHEN l.modality = 'voice' THEN 'voice'
	            ELSE 'ai_request' END,
	       CASE WHEN l.modality = 'vision' THEN 'Pindai Struk'
	            WHEN l.modality = 'voice' THEN 'Input Suara'
	            ELSE 'Permintaan AI' END,
	       CASE WHEN l.modality = 'vision' THEN 'Pindai struk'
	            WHEN l.modality = 'voice' THEN 'Input suara'
	            ELSE 'Permintaan AI (' || l.modality || ')' END,
	       1::bigint,
	       CASE WHEN l.modality = 'vision' THEN 'pindai' ELSE 'permintaan' END,
	       0::bigint,
	       l.created_at
	FROM token_ledger l
	WHERE l.user_id = $1 AND l.entry_type = 'reserve'
	UNION ALL
	SELECT l.id::text,
	       'ai_token',
	       'Token AI',
	       'Token AI (' || l.modality || ')',
	       l.amount,
	       'token',
	       l.cost_idr,
	       l.created_at
	FROM token_ledger l
	WHERE l.user_id = $1 AND l.entry_type = 'commit'
`

// History returns one page of consumption events plus the total row count. The
// events come from three sources (transactions, AI reserves, AI commits) so the
// table is a genuine, paginatable ledger rather than a fixed summary.
func (r *UsageRepo) History(ctx context.Context, userID uuid.UUID, f usage.HistoryFilter) ([]usage.HistoryEvent, int, error) {
	// Build the outer WHERE from the union.
	where := []string{"1=1"}
	args := []any{userID}
	idx := 2

	if f.Kind != "" {
		where = append(where, fmt.Sprintf("kind = $%d", idx))
		args = append(args, f.Kind)
		idx++
	}
	if f.Start != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *f.Start)
		idx++
	}
	if f.End != nil {
		where = append(where, fmt.Sprintf("created_at < $%d", idx))
		args = append(args, *f.End)
		idx++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("description ILIKE $%d", idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	whereClause := strings.Join(where, " AND ")

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s) e WHERE %s", historyEventsSQL, whereClause)
	var total int
	if err := q(ctx, r.pool).QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count usage history: %w", err)
	}

	orderClause := buildOrderBy(f.Sorts)

	offset := (f.Page - 1) * f.PerPage
	pageSQL := fmt.Sprintf(
		"SELECT id, kind, kind_label, description, amount, unit, cost_idr, created_at FROM (%s) e WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d",
		historyEventsSQL, whereClause, orderClause, idx, idx+1,
	)
	args = append(args, f.PerPage, offset)

	rows, err := q(ctx, r.pool).Query(ctx, pageSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list usage history: %w", err)
	}
	defer rows.Close()

	out := make([]usage.HistoryEvent, 0, f.PerPage)
	for rows.Next() {
		var e usage.HistoryEvent
		if err := rows.Scan(&e.ID, &e.Kind, &e.KindLabel, &e.Description, &e.Amount, &e.Unit, &e.CostIDR, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan usage history: %w", err)
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}
