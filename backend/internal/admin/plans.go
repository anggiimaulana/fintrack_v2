package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fintrack/backend/pkg/response"
)

// ─── Plans (P06-008) ──────────────────────────────────────────

type planDTO struct {
	ID                string           `json:"id"`
	Code              string           `json:"code"`
	Name              string           `json:"name"`
	Description       *string          `json:"description,omitempty"`
	PriceMonthlyIDR   int64            `json:"price_monthly_idr"`
	PriceYearlyIDR    int64            `json:"price_yearly_idr"`
	BillingPeriodDays int              `json:"billing_period_days"`
	IsActive          bool             `json:"is_active"`
	IsPublic          bool             `json:"is_public"`
	FallbackGateway   string           `json:"fallback_gateway"`
	SortOrder         int              `json:"sort_order"`
	TokenQuotaRatio   int64            `json:"token_quota_ratio"`
	Entitlements      map[string]bool  `json:"entitlements"`
	Limits            map[string]int64 `json:"limits"`
}

type planRequest struct {
	Code              string            `json:"code"`
	Name              string            `json:"name"`
	Description       *string           `json:"description"`
	PriceMonthlyIDR   *int64            `json:"price_monthly_idr"`
	PriceYearlyIDR    *int64            `json:"price_yearly_idr"`
	BillingPeriodDays *int              `json:"billing_period_days"`
	IsActive          *bool             `json:"is_active"`
	IsPublic          *bool             `json:"is_public"`
	FallbackGateway   *string           `json:"fallback_gateway"`
	SortOrder         *int              `json:"sort_order"`
	TokenQuotaRatio   *int64            `json:"token_quota_ratio"`
	Entitlements      *map[string]bool  `json:"entitlements"`
	Limits            *map[string]int64 `json:"limits"`
}

// ListPlans handles GET /admin/plans (all plans, including inactive).
func (h *Handler) ListPlans(c *gin.Context) {
	ctx := c.Request.Context()
	plans, err := h.loadPlans(ctx)
	if err != nil {
		response.InternalError(c, "Gagal memuat paket")
		return
	}
	response.OK(c, "Daftar paket berhasil dimuat", plans)
}

func (h *Handler) loadPlans(ctx contextT) ([]planDTO, error) {
	rows, err := h.svc.pool.Query(ctx, `
		SELECT id, code, name, description, price_monthly_idr, price_yearly_idr,
		       billing_period_days, is_active, is_public, fallback_gateway, sort_order, token_quota_ratio
		FROM plans ORDER BY sort_order ASC, price_monthly_idr ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []planDTO{}
	for rows.Next() {
		var p planDTO
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.PriceMonthlyIDR,
			&p.PriceYearlyIDR, &p.BillingPeriodDays, &p.IsActive, &p.IsPublic,
			&p.FallbackGateway, &p.SortOrder, &p.TokenQuotaRatio); err != nil {
			return nil, err
		}
		p.Entitlements = map[string]bool{}
		p.Limits = map[string]int64{}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// hydrate entitlements + limits in two bulk queries
	erows, err := h.svc.pool.Query(ctx, `SELECT plan_id, feature_key, is_enabled FROM plan_entitlements`)
	if err == nil {
		defer erows.Close()
		idx := indexByID(out)
		for erows.Next() {
			var pid uuid.UUID
			var key string
			var enabled bool
			if erows.Scan(&pid, &key, &enabled) == nil {
				if p, ok := idx[pid.String()]; ok {
					out[p].Entitlements[key] = enabled
				}
			}
		}
	}
	lrows, err := h.svc.pool.Query(ctx, `SELECT plan_id, limit_key, limit_value FROM plan_limits`)
	if err == nil {
		defer lrows.Close()
		idx := indexByID(out)
		for lrows.Next() {
			var pid uuid.UUID
			var key string
			var val int64
			if lrows.Scan(&pid, &key, &val) == nil {
				if p, ok := idx[pid.String()]; ok {
					out[p].Limits[key] = val
				}
			}
		}
	}
	return out, nil
}

func indexByID(plans []planDTO) map[string]int {
	m := make(map[string]int, len(plans))
	for i, p := range plans {
		m[p.ID] = i
	}
	return m
}

// CreatePlan handles POST /admin/plans.
func (h *Handler) CreatePlan(c *gin.Context) {
	var req planRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	req.Code = strings.TrimSpace(strings.ToLower(req.Code))
	if req.Code == "" || strings.TrimSpace(req.Name) == "" {
		response.BadRequest(c, "kode dan nama paket wajib diisi", "validation")
		return
	}
	if (req.PriceMonthlyIDR != nil && *req.PriceMonthlyIDR < 0) ||
		(req.PriceYearlyIDR != nil && *req.PriceYearlyIDR < 0) {
		response.BadRequest(c, "harga tidak boleh negatif", "validation")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		response.InternalError(c, "Gagal menyimpan paket")
		return
	}
	defer tx.Rollback(ctx)

	monthly := derefInt64(req.PriceMonthlyIDR)
	yearly := derefInt64(req.PriceYearlyIDR)
	periodDays := 30
	if req.BillingPeriodDays != nil {
		periodDays = *req.BillingPeriodDays
	}
	isActive := req.IsActive == nil || *req.IsActive
	isPublic := req.IsPublic == nil || *req.IsPublic
	gateway := "midtrans"
	if req.FallbackGateway != nil {
		gateway = *req.FallbackGateway
	}
	sortOrder := derefInt(req.SortOrder)

	// Ratio default comes from the operator's CURRENT setting (LiveInt), not a
	// constant and not merely the definition default — otherwise changing the
	// value in the settings UI would have no effect here.
	ratio := int64(1)
	if req.TokenQuotaRatio != nil {
		ratio = *req.TokenQuotaRatio
	} else if h.svc.catalog != nil {
		ratio = h.svc.catalog.LiveInt(ctx, "quota_default_ratio", 1)
	}
	if ratio < 1 {
		response.BadRequest(c, "rasio kuota harus antara 1 dan 1000", "validation")
		return
	}
	if ratio > 1000 {
		response.BadRequest(c, "rasio kuota harus antara 1 dan 1000", "validation")
		return
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO plans (code, name, description, price_monthly_idr, price_yearly_idr,
		                   billing_period_days, is_active, is_public, fallback_gateway, sort_order, token_quota_ratio)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		req.Code, req.Name, req.Description, monthly, yearly, periodDays,
		isActive, isPublic, gateway, sortOrder, ratio).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			response.Conflict(c, "Kode paket sudah dipakai")
			return
		}
		response.InternalError(c, "Gagal menyimpan paket")
		return
	}
	if err := applyEntitlements(ctx, tx, id, derefBoolMap(req.Entitlements)); err != nil {
		response.InternalError(c, "Gagal menyimpan entitlement")
		return
	}
	if err := applyLimits(ctx, tx, id, derefInt64Map(req.Limits)); err != nil {
		response.InternalError(c, "Gagal menyimpan limit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalError(c, "Gagal menyimpan paket")
		return
	}

	// A new plan may be assigned to users immediately; clear any cached
	// entitlement state keyed to it.
	h.invalidatePlan(ctx, id)
	h.publishConfigChanged(ctx, "plans", []string{req.Code})

	h.audit(c, "create", "plan", id.String(), nil,
		map[string]any{"code": req.Code, "name": req.Name, "price_monthly_idr": monthly})
	response.Created(c, "Paket berhasil dibuat", gin.H{"id": id})
}

// UpdatePlan handles PATCH /admin/plans/:id. Only supplied fields change.
func (h *Handler) UpdatePlan(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req planRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	if (req.PriceMonthlyIDR != nil && *req.PriceMonthlyIDR < 0) ||
		(req.PriceYearlyIDR != nil && *req.PriceYearlyIDR < 0) {
		response.BadRequest(c, "harga tidak boleh negatif", "validation")
		return
	}
	if req.TokenQuotaRatio != nil && (*req.TokenQuotaRatio < 1 || *req.TokenQuotaRatio > 1000) {
		response.BadRequest(c, "rasio kuota harus antara 1 dan 1000", "validation")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		response.InternalError(c, "Gagal memperbarui paket")
		return
	}
	defer tx.Rollback(ctx)

	before, err := h.planSnapshot(ctx, tx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFound(c, "Paket tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal memuat paket")
		return
	}

	tag, err := tx.Exec(ctx, `
		UPDATE plans SET
			code                = COALESCE($2, code),
			name                = COALESCE($3, name),
			price_monthly_idr   = COALESCE($4, price_monthly_idr),
			price_yearly_idr    = COALESCE($5, price_yearly_idr),
			billing_period_days = COALESCE($6, billing_period_days),
			is_active           = COALESCE($7, is_active),
			is_public           = COALESCE($8, is_public),
			fallback_gateway    = COALESCE($9, fallback_gateway),
			sort_order          = COALESCE($10, sort_order),
			token_quota_ratio   = COALESCE($11, token_quota_ratio)
		WHERE id = $1`,
		id, nullableStr(req.Code), nullableStr(req.Name), req.PriceMonthlyIDR,
		req.PriceYearlyIDR, req.BillingPeriodDays, req.IsActive, req.IsPublic,
		req.FallbackGateway, req.SortOrder, req.TokenQuotaRatio)
	if err != nil {
		if isUniqueViolation(err) {
			response.Conflict(c, "Kode paket sudah dipakai")
			return
		}
		response.InternalError(c, "Gagal memperbarui paket")
		return
	}
	if tag.RowsAffected() == 0 {
		response.NotFound(c, "Paket tidak ditemukan")
		return
	}
	if req.Entitlements != nil {
		if err := applyEntitlements(ctx, tx, id, *req.Entitlements); err != nil {
			response.InternalError(c, "Gagal menyimpan entitlement")
			return
		}
	}
	if req.Limits != nil {
		if err := applyLimits(ctx, tx, id, *req.Limits); err != nil {
			response.InternalError(c, "Gagal menyimpan limit")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalError(c, "Gagal memperbarui paket")
		return
	}

	// Editing entitlements/limits affects every subscriber of this plan.
	// A single O(1) version bump invalidates them all, instead of leaving every
	// user on stale entitlements until the 24h TTL lapses.
	h.invalidatePlan(ctx, id)
	var changedCodes []string
	if req.Code != "" {
		changedCodes = []string{req.Code}
	}
	h.publishConfigChanged(ctx, "plans", changedCodes)

	after, _ := h.planSnapshot(ctx, h.svc.pool, id)
	h.audit(c, "update", "plan", id.String(), before, after)
	response.OK(c, "Paket berhasil diperbarui", nil)
}

// DeactivatePlan handles DELETE /admin/plans/:id (soft delete).
func (h *Handler) DeactivatePlan(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	ctx := c.Request.Context()

	// Refuse when active subscriptions still reference the plan.
	var active int64
	if err := h.svc.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscriptions WHERE plan_id = $1 AND status IN ('active','trialing','past_due','grace')`,
		id).Scan(&active); err != nil {
		response.InternalError(c, "Gagal memeriksa langganan")
		return
	}
	if active > 0 {
		response.Error(c, http.StatusConflict, "Paket masih dipakai langganan aktif", "PLAN_IN_USE")
		return
	}

	tag, err := h.svc.pool.Exec(ctx, `UPDATE plans SET is_active = false WHERE id = $1`, id)
	if err != nil {
		response.InternalError(c, "Gagal menonaktifkan paket")
		return
	}
	if tag.RowsAffected() == 0 {
		response.NotFound(c, "Paket tidak ditemukan")
		return
	}
	h.invalidatePlan(ctx, id)
	h.publishConfigChanged(ctx, "plans", nil)
	h.audit(c, "deactivate", "plan", id.String(), nil, map[string]any{"is_active": false})
	response.OK(c, "Paket dinonaktifkan", nil)
}

// ─── helpers ──────────────────────────────────────────────────

func (h *Handler) planSnapshot(ctx contextT, q querier, id uuid.UUID) (map[string]any, error) {
	var m map[string]any
	err := q.QueryRow(ctx, `
		SELECT to_jsonb(p) FROM plans p WHERE id = $1`, id).Scan(&m)
	return m, err
}

// invalidatePlan drops cached entitlements for every subscriber of a plan.
func (h *Handler) invalidatePlan(ctx contextT, planID uuid.UUID) {
	if h.svc.entInvalidator != nil {
		h.svc.entInvalidator.InvalidatePlan(ctx, planID.String())
	}
}

func applyEntitlements(ctx contextT, tx pgx.Tx, planID uuid.UUID, m map[string]bool) error {
	for k, v := range m {
		if _, err := tx.Exec(ctx, `
			INSERT INTO plan_entitlements (plan_id, feature_key, is_enabled)
			VALUES ($1,$2,$3)
			ON CONFLICT (plan_id, feature_key)
			DO UPDATE SET is_enabled = EXCLUDED.is_enabled, updated_at = now()`,
			planID, k, v); err != nil {
			return err
		}
	}
	return nil
}

func applyLimits(ctx contextT, tx pgx.Tx, planID uuid.UUID, m map[string]int64) error {
	for k, v := range m {
		if _, err := tx.Exec(ctx, `
			INSERT INTO plan_limits (plan_id, limit_key, limit_value)
			VALUES ($1,$2,$3)
			ON CONFLICT (plan_id, limit_key)
			DO UPDATE SET limit_value = EXCLUDED.limit_value, updated_at = now()`,
			planID, k, v); err != nil {
			return err
		}
	}
	return nil
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func derefBoolMap(p *map[string]bool) map[string]bool {
	if p == nil {
		return nil
	}
	return *p
}

func derefInt64Map(p *map[string]int64) map[string]int64 {
	if p == nil {
		return nil
	}
	return *p
}

func nullableStr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
