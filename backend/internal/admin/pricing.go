package admin

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fintrack/backend/pkg/response"
)

// ─── Pricing rules (F5) ───────────────────────────────────────
//
// The pricing rule converts committed tokens to integer Rupiah. Before this,
// the only way to change it was direct SQL (no audit, no UI). These endpoints
// make it fully dynamic from the Super Admin portal.

// knownModalities are the modality keys a pricing rule may scale. Any other key
// is rejected so a typo cannot silently produce a 1.0 fallback.
var knownModalities = map[string]bool{
	"chat": true, "vision": true, "voice": true, "embedding": true, "reasoning": true,
}

type pricingRuleDTO struct {
	ID                  string             `json:"id"`
	RuleName            string             `json:"rule_name"`
	BasePer1kIDR        int64              `json:"base_per_1k_idr"`
	MarkupMultiplier    float64            `json:"markup_multiplier"`
	MarkupPercent       int                `json:"markup_percent"`
	ModalityMultipliers map[string]float64 `json:"modality_multipliers"`
	IsActive            bool               `json:"is_active"`
	CreatedAt           string             `json:"created_at"`
	UpdatedAt           string             `json:"updated_at"`
}

type pricingRuleRequest struct {
	RuleName            string             `json:"rule_name"`
	BasePer1kIDR        *int64             `json:"base_per_1k_idr"`
	MarkupMultiplier    *float64           `json:"markup_multiplier"`
	MarkupPercent       *int               `json:"markup_percent"`
	ModalityMultipliers map[string]float64 `json:"modality_multipliers"`
	IsActive            *bool              `json:"is_active"`
}

const pricingCols = `id, rule_name, base_per_1k_idr, markup_multiplier::float8, markup_percent,
    modality_multipliers, is_active, created_at, updated_at`

func scanPricingRule(row pgx.Row) (*pricingRuleDTO, error) {
	var d pricingRuleDTO
	var createdAt, updatedAt time.Time
	if err := row.Scan(&d.ID, &d.RuleName, &d.BasePer1kIDR, &d.MarkupMultiplier, &d.MarkupPercent,
		&d.ModalityMultipliers, &d.IsActive, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	d.CreatedAt, d.UpdatedAt = createdAt.Format(time.RFC3339), updatedAt.Format(time.RFC3339)
	return &d, nil
}

// validatePricingRequest rejects values that would break billing. Returns a
// user-facing message and an error code, or ok=true when valid.
func validatePricingRequest(req *pricingRuleRequest) (msg, code string, ok bool) {
	if req.BasePer1kIDR != nil && *req.BasePer1kIDR < 0 {
		return "base_per_1k_idr tidak boleh negatif", "validation", false
	}
	if req.MarkupMultiplier != nil && *req.MarkupMultiplier < 0 {
		return "markup_multiplier tidak boleh negatif", "validation", false
	}
	if req.MarkupPercent != nil && (*req.MarkupPercent < 0 || *req.MarkupPercent > 100) {
		return "markup_percent harus antara 0 dan 100", "validation", false
	}
	for k, v := range req.ModalityMultipliers {
		if !knownModalities[k] {
			return "modalitas tidak dikenal: " + k, "validation", false
		}
		if v < 0 {
			return "multiplier modalitas tidak boleh negatif: " + k, "validation", false
		}
	}
	return "", "", true
}

// ListPricingRules handles GET /admin/pricing-rules.
func (h *Handler) ListPricingRules(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := h.svc.pool.Query(ctx, `SELECT `+pricingCols+` FROM pricing_rules ORDER BY created_at ASC`)
	if err != nil {
		response.InternalError(c, "Gagal memuat aturan harga")
		return
	}
	defer rows.Close()

	out := []pricingRuleDTO{}
	for rows.Next() {
		d, err := scanPricingRule(rows)
		if err != nil {
			response.InternalError(c, "Gagal memuat aturan harga")
			return
		}
		out = append(out, *d)
	}
	response.OK(c, "Daftar aturan harga berhasil dimuat", out)
}

// GetActivePricingRule handles GET /admin/pricing-rules/active.
func (h *Handler) GetActivePricingRule(c *gin.Context) {
	row := h.svc.pool.QueryRow(c.Request.Context(),
		`SELECT `+pricingCols+` FROM pricing_rules WHERE is_active = true ORDER BY created_at ASC LIMIT 1`)
	d, err := scanPricingRule(row)
	if err == pgx.ErrNoRows {
		response.NotFound(c, "Belum ada aturan harga aktif")
		return
	}
	if err != nil {
		response.InternalError(c, "Gagal memuat aturan harga")
		return
	}
	response.OK(c, "Aturan harga aktif", d)
}

// CreatePricingRule handles POST /admin/pricing-rules.
func (h *Handler) CreatePricingRule(c *gin.Context) {
	var req pricingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	req.RuleName = strings.TrimSpace(req.RuleName)
	if req.RuleName == "" {
		response.BadRequest(c, "rule_name wajib diisi", "validation")
		return
	}
	if msg, code, ok := validatePricingRequest(&req); !ok {
		response.BadRequest(c, msg, code)
		return
	}

	base := int64(300)
	if req.BasePer1kIDR != nil {
		base = *req.BasePer1kIDR
	}
	mult := 1.25
	if req.MarkupMultiplier != nil {
		mult = *req.MarkupMultiplier
	}
	percent := 0
	if req.MarkupPercent != nil {
		percent = *req.MarkupPercent
	}
	multipliers := req.ModalityMultipliers
	if multipliers == nil {
		multipliers = map[string]float64{"chat": 1.0, "vision": 2.5, "voice": 3.0, "embedding": 0.5, "reasoning": 2.0}
	}
	active := false
	if req.IsActive != nil {
		active = *req.IsActive
	}

	ctx := c.Request.Context()
	var id uuid.UUID
	err := h.svc.pool.QueryRow(ctx, `
		INSERT INTO pricing_rules (rule_name, base_per_1k_idr, markup_multiplier, markup_percent, modality_multipliers, is_active)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		req.RuleName, base, mult, percent, multipliers, active).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			response.Conflict(c, "Nama aturan sudah dipakai")
			return
		}
		response.InternalError(c, "Gagal menyimpan aturan harga")
		return
	}
	if active {
		h.activatePricingRule(c, id)
	}
	h.pricingChanged(c)
	h.audit(c, "create", "pricing_rule", id.String(), nil,
		map[string]any{"rule_name": req.RuleName, "base_per_1k_idr": base, "is_active": active})
	response.Created(c, "Aturan harga dibuat", gin.H{"id": id})
}

// UpdatePricingRule handles PUT /admin/pricing-rules/:id.
func (h *Handler) UpdatePricingRule(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req pricingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	if msg, code, ok := validatePricingRequest(&req); !ok {
		response.BadRequest(c, msg, code)
		return
	}

	ctx := c.Request.Context()
	before, err := scanPricingRule(h.svc.pool.QueryRow(ctx, `SELECT `+pricingCols+` FROM pricing_rules WHERE id = $1`, id))
	if err == pgx.ErrNoRows {
		response.NotFound(c, "Aturan harga tidak ditemukan")
		return
	}
	if err != nil {
		response.InternalError(c, "Gagal memuat aturan harga")
		return
	}

	// COALESCE-with-NULLIF keeps the existing value when a field is omitted,
	// while still allowing an explicit 0.
	tag, err := h.svc.pool.Exec(ctx, `
		UPDATE pricing_rules SET
			rule_name         = COALESCE(NULLIF($2,''), rule_name),
			base_per_1k_idr   = COALESCE($3, base_per_1k_idr),
			markup_multiplier = COALESCE($4, markup_multiplier),
			markup_percent    = COALESCE($5, markup_percent),
			modality_multipliers = COALESCE($6, modality_multipliers),
			is_active         = COALESCE($7, is_active)
		WHERE id = $1`,
		id, req.RuleName, req.BasePer1kIDR, req.MarkupMultiplier, req.MarkupPercent, req.ModalityMultipliers, req.IsActive)
	if err != nil {
		if isUniqueViolation(err) {
			response.Conflict(c, "Nama aturan sudah dipakai")
			return
		}
		response.InternalError(c, "Gagal memperbarui aturan harga")
		return
	}
	if tag.RowsAffected() == 0 {
		response.NotFound(c, "Aturan harga tidak ditemukan")
		return
	}
	after, _ := scanPricingRule(h.svc.pool.QueryRow(ctx, `SELECT `+pricingCols+` FROM pricing_rules WHERE id = $1`, id))
	if after != nil && after.IsActive {
		h.activatePricingRule(c, id)
	}
	h.pricingChanged(c)
	h.auditAny(c, "update", "pricing_rule", id.String(), before, after)
	response.OK(c, "Aturan harga diperbarui", after)
}

// ActivatePricingRule handles POST /admin/pricing-rules/:id/activate.
func (h *Handler) ActivatePricingRule(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	ctx := c.Request.Context()
	var exists bool
	if err := h.svc.pool.QueryRow(ctx, `SELECT true FROM pricing_rules WHERE id = $1`, id).Scan(&exists); err != nil {
		if err == pgx.ErrNoRows {
			response.NotFound(c, "Aturan harga tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal memuat aturan harga")
		return
	}
	h.activatePricingRule(c, id)
	h.pricingChanged(c)
	h.audit(c, "activate", "pricing_rule", id.String(), nil, nil)
	response.OK(c, "Aturan harga diaktifkan", gin.H{"id": id})
}

// DeletePricingRule handles DELETE /admin/pricing-rules/:id. The active rule
// cannot be deleted (deactivate or activate another first).
func (h *Handler) DeletePricingRule(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	ctx := c.Request.Context()
	var active bool
	if err := h.svc.pool.QueryRow(ctx, `SELECT is_active FROM pricing_rules WHERE id = $1`, id).Scan(&active); err != nil {
		if err == pgx.ErrNoRows {
			response.NotFound(c, "Aturan harga tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal memuat aturan harga")
		return
	}
	if active {
		response.Conflict(c, "Aturan aktif tidak dapat dihapus; aktifkan aturan lain lebih dulu")
		return
	}
	if _, err := h.svc.pool.Exec(ctx, `DELETE FROM pricing_rules WHERE id = $1`, id); err != nil {
		response.InternalError(c, "Gagal menghapus aturan harga")
		return
	}
	h.pricingChanged(c)
	h.audit(c, "delete", "pricing_rule", id.String(), nil, nil)
	response.OK(c, "Aturan harga dihapus", nil)
}

// activatePricingRule makes `id` the single active rule in one atomic statement
// (is_active = id match), so there is never a window with two active rules.
func (h *Handler) activatePricingRule(c *gin.Context, id uuid.UUID) {
	if _, err := h.svc.pool.Exec(c.Request.Context(),
		`UPDATE pricing_rules SET is_active = (id = $1)`, id); err != nil {
		response.InternalError(c, "Gagal mengaktifkan aturan harga")
	}
}

// pricingChanged notifies the pricing engine to reload the active rule so the
// new cost applies to the very next commit, without a restart.
func (h *Handler) pricingChanged(c *gin.Context) {
	if h.svc.pricingChanged != nil {
		h.svc.pricingChanged(c.Request.Context())
	}
}
