package admin

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/internal/featureflag"
	"github.com/fintrack/backend/internal/runtimeconfig"
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
		// Never ship a secret to the client, even encrypted: the UI only needs to
		// know whether one is set. Secret-kind keys are blanked here; the UI shows
		// "sudah diatur" from a separate status endpoint.
		if writableSettings[r.Key].kind == "secret" {
			r.Value = json.RawMessage(`""`)
		}
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
	kind string // "string" | "int" | "enum" | "bool"
	min  int64
	max  int64
	enum []string
	// maxLen caps a string's length (0 = unlimited). Used for short labels so a
	// pasted novel cannot end up on every invoice.
	maxLen int
}

// writableSettings is the allowlist of keys the UI may change. Keys not listed
// (e.g. ai_kill_switch, which has its own audited endpoint) are rejected.
var writableSettings = map[string]settingSpec{
	"payment_gateway_default": {kind: "enum", enum: []string{"manual", "midtrans"}},
	"grace_period_hours":      {kind: "int", min: 0, max: 720},
	"tax_rate_percent":        {kind: "int", min: 0, max: 100},
	"invoice_due_hours":       {kind: "int", min: 1, max: 720},
	// Tax is opt-in: tax_enabled gates whether tax_rate_percent is applied at
	// all, and tax_label names it on the invoice (e.g. "PPN" / "PB1").
	"tax_enabled": {kind: "bool"},
	"tax_label":   {kind: "string", maxLen: 32},
	// Invoice expiry window, as hours + minutes so an operator can set e.g.
	// 1h 30m. Resolved to a due date when an invoice is issued.
	"invoice_expiry_hours":   {kind: "int", min: 0, max: 8760},
	"invoice_expiry_minutes": {kind: "int", min: 0, max: 59},
	// Proof storage backend. Switching takes effect within seconds for new
	// uploads; reads fall back to the other driver so existing proofs stay served.
	"storage_driver": {kind: "enum", enum: []string{"local", "r2"}},
	// AI reserve estimates (P3): tokens held before an AI call, per modality.
	// Editable so an operator can tune the hold to match real usage and avoid
	// over-run clamping at commit.
	"ai_estimate_chat":      {kind: "int", min: 1, max: 1_000_000},
	"ai_estimate_reasoning": {kind: "int", min: 1, max: 1_000_000},
	"ai_estimate_vision":    {kind: "int", min: 1, max: 1_000_000},
	"ai_estimate_voice":     {kind: "int", min: 1, max: 1_000_000},
	"ai_estimate_embedding": {kind: "int", min: 1, max: 1_000_000},
	// Dunning reminder offsets (hours before an invoice is due). Retunable at
	// runtime so the reminder cadence does not require a redeploy.
	"dunning_hours_urgent": {kind: "int", min: 1, max: 720},
	"dunning_hours_soon":   {kind: "int", min: 1, max: 720},
	// AI gateway circuit-breaker tuning. Read at boot and on change; the endpoint
	// breaker guards a whole upstream, the route breaker a single (endpoint,model).
	"breaker_failure_threshold":   {kind: "int", min: 1, max: 100},
	"breaker_cooldown_seconds":    {kind: "int", min: 1, max: 3600},
	"breaker_half_open_probes":    {kind: "int", min: 1, max: 100},
	"route_breaker_threshold":     {kind: "int", min: 1, max: 100},
	"route_breaker_cooldown_secs": {kind: "int", min: 1, max: 3600},
	// Token metering heuristics (local estimation when provider omits usage).
	"metering_chars_per_token":      {kind: "int", min: 1, max: 100},
	"metering_safety_buffer_pct":    {kind: "int", min: 0, max: 100},
	"metering_image_tokens":         {kind: "int", min: 1, max: 10_000},
	"metering_audio_tokens_per_sec": {kind: "int", min: 1, max: 1_000},
	// Transactional email (notifications). A DB value overrides the env default,
	// so SMTP can be changed without a redeploy. `mail_password` is stored
	// AES-256-GCM encrypted (hex) by the handler before it reaches this allowlist.
	"mail_enabled":   {kind: "bool"},
	"mail_host":      {kind: "string"},
	"mail_port":      {kind: "int", min: 1, max: 65535},
	"mail_username":  {kind: "string"},
	"mail_password":  {kind: "secret"},
	"mail_from":      {kind: "string"},
	"mail_from_name": {kind: "string"},
	"mail_use_tls":   {kind: "bool"},
	// Midtrans payment gateway. A DB value overrides the env default, so the
	// gateway can be reconfigured without a redeploy. Secrets are encrypted at
	// rest by the handler (kind "secret").
	"midtrans_enabled":       {kind: "bool"},
	"midtrans_server_key":    {kind: "secret"},
	"midtrans_client_key":    {kind: "string"},
	"midtrans_is_production": {kind: "bool"},
	// Manual bank transfer: the instructions shown to customers. Not secret.
	"manual_bank_name":      {kind: "string"},
	"manual_account_number": {kind: "string"},
	"manual_account_holder": {kind: "string"},
	// Cloudflare R2 credentials. Changing these rebuilds the R2 store within the
	// resolver TTL; `storage_driver` selects whether R2 is actually used.
	"r2_account_id":        {kind: "string"},
	"r2_access_key_id":     {kind: "string"},
	"r2_secret_access_key": {kind: "secret"},
	"r2_bucket":            {kind: "string"},
	"r2_region":            {kind: "string"},
	"r2_endpoint":          {kind: "string"},
	"r2_prefix":            {kind: "string"},
	"r2_public_base_url":   {kind: "string"},
	// AI knobs.
	"ai_reasoning_effort":      {kind: "enum", enum: []string{"", "low", "medium", "high"}},
	"gemini_embedding_api_key": {kind: "secret"},
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

	// Encrypt every `secret`-kind value (AES-256-GCM, hex) so a DB dump never
	// exposes a credential. An empty value means "leave unchanged", which lets
	// the UI submit the rest of the form without re-sending the secret.
	for key, raw := range req.Settings {
		if writableSettings[key].kind != "secret" {
			continue
		}
		var plain string
		if err := json.Unmarshal(raw, &plain); err != nil {
			continue
		}
		if strings.TrimSpace(plain) == "" {
			delete(req.Settings, key)
			delete(after, key)
			continue
		}
		enc, eerr := runtimeconfig.EncryptSecret(plain, h.svc.encKey)
		if eerr != nil {
			response.InternalError(c, "Gagal mengenkripsi kredensial (kunci enkripsi tidak valid)")
			return
		}
		encRaw, _ := json.Marshal(enc)
		req.Settings[key] = encRaw
		after[key] = encRaw
	}

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

	// A storage_driver change must take effect immediately for new uploads.
	if _, ok := req.Settings["storage_driver"]; ok && h.svc.storageSwitcher != nil {
		h.svc.storageSwitcher.Invalidate()
	}
	// Runtime-resolved config (payment, R2, mail, AI) is cached briefly; clear it
	// so the change applies now rather than after the TTL.
	if h.svc.runtimeConfig != nil {
		h.svc.runtimeConfig.Invalidate()
	}

	// Breaker tuning changes must apply to the running gateway immediately.
	if h.svc.breakerReload != nil && hasBreakerKey(req.Settings) {
		h.svc.breakerReload(ctx)
	}

	// Metering heuristic changes must retune the estimator immediately.
	if h.svc.meteringReload != nil && hasMeteringKey(req.Settings) {
		h.svc.meteringReload(ctx)
	}

	h.audit(c, "update", "settings", "bulk", rawMapToAny(before), rawMapToAny(req.Settings))
	response.OK(c, "Pengaturan disimpan", nil)
}

// hasBreakerKey reports whether any AI gateway breaker tuning key is present in
// the update payload, so the reload callback only fires when relevant.
func hasBreakerKey(settings map[string]json.RawMessage) bool {
	for _, k := range []string{
		"breaker_failure_threshold", "breaker_cooldown_seconds", "breaker_half_open_probes",
		"route_breaker_threshold", "route_breaker_cooldown_secs",
	} {
		if _, ok := settings[k]; ok {
			return true
		}
	}
	return false
}

// hasMeteringKey reports whether any token metering heuristic tuning key is present.
func hasMeteringKey(settings map[string]json.RawMessage) bool {
	for _, k := range []string{
		"metering_chars_per_token", "metering_safety_buffer_pct",
		"metering_image_tokens", "metering_audio_tokens_per_sec",
	} {
		if _, ok := settings[k]; ok {
			return true
		}
	}
	return false
}

// StorageStatus handles GET /admin/settings/storage. It reports which proof
// storage driver is active and whether each driver is configured/healthy.
func (h *Handler) StorageStatus(c *gin.Context) {
	if h.svc.storage == nil {
		response.OK(c, "Penyimpanan bukti belum dikonfigurasi", gin.H{
			"active": "local", "default": "local", "local_ready": false,
		})
		return
	}
	st := h.svc.storage.Status(c.Request.Context())
	response.OK(c, "Status penyimpanan berhasil dimuat", gin.H{
		"status":  st,
		"drivers": h.svc.storage.DriverNames(),
	})
}

// settingsSnapshot reads the current values of the writable keys.
func (h *Handler) settingsSnapshot(ctx contextT) (map[string]json.RawMessage, error) {
	rows, err := h.svc.pool.Query(ctx, `
		SELECT key, value FROM settings
		WHERE key = ANY($1)`, []string{
		"payment_gateway_default", "grace_period_hours", "tax_rate_percent", "invoice_due_hours",
		"tax_enabled", "tax_label", "invoice_expiry_hours", "invoice_expiry_minutes",
		"storage_driver",
		"ai_estimate_chat", "ai_estimate_reasoning", "ai_estimate_vision",
		"ai_estimate_voice", "ai_estimate_embedding",
		"dunning_hours_urgent", "dunning_hours_soon",
		"breaker_failure_threshold", "breaker_cooldown_seconds", "breaker_half_open_probes",
		"route_breaker_threshold", "route_breaker_cooldown_secs",
		"metering_chars_per_token", "metering_safety_buffer_pct",
		"metering_image_tokens", "metering_audio_tokens_per_sec",
		"mail_enabled", "mail_host", "mail_port", "mail_username",
		"mail_password", "mail_from", "mail_from_name", "mail_use_tls",
		"midtrans_enabled", "midtrans_server_key", "midtrans_client_key", "midtrans_is_production",
		"manual_bank_name", "manual_account_number", "manual_account_holder",
		"r2_account_id", "r2_access_key_id", "r2_secret_access_key", "r2_bucket",
		"r2_region", "r2_endpoint", "r2_prefix", "r2_public_base_url",
		"ai_reasoning_effort", "gemini_embedding_api_key",
	})
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
		if spec.maxLen > 0 && len([]rune(s)) > spec.maxLen {
			return errStringTooLong
		}
	case "bool":
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return errInvalidBool
		}
	case "secret":
		// Secrets arrive as plaintext from the UI and are encrypted by the
		// handler before storage; here we only check the shape.
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
	errInvalidBool   settingError = "harus berupa true atau false"
	errInvalidEnum   settingError = "nilai tidak dikenal"
	errStringTooLong settingError = "teks terlalu panjang"
	errOutOfRange    settingError = "nilai di luar rentang yang diizinkan"
)
