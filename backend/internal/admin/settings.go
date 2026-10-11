package admin

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/internal/featureflag"
	"github.com/fintrack/backend/pkg/response"
)

// ─── Global settings & feature flags (P06-018) ────────────────

type settingRow struct {
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"`
	Description string          `json:"description"`
	UpdatedBy   *string         `json:"updated_by,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ListSettings handles GET /admin/settings.
func (h *Handler) ListSettings(c *gin.Context) {
	rows, err := h.svc.pool.Query(c.Request.Context(), `
		SELECT key, value, description, updated_by, updated_at
		FROM settings ORDER BY key`)
	if err != nil {
		response.InternalError(c, "Gagal memuat pengaturan")
		return
	}
	defer rows.Close()

	out := []settingRow{}
	for rows.Next() {
		var (
			r         settingRow
			updatedBy *string
			raw       []byte
		)
		if err := rows.Scan(&r.Key, &raw, &r.Description, &updatedBy, &r.UpdatedAt); err != nil {
			response.InternalError(c, "Gagal memuat pengaturan")
			return
		}
		r.Value = json.RawMessage(raw)
		if updatedBy != nil {
			s := *updatedBy
			r.UpdatedBy = &s
		}
		out = append(out, r)
	}
	response.OK(c, "Pengaturan berhasil dimuat", out)
}

// settingSpec declares a writable setting and its validator.
type settingSpec struct {
	kind string // "string" | "int" | "enum"
	min  int64
	max  int64
	enum []string
}

// writableSettings is the allowlist of keys the UI may change. Keys not listed
// (e.g. ai_kill_switch, which has its own audited endpoint) are rejected.
var writableSettings = map[string]settingSpec{
	"payment_gateway_default": {kind: "enum", enum: []string{"manual", "midtrans"}},
	"grace_period_hours":      {kind: "int", min: 0, max: 720},
	"tax_rate_percent":        {kind: "int", min: 0, max: 100},
	"invoice_due_hours":       {kind: "int", min: 1, max: 720},
}

type updateSettingsRequest struct {
	Settings map[string]json.RawMessage `json:"settings"`
}

// UpdateSettings handles PUT /admin/settings (bulk upsert, validated, audited).
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req updateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Settings) == 0 {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	for key, raw := range req.Settings {
		spec, ok := writableSettings[key]
		if !ok {
			response.BadRequest(c, "Pengaturan tidak dapat diubah: "+key, "setting_not_writable")
			return
		}
		if err := validateSetting(spec, raw); err != nil {
			response.BadRequest(c, key+": "+err.Error(), "validation")
			return
		}
	}

	ctx := c.Request.Context()
	actor := actorFrom(c)
	before, _ := h.settingsSnapshot(ctx)
	after := map[string]json.RawMessage{}
	for k, v := range before {
		after[k] = v
	}
	for k, v := range req.Settings {
		after[k] = v
	}

	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		response.InternalError(c, "Gagal menyimpan pengaturan")
		return
	}
	defer tx.Rollback(ctx)

	var actorID any
	if actor.actorID != nil {
		actorID = *actor.actorID
	}
	for key, raw := range req.Settings {
		if _, err := tx.Exec(ctx, `
			INSERT INTO settings (key, value, updated_by, updated_at)
			VALUES ($1, $2, $3, now())
			ON CONFLICT (key) DO UPDATE
			SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()`,
			key, []byte(raw), actorID); err != nil {
			response.InternalError(c, "Gagal menyimpan pengaturan")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalError(c, "Gagal menyimpan pengaturan")
		return
	}

	h.audit(c, "update", "settings", "bulk", rawMapToAny(before), rawMapToAny(req.Settings))
	response.OK(c, "Pengaturan disimpan", nil)
}

// settingsSnapshot reads the current values of the writable keys.
func (h *Handler) settingsSnapshot(ctx contextT) (map[string]json.RawMessage, error) {
	rows, err := h.svc.pool.Query(ctx, `
		SELECT key, value FROM settings
		WHERE key = ANY($1)`, []string{"payment_gateway_default", "grace_period_hours", "tax_rate_percent", "invoice_due_hours"})
	if err != nil {
		return map[string]json.RawMessage{}, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var (
			k   string
			raw []byte
		)
		if rows.Scan(&k, &raw) == nil {
			out[k] = json.RawMessage(raw)
		}
	}
	return out, nil
}

func rawMapToAny(m map[string]json.RawMessage) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = json.RawMessage(v)
	}
	return out
}

func validateSetting(spec settingSpec, raw json.RawMessage) error {
	switch spec.kind {
	case "int":
		var n int64
		if err := json.Unmarshal(raw, &n); err != nil {
			return errInvalidInt
		}
		if n < spec.min || n > spec.max {
			return errOutOfRange
		}
	case "enum":
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return errInvalidString
		}
		for _, allowed := range spec.enum {
			if s == allowed {
				return nil
			}
		}
		return errInvalidEnum
	case "string":
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return errInvalidString
		}
	}
	return nil
}

// ─── Feature flags ────────────────────────────────────────────

// ListFeatureFlags handles GET /admin/feature-flags.
func (h *Handler) ListFeatureFlags(c *gin.Context) {
	if h.svc.flags == nil {
		response.OK(c, "Feature flag tidak aktif", []featureflag.Flag{})
		return
	}
	flags, err := h.svc.flags.List(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Gagal memuat feature flag")
		return
	}
	response.OK(c, "Feature flag berhasil dimuat", flags)
}

type flagRequest struct {
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
	Rollout     int            `json:"rollout"`
	Rules       map[string]any `json:"rules"`
}

// UpdateFeatureFlag handles PUT /admin/feature-flags.
func (h *Handler) UpdateFeatureFlag(c *gin.Context) {
	if h.svc.flags == nil {
		response.Error(c, 503, "Feature flag tidak aktif", "FLAGS_UNAVAILABLE")
		return
	}
	var req flagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	if req.Key == "" {
		response.BadRequest(c, "key wajib diisi", "validation")
		return
	}
	if req.Rollout < 0 || req.Rollout > 100 {
		response.BadRequest(c, "rollout harus 0-100", "validation")
		return
	}
	if req.Name == "" {
		req.Name = req.Key
	}

	ctx := c.Request.Context()
	before, _ := h.svc.flags.GetByKey(ctx, req.Key)
	flag := featureflag.Flag{
		Key: req.Key, Name: req.Name, Description: req.Description,
		IsEnabled: req.Enabled, Rollout: req.Rollout, Rules: req.Rules,
	}
	actor := actorFrom(c)
	if err := h.svc.flags.Set(ctx, flag, actor.actorID); err != nil {
		response.InternalError(c, "Gagal menyimpan feature flag")
		return
	}
	var beforeAny, afterAny any
	if before != nil {
		beforeAny = before
	}
	afterAny = flag
	h.auditAny(c, "update", "feature_flag", req.Key, beforeAny, afterAny)
	response.OK(c, "Feature flag disimpan", flag)
}

// ─── errors ───────────────────────────────────────────────────

type settingError string

func (e settingError) Error() string { return string(e) }

const (
	errInvalidInt    settingError = "harus berupa bilangan bulat"
	errInvalidString settingError = "harus berupa teks"
	errInvalidEnum   settingError = "nilai tidak dikenal"
	errOutOfRange    settingError = "nilai di luar rentang yang diizinkan"
)
