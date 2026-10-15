package admin

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fintrack/backend/pkg/response"
)

// ─── Token ledger administration (P06-014) ────────────────────

type tokenBalanceRow struct {
	UserID      string  `json:"user_id"`
	Email       string  `json:"user_email"`
	FullName    string  `json:"user_full_name"`
	PeriodID    *string `json:"period_id,omitempty"`
	PeriodStart *string `json:"period_start,omitempty"`
	PeriodEnd   *string `json:"period_end,omitempty"`
	Quota       int64   `json:"quota_tokens"`
	Used        int64   `json:"used_tokens"`
	Reserved    int64   `json:"reserved_tokens"`
	Available   int64   `json:"available_tokens"`
}

// ListTokenBalances handles GET /admin/tokens/users.
func (h *Handler) ListTokenBalances(c *gin.Context) {
	ctx := c.Request.Context()
	page, perPage := paginationParams(c)
	search := c.Query("q")

	rows, err := h.svc.pool.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (user_id) *
			FROM token_periods
			ORDER BY user_id, period_start DESC
		)
		SELECT u.id, u.email, u.full_name,
		       t.id, t.period_start, t.period_end,
		       COALESCE(t.quota_tokens,0), COALESCE(t.used_tokens,0), COALESCE(t.reserved_tokens,0)
		FROM users u
		LEFT JOIN latest t ON t.user_id = u.id
		WHERE ($1::text = '' OR u.email ILIKE '%'||$1||'%' OR u.full_name ILIKE '%'||$1||'%')
		ORDER BY `+orderBy(c, map[string]string{
		"user":  "u.email",
		"quota": "COALESCE(t.quota_tokens,0)",
		"used":  "COALESCE(t.used_tokens,0)",
	}, "COALESCE(t.used_tokens,0) DESC, u.created_at DESC")+`
		LIMIT $2 OFFSET $3`, search, perPage, (page-1)*perPage)
	if err != nil {
		response.InternalError(c, "Gagal memuat saldo token")
		return
	}
	defer rows.Close()

	out := []tokenBalanceRow{}
	for rows.Next() {
		var b tokenBalanceRow
		var start, end *time.Time
		if err := rows.Scan(&b.UserID, &b.Email, &b.FullName, &b.PeriodID, &start, &end,
			&b.Quota, &b.Used, &b.Reserved); err != nil {
			response.InternalError(c, "Gagal memuat saldo token")
			return
		}
		if start != nil {
			s := start.Format(time.RFC3339)
			b.PeriodStart = &s
		}
		if end != nil {
			s := end.Format(time.RFC3339)
			b.PeriodEnd = &s
		}
		b.Available = b.Quota - b.Used - b.Reserved
		out = append(out, b)
	}

	var total int64
	_ = h.svc.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE ($1::text = '' OR email ILIKE '%'||$1||'%' OR full_name ILIKE '%'||$1||'%')`,
		search).Scan(&total)
	response.Paginated(c, "Saldo token berhasil dimuat", out, int(total), page, perPage)
}

type ledgerRow struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	EntryType string `json:"entry_type"`
	// Amount is the plan-quota UNIT delta (raw / token_quota_ratio).
	Amount int64 `json:"amount"`
	// RawTokens is the provider token count (the cost basis). It is 0 for
	// reserve/release; rows written before the ratio migration also carry 0 and
	// should be read as |Amount| (see billing.TokenLedgerEntry.RawTokensOrAmount).
	RawTokens      int64          `json:"raw_tokens"`
	Modality       string         `json:"modality"`
	CostIDR        int64          `json:"cost_idr"`
	IdempotencyKey string         `json:"idempotency_key"`
	ReferenceID    *string        `json:"reference_id,omitempty"`
	Notes          *string        `json:"notes,omitempty"`
	CreatedAt      string         `json:"created_at"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// ListTokenLedger handles GET /admin/tokens/ledger.
func (h *Handler) ListTokenLedger(c *gin.Context) {
	ctx := c.Request.Context()
	page, perPage := paginationParams(c)
	entryType := c.Query("entry_type")
	userID := c.Query("user_id")
	modality := c.Query("modality")
	from, to := dateRange(c)

	var userArg any
	if id, err := uuid.Parse(userID); err == nil {
		userArg = id
	}

	const where = `
		WHERE ($1::text = '' OR l.entry_type = $1)
		  AND ($2::uuid IS NULL OR l.user_id = $2)
		  AND ($3::text = '' OR l.modality = $3)
		  AND ($4::timestamptz IS NULL OR l.created_at >= $4)
		  AND ($5::timestamptz IS NULL OR l.created_at < $5)`

	var total int64
	if err := h.svc.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM token_ledger l`+where, entryType, userArg, modality, from, to).Scan(&total); err != nil {
		response.InternalError(c, "Gagal memuat ledger token")
		return
	}

	rows, err := h.svc.pool.Query(ctx, `
		SELECT l.id, l.user_id, u.email, l.entry_type, l.amount, l.raw_tokens, l.modality, l.cost_idr,
		       l.idempotency_key, l.reference_id, l.notes, l.created_at
		FROM token_ledger l
		JOIN users u ON u.id = l.user_id`+where+`
		ORDER BY `+orderBy(c, map[string]string{
		"created_at": "l.created_at",
		"amount":     "l.amount",
		"raw_tokens": "l.raw_tokens",
		"cost":       "l.cost_idr",
		"user":       "u.email",
		"type":       "l.entry_type",
	}, "l.created_at DESC")+` LIMIT $6 OFFSET $7`,
		entryType, userArg, modality, from, to, perPage, (page-1)*perPage)
	if err != nil {
		response.InternalError(c, "Gagal memuat ledger token")
		return
	}
	defer rows.Close()

	out := []ledgerRow{}
	for rows.Next() {
		var l ledgerRow
		var created time.Time
		if err := rows.Scan(&l.ID, &l.UserID, &l.UserEmail, &l.EntryType, &l.Amount, &l.RawTokens, &l.Modality,
			&l.CostIDR, &l.IdempotencyKey, &l.ReferenceID, &l.Notes, &created); err != nil {
			response.InternalError(c, "Gagal memuat ledger token")
			return
		}
		l.CreatedAt = created.Format(time.RFC3339)
		out = append(out, l)
	}
	response.Paginated(c, "Ledger token berhasil dimuat", out, int(total), page, perPage)
}

type adjustRequest struct {
	UserID         string `json:"user_id"`
	Amount         int64  `json:"amount"` // signed: +credit / -debit
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

// adjustResponse is the stored/served result of an adjustment.
type adjustResponse struct {
	LedgerID string `json:"ledger_id"`
	UserID   string `json:"user_id"`
	Amount   int64  `json:"amount"`
	Quota    int64  `json:"quota_tokens"`
	Replayed bool   `json:"replayed"`
}

// AdjustTokens handles POST /admin/tokens/adjust.
//
// Idempotent on `idempotency_key` (K-28): the ledger row is inserted with
// ON CONFLICT DO NOTHING, and a replay returns the original result without
// touching the period again.
func (h *Handler) AdjustTokens(c *gin.Context) {
	var req adjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.BadRequest(c, "user_id tidak valid", "validation")
		return
	}
	if req.Amount == 0 {
		response.BadRequest(c, "amount tidak boleh 0", "validation")
		return
	}
	if req.Reason == "" {
		response.BadRequest(c, "alasan administratif wajib diisi", "validation")
		return
	}
	if _, err := uuid.Parse(req.IdempotencyKey); err != nil {
		response.BadRequest(c, "idempotency_key harus UUID valid", "validation")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		response.InternalError(c, "Gagal menyesuaikan token")
		return
	}
	defer tx.Rollback(ctx)

	// Lock the user's current period.
	var periodID uuid.UUID
	var quota, used, reserved int64
	err = tx.QueryRow(ctx, `
		SELECT id, quota_tokens, used_tokens, reserved_tokens
		FROM token_periods
		WHERE user_id = $1 AND period_start <= now() AND period_end >= now()
		ORDER BY period_start DESC LIMIT 1
		FOR UPDATE`, userID).Scan(&periodID, &quota, &used, &reserved)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, 409, "Pengguna tidak memiliki periode token aktif", "NO_ACTIVE_PERIOD")
			return
		}
		response.InternalError(c, "Gagal menyesuaikan token")
		return
	}

	available := quota - used - reserved
	if req.Amount < 0 && -req.Amount > available {
		response.BadRequest(c, "debit melebihi saldo tersedia", "validation")
		return
	}

	// Atomic dedupe: only the first insert for a key proceeds.
	var ledgerID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO token_ledger
		    (period_id, user_id, entry_type, amount, modality, cost_idr, idempotency_key, notes)
		VALUES ($1,$2,'adjust',$3,'admin',0,$4,$5)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id`,
		periodID, userID, req.Amount, req.IdempotencyKey, req.Reason).Scan(&ledgerID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Replay: return the stored result, no second mutation.
		var storedAmount, storedQuota int64
		if err := tx.QueryRow(ctx, `
			SELECT l.id, l.amount, p.quota_tokens
			FROM token_ledger l JOIN token_periods p ON p.id = l.period_id
			WHERE l.idempotency_key = $1`, req.IdempotencyKey).Scan(&ledgerID, &storedAmount, &storedQuota); err != nil {
			response.InternalError(c, "Gagal membaca hasil tersimpan")
			return
		}
		response.OK(c, "Penyesuaian sudah pernah diproses", adjustResponse{
			LedgerID: ledgerID.String(), UserID: userID.String(),
			Amount: storedAmount, Quota: storedQuota, Replayed: true,
		})
		return
	}
	if err != nil {
		response.InternalError(c, "Gagal menyesuaikan token")
		return
	}

	newQuota := quota + req.Amount
	if _, err := tx.Exec(ctx, `UPDATE token_periods SET quota_tokens = $2 WHERE id = $1`, periodID, newQuota); err != nil {
		response.InternalError(c, "Gagal menyesuaikan token")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalError(c, "Gagal menyesuaikan token")
		return
	}

	h.invalidateUser(ctx, userID)
	h.audit(c, "adjust", "token", ledgerID.String(), nil, map[string]any{
		"user_id": userID.String(), "amount": req.Amount, "quota_tokens": newQuota, "reason": req.Reason,
	})
	response.OK(c, "Penyesuaian token tersimpan", adjustResponse{
		LedgerID: ledgerID.String(), UserID: userID.String(),
		Amount: req.Amount, Quota: newQuota,
	})
}
