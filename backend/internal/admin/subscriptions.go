package admin

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fintrack/backend/pkg/response"
)

// ─── Subscriptions (P06-010) ──────────────────────────────────

type subscriptionRow struct {
	ID                string  `json:"id"`
	UserID            string  `json:"user_id"`
	UserEmail         string  `json:"user_email"`
	UserFullName      string  `json:"user_full_name"`
	PlanID            string  `json:"plan_id"`
	PlanCode          string  `json:"plan_code"`
	PlanName          string  `json:"plan_name"`
	Status            string  `json:"status"`
	PeriodStart       string  `json:"current_period_start"`
	PeriodEnd         *string `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd bool    `json:"cancel_at_period_end"`
	Gateway           string  `json:"gateway"`
	CreatedAt         string  `json:"created_at"`
}

// ListSubscriptions handles GET /admin/subscriptions.
func (h *Handler) ListSubscriptions(c *gin.Context) {
	ctx := c.Request.Context()
	page, perPage := paginationParams(c)
	offset := (page - 1) * perPage

	filterUser := c.Query("user_id")
	filterPlan := c.Query("plan_code")
	filterStatus := c.Query("status")
	search := strings.TrimSpace(c.Query("q"))

	var userArg any
	if filterUser != "" {
		if id, err := uuid.Parse(filterUser); err == nil {
			userArg = id
		}
	}

	const where = `
		WHERE ($1::uuid IS NULL OR s.user_id = $1)
		  AND ($2::text = '' OR p.code = $2)
		  AND ($3::text = '' OR s.status = $3)
		  AND ($4::text = '' OR u.email ILIKE '%'||$4||'%' OR u.full_name ILIKE '%'||$4||'%')`

	var total int64
	if err := h.svc.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscriptions s JOIN plans p ON p.id = s.plan_id
		 JOIN users u ON u.id = s.user_id`+where,
		userArg, filterPlan, filterStatus, search).Scan(&total); err != nil {
		response.InternalError(c, "Gagal memuat langganan")
		return
	}

	orderExpr := orderBy(c, map[string]string{
		"user":       "u.email",
		"plan":       "p.price_monthly_idr",
		"status":     "s.status",
		"created_at": "s.created_at",
		"ends_at":    "s.current_period_end",
	}, "s.created_at DESC")

	rows, err := h.svc.pool.Query(ctx, `
		SELECT s.id, s.user_id, u.email, u.full_name, s.plan_id, p.code, p.name,
		       s.status, s.current_period_start, s.current_period_end,
		       s.cancel_at_period_end, s.gateway, s.created_at
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		JOIN users u ON u.id = s.user_id`+where+`
		ORDER BY `+orderExpr+` LIMIT $5 OFFSET $6`,
		userArg, filterPlan, filterStatus, search, perPage, offset)
	if err != nil {
		response.InternalError(c, "Gagal memuat langganan")
		return
	}
	defer rows.Close()

	out := []subscriptionRow{}
	for rows.Next() {
		var s subscriptionRow
		var start, created time.Time
		var end *time.Time
		if err := rows.Scan(&s.ID, &s.UserID, &s.UserEmail, &s.UserFullName, &s.PlanID,
			&s.PlanCode, &s.PlanName, &s.Status, &start, &end,
			&s.CancelAtPeriodEnd, &s.Gateway, &created); err != nil {
			response.InternalError(c, "Gagal memuat langganan")
			return
		}
		s.PeriodStart = start.Format(time.RFC3339)
		if end != nil {
			t := end.Format(time.RFC3339)
			s.PeriodEnd = &t
		}
		s.CreatedAt = created.Format(time.RFC3339)
		out = append(out, s)
	}
	response.Paginated(c, "Daftar langganan berhasil dimuat", out, int(total), page, perPage)
}

type extendRequest struct {
	Days   int    `json:"days"`
	Reason string `json:"reason"`
}

// ExtendSubscription handles POST /admin/subscriptions/:id/extend.
func (h *Handler) ExtendSubscription(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req extendRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Days <= 0 {
		response.BadRequest(c, "days harus bilangan bulat positif", "validation")
		return
	}
	if req.Reason == "" {
		response.BadRequest(c, "alasan administratif wajib diisi", "validation")
		return
	}
	ctx := c.Request.Context()

	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		response.InternalError(c, "Gagal memperpanjang langganan")
		return
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	var before, after map[string]any
	err = tx.QueryRow(ctx, `
		UPDATE subscriptions
		SET current_period_end = GREATEST(COALESCE(current_period_end, now()), now())
		      + make_interval(days => $2),
		    status = CASE WHEN status IN ('expired','canceled') THEN 'active' ELSE status END,
		    cancel_at_period_end = false
		WHERE id = $1
		RETURNING user_id`, id, req.Days).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFound(c, "Langganan tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal memperpanjang langganan")
		return
	}

	before = map[string]any{"extended_days": req.Days, "reason": req.Reason}
	after = map[string]any{"status": "active"}
	if err := recordSubscriptionEvent(ctx, tx, id, userID, "extended", before, after, req.Reason, h.svc.actorIDOrNil(c)); err != nil {
		response.InternalError(c, "Gagal mencatat event langganan")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalError(c, "Gagal memperpanjang langganan")
		return
	}
	h.invalidateUser(ctx, userID)
	h.audit(c, "extend", "subscription", id.String(), nil, map[string]any{"days": req.Days, "reason": req.Reason})
	response.OK(c, "Langganan diperpanjang", nil)
}

type downgradeRequest struct {
	PlanCode string `json:"plan_code"`
	Reason   string `json:"reason"`
}

// ForceDowngrade handles POST /admin/subscriptions/:id/force-downgrade.
//
// K-09: this switches the plan (and therefore its entitlements/limits) without
// ever deleting financial rows (invoices, payments, ledger).
func (h *Handler) ForceDowngrade(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req downgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Reason == "" {
		response.BadRequest(c, "plan_code dan alasan wajib diisi", "validation")
		return
	}
	ctx := c.Request.Context()

	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		response.InternalError(c, "Gagal menurunkan paket")
		return
	}
	defer tx.Rollback(ctx)

	// Resolve target plan: explicit code, else the cheapest active plan.
	var targetID uuid.UUID
	var targetCode string
	if req.PlanCode != "" {
		err = tx.QueryRow(ctx,
			`SELECT id, code FROM plans WHERE code = $1 AND is_active`, req.PlanCode).Scan(&targetID, &targetCode)
	} else {
		err = tx.QueryRow(ctx,
			`SELECT id, code FROM plans WHERE is_active ORDER BY price_monthly_idr ASC LIMIT 1`).Scan(&targetID, &targetCode)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFound(c, "Paket tujuan tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal memuat paket tujuan")
		return
	}

	var userID uuid.UUID
	var beforePlan string
	err = tx.QueryRow(ctx, `
		SELECT s.user_id, p.code FROM subscriptions s JOIN plans p ON p.id = s.plan_id
		WHERE s.id = $1 FOR UPDATE OF s`, id).Scan(&userID, &beforePlan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFound(c, "Langganan tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal memuat langganan")
		return
	}

	if _, err := tx.Exec(ctx,
		`UPDATE subscriptions SET plan_id = $2, status = 'active', cancel_at_period_end = false WHERE id = $1`,
		id, targetID); err != nil {
		response.InternalError(c, "Gagal menurunkan paket")
		return
	}

	before := map[string]any{"plan_code": beforePlan}
	after := map[string]any{"plan_code": targetCode}
	if err := recordSubscriptionEvent(ctx, tx, id, userID, "force_downgrade", before, after, req.Reason, h.svc.actorIDOrNil(c)); err != nil {
		response.InternalError(c, "Gagal mencatat event langganan")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalError(c, "Gagal menurunkan paket")
		return
	}
	h.invalidateUser(ctx, userID)
	h.audit(c, "force_downgrade", "subscription", id.String(), before, after)
	response.OK(c, "Paket diturunkan", nil)
}

type cancelRequest struct {
	Reason    string `json:"reason"`
	Immediate bool   `json:"immediate"`
}

// CancelSubscription handles POST /admin/subscriptions/:id/cancel.
func (h *Handler) CancelSubscription(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req cancelRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Reason == "" {
		response.BadRequest(c, "alasan wajib diisi", "validation")
		return
	}
	ctx := c.Request.Context()

	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		response.InternalError(c, "Gagal membatalkan langganan")
		return
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	status := "canceled"
	immediate := req.Immediate
	// Immediate cancel ends access now; otherwise access runs to period end.
	err = tx.QueryRow(ctx, `
		UPDATE subscriptions
		SET status            = CASE WHEN $2 THEN 'canceled' ELSE status END,
		    cancel_at_period_end = NOT $2,
		    canceled_at       = now(),
		    current_period_end = CASE WHEN $2 THEN now() ELSE current_period_end END
		WHERE id = $1
		RETURNING user_id`, id, immediate).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFound(c, "Langganan tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal membatalkan langganan")
		return
	}

	after := map[string]any{"status": status, "immediate": immediate, "cancel_at_period_end": !immediate}
	if err := recordSubscriptionEvent(ctx, tx, id, userID, "canceled", nil, after, req.Reason, h.svc.actorIDOrNil(c)); err != nil {
		response.InternalError(c, "Gagal mencatat event langganan")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalError(c, "Gagal membatalkan langganan")
		return
	}
	h.invalidateUser(ctx, userID)
	h.audit(c, "cancel", "subscription", id.String(), nil, after)
	response.OK(c, "Langganan dibatalkan", nil)
}

// ─── helpers ──────────────────────────────────────────────────

func recordSubscriptionEvent(ctx contextT, tx pgx.Tx, subID, userID uuid.UUID, eventType string, before, after map[string]any, reason string, actor *uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO subscription_events
		    (subscription_id, user_id, event_type, previous_state, new_state, reason, actor_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		subID, userID, eventType, jsonOrEmpty(before), jsonOrEmpty(after), reason, actor)
	return err
}

// invalidateUser bumps the user's cache version so entitlement/limit reads
// reflect the change immediately (K-14) and drops the cached role.
func (h *Handler) invalidateUser(ctx contextT, userID uuid.UUID) {
	if h.svc.cache == nil {
		return
	}
	h.svc.cache.BumpUserVersion(ctx, userID)
	h.svc.cache.DeletePattern(ctx, roleKey(userID.String()))
}

// actorIDOrNil extracts the authenticated admin id from the request context.
func (s *Service) actorIDOrNil(c *gin.Context) *uuid.UUID {
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uuid.UUID); ok {
			return &id
		}
	}
	return nil
}

func paginationParams(c *gin.Context) (page, perPage int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage, _ = strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if perPage < 1 || perPage > 200 {
		perPage = 20
	}
	return page, perPage
}
