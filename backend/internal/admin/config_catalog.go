package admin

import (
	"context"
	"encoding/json"
	"math"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/pkg/response"
)

// ─── Dynamic configuration catalog (P-13) ─────────────────────

// SettingDefinitions handles GET /admin/setting-definitions. It returns the
// data-driven catalog so the settings UI renders whatever the DB defines,
// instead of a hardcoded list compiled into the frontend.
func (h *Handler) SettingDefinitions(c *gin.Context) {
	if h.svc.catalog == nil {
		response.InternalError(c, "Katalog konfigurasi tidak tersedia")
		return
	}
	response.OK(c, "Katalog setting", h.svc.catalog.Definitions(c.Request.Context()))
}

// MetricDefinitions handles GET /admin/metric-definitions.
func (h *Handler) MetricDefinitions(c *gin.Context) {
	if h.svc.catalog == nil {
		response.InternalError(c, "Katalog metrik tidak tersedia")
		return
	}
	response.OK(c, "Katalog metrik", h.svc.catalog.MetricDefinitions(c.Request.Context()))
}

// FeatureDefinitions handles GET /admin/feature-definitions.
func (h *Handler) FeatureDefinitions(c *gin.Context) {
	if h.svc.catalog == nil {
		response.InternalError(c, "Katalog fitur tidak tersedia")
		return
	}
	response.OK(c, "Katalog fitur", h.svc.catalog.FeatureDefinitions(c.Request.Context()))
}

// Resolvers handles GET /admin/resolvers. It lists the usage-resolver
// CAPABILITIES the code provides, so an operator can point a metric at one via
// a dropdown. Adding a capability needs a deploy; selecting one does not.
func (h *Handler) Resolvers(c *gin.Context) {
	response.OK(c, "Daftar resolver", gin.H{"resolvers": availableResolvers()})
}

type resolverInfo struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	Unit           string `json:"unit"`
	RequiresPeriod bool   `json:"requires_period"`
}

// availableResolvers is the authoritative list of resolver capabilities. It is
// code-defined because the code is what computes each one; the operator picks
// from this list when creating a metric.
func availableResolvers() []resolverInfo {
	return []resolverInfo{
		{Key: "token_period_usage", Label: "Pemakaian kuota per siklus", Unit: "kuota", RequiresPeriod: true},
		{Key: "ai_requests_in_period", Label: "Permintaan AI per siklus", Unit: "permintaan", RequiresPeriod: true},
		{Key: "active_transactions", Label: "Transaksi aktif", Unit: "transaksi", RequiresPeriod: false},
		{Key: "vision_scans_in_period", Label: "Pemindaian struk per siklus", Unit: "scan", RequiresPeriod: true},
		{Key: "voice_requests_per_day", Label: "Permintaan suara per hari", Unit: "permintaan", RequiresPeriod: false},
	}
}

// ConfigReload handles POST /admin/config/reload. It forces a catalog reload for
// changes made OUTSIDE the API (e.g. a manual SQL edit), which would otherwise
// only be picked up by the fallback TTL.
func (h *Handler) ConfigReload(c *gin.Context) {
	if h.svc.catalog == nil {
		response.InternalError(c, "Katalog konfigurasi tidak tersedia")
		return
	}
	h.svc.catalog.Invalidate(c.Request.Context())
	h.publishConfigChanged(c.Request.Context(), "all", nil)
	h.audit(c, "reload", "config_catalog", "all", nil, nil)
	response.OK(c, "Katalog dimuat ulang", nil)
}

// publishConfigChanged notifies clients (and peers) that configuration changed,
// so the UI reloads immediately instead of waiting for a TTL. Best-effort: a
// nil emitter is a no-op, and the version bump is the authoritative signal.
func (h *Handler) publishConfigChanged(ctx context.Context, scope string, keys []string) {
	if h.svc.emitter == nil {
		return
	}
	h.svc.emitter.PublishConfigChanged(ctx, scope, keys)
}

// ─── Quota estimator (P-9) ────────────────────────────────────

type estimateRequest struct {
	TokenQuotaRatio  int64  `json:"token_quota_ratio"`
	AITokensPerCycle int64  `json:"ai_tokens_per_cycle"`
	AvgTokensPerChat *int64 `json:"avg_tokens_per_chat"`
	VoiceSeconds     *int64 `json:"voice_seconds"`
	PriceRuleID      string `json:"price_rule_id"`
}

// PlanEstimate handles POST /admin/plans/estimate. It answers "how many chats
// does this quota allow, and what does it cost?" — a read-only calculator so an
// operator can see the impact of a ratio/quota change before saving it.
func (h *Handler) PlanEstimate(c *gin.Context) {
	var req estimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	if req.TokenQuotaRatio < 1 || req.AITokensPerCycle < 1 {
		response.BadRequest(c, "rasio dan kuota harus >= 1", "validation")
		return
	}
	avg := int64(8000)
	if req.AvgTokensPerChat != nil && *req.AvgTokensPerChat > 0 {
		avg = *req.AvgTokensPerChat
	}

	ctx := c.Request.Context()

	// Rounding mode + minimum charge come from the operator's CURRENT settings
	// (LiveString/LiveInt) so the calculator reflects exactly what the ledger
	// will apply — not merely the definition defaults.
	mode := billing.RoundCeil
	minCharge := int64(1)
	if h.svc.catalog != nil {
		if v := h.svc.catalog.LiveString(ctx, "quota_rounding_mode", string(billing.RoundCeil)); v != "" {
			mode = billing.ParseRoundingMode(v)
		}
		if v := h.svc.catalog.LiveInt(ctx, "quota_min_charge_units", 1); v >= 0 {
			minCharge = v
		}
	}

	unitPerChat := billing.ApplyMinCharge(billing.RoundUnits(avg, req.TokenQuotaRatio, mode), minCharge, avg)
	// Guard against a zero unit cost: with mode=floor and no minimum charge, a
	// tiny average rounds to 0, which would divide by zero and emit +Inf (invalid
	// JSON that breaks the client). Treat it as "quota cannot cover one chat".
	if unitPerChat < 1 {
		unitPerChat = 1
	}
	chatsExact := float64(req.AITokensPerCycle) / float64(unitPerChat)
	chats := int64(math.Floor(chatsExact))
	leftover := req.AITokensPerCycle - chats*unitPerChat

	// Without the ratio (1:1) — for the comparison panel.
	unitNoRatio := billing.ApplyMinCharge(billing.RoundUnits(avg, 1, mode), minCharge, avg)
	if unitNoRatio < 1 {
		unitNoRatio = 1
	}
	chatsNoRatio := float64(req.AITokensPerCycle) / float64(unitNoRatio)

	// Cost comes from the active (or named) pricing rule, ALWAYS on raw tokens.
	rule, err := h.activeOrNamedRule(ctx, req.PriceRuleID)
	if err != nil {
		response.InternalError(c, "Gagal memuat aturan harga")
		return
	}
	engine := billing.NewPricingEngine(rule)
	sell := engine.CostIDR(avg, billing.ModalityChat)
	provider := engine.ProviderCostIDR(avg, billing.ModalityChat)
	margin := sell - provider
	marginPct := 0.0
	if sell > 0 {
		marginPct = float64(margin) / float64(sell) * 100
	}

	voiceSec := int64(15)
	if req.VoiceSeconds != nil && *req.VoiceSeconds > 0 {
		voiceSec = *req.VoiceSeconds
	}
	voiceTokensPerSec := int64(25)
	voiceOverhead := int64(50)
	voiceHeadroom := int64(150)
	if h.svc.catalog != nil {
		if v := h.svc.catalog.LiveInt(ctx, "ai_estimate_voice_tokens_per_sec", 25); v > 0 {
			voiceTokensPerSec = v
		}
		if v := h.svc.catalog.LiveInt(ctx, "ai_estimate_voice_overhead", 50); v >= 0 {
			voiceOverhead = v
		}
		if v := h.svc.catalog.LiveInt(ctx, "ai_estimate_voice_output_headroom", 150); v >= 0 {
			voiceHeadroom = v
		}
	}
	voiceRaw := voiceSec*voiceTokensPerSec + voiceOverhead + voiceHeadroom
	unitPerVoice := billing.ApplyMinCharge(billing.RoundUnits(voiceRaw, req.TokenQuotaRatio, mode), minCharge, voiceRaw)
	if unitPerVoice < 1 {
		unitPerVoice = 1
	}
	callsVoice := req.AITokensPerCycle / unitPerVoice

	visionImageTokens := int64(800)
	visionHeadroom := int64(400)
	if h.svc.catalog != nil {
		if v := h.svc.catalog.LiveInt(ctx, "ai_estimate_vision_image_tokens", 800); v > 0 {
			visionImageTokens = v
		}
		if v := h.svc.catalog.LiveInt(ctx, "ai_estimate_vision_output_headroom", 400); v >= 0 {
			visionHeadroom = v
		}
	}
	visionRaw := int64(100) + visionImageTokens + visionHeadroom
	unitPerVision := billing.ApplyMinCharge(billing.RoundUnits(visionRaw, req.TokenQuotaRatio, mode), minCharge, visionRaw)
	if unitPerVision < 1 {
		unitPerVision = 1
	}
	callsVision := req.AITokensPerCycle / unitPerVision

	warnings := estimateWarnings(chatsExact, margin, marginPct, req.TokenQuotaRatio)

	data := gin.H{
		"input": gin.H{
			"token_quota_ratio":   req.TokenQuotaRatio,
			"ai_tokens_per_cycle": req.AITokensPerCycle,
			"avg_tokens_per_chat": avg,
			"rounding_mode":       string(mode),
			"min_charge_units":    minCharge,
		},
		"quota": gin.H{
			"unit_per_chat":         unitPerChat,
			"chats_per_cycle":       chats,
			"chats_per_cycle_exact": round2(chatsExact),
			"leftover_units":        leftover,
		},
		"cost": gin.H{
			"price_unit":                 ruleUnit(rule),
			"sell_per_1k_idr":            ruleSellPer1k(rule),
			"provider_cost_per_1k_idr":   ruleProviderCost(rule),
			"sell_per_chat_idr":          sell,
			"provider_cost_per_chat_idr": provider,
			"margin_per_chat_idr":        margin,
			"margin_percent":             round2(marginPct),
			"cost_per_full_cycle_idr":    chats * provider,
			"revenue_per_full_cycle_idr": chats * sell,
			"margin_per_full_cycle_idr":  chats * margin,
		},
		"comparison": gin.H{
			"without_ratio": gin.H{"chats_per_cycle": round2(chatsNoRatio)},
			"with_ratio":    gin.H{"chats_per_cycle": round2(chatsExact)},
		},
		"modalities": gin.H{
			"voice": gin.H{
				"avg_seconds":     voiceSec,
				"raw_tokens":      voiceRaw,
				"unit_per_call":   unitPerVoice,
				"calls_per_cycle": callsVoice,
				"sell_idr":        engine.CostIDR(voiceRaw, billing.ModalityVoice),
				"cost_idr":        engine.ProviderCostIDR(voiceRaw, billing.ModalityVoice),
			},
			"vision": gin.H{
				"raw_tokens":      visionRaw,
				"unit_per_call":   unitPerVision,
				"calls_per_cycle": callsVision,
				"sell_idr":        engine.CostIDR(visionRaw, billing.ModalityVision),
				"cost_idr":        engine.ProviderCostIDR(visionRaw, billing.ModalityVision),
			},
		},
		"warnings": warnings,
	}
	response.OK(c, "Estimasi berhasil dihitung", data)
}

// activeOrNamedRule loads the named rule, or the active one when no id is given.
// It reuses the pricing rule query so the estimator always reflects what the
// engine actually charges.
func (h *Handler) activeOrNamedRule(ctx context.Context, id string) (*billing.PricingRule, error) {
	if id != "" {
		if pid, err := uuid.Parse(id); err == nil {
			row := h.svc.pool.QueryRow(ctx, `SELECT `+pricingCols+` FROM pricing_rules WHERE id = $1`, pid)
			if r, err := scanPricingRuleModel(row); err == nil && r != nil {
				return r, nil
			}
		}
	}
	row := h.svc.pool.QueryRow(ctx, `SELECT `+pricingCols+` FROM pricing_rules WHERE is_active = true ORDER BY created_at DESC LIMIT 1`)
	return scanPricingRuleModel(row)
}

func estimateWarnings(chats float64, margin int64, marginPct float64, ratio int64) []string {
	var out []string
	switch {
	case chats < 1:
		out = append(out, "Kuota tidak cukup untuk satu chat. Naikkan kuota atau rasio.")
	case chats < 3:
		out = append(out, "Rasio ini memberi kurang dari 3 chat per siklus. Pertimbangkan menaikkan kuota.")
	}
	if margin < 0 {
		out = append(out, "Harga jual di bawah biaya provider. Periksa aturan harga.")
	} else if marginPct > 0 && marginPct < 20 {
		out = append(out, "Margin di bawah 20%. Pertimbangkan menyesuaikan markup.")
	}
	if ratio > 20 {
		out = append(out, "Rasio sangat tinggi; user mungkin tidak menyadari biaya nyata.")
	}
	return out
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

func ruleUnit(r *billing.PricingRule) string {
	if r == nil || r.PriceUnit == "" {
		return billing.PriceUnitPer1k
	}
	return r.PriceUnit
}

func ruleSellPer1k(r *billing.PricingRule) int64 {
	if r == nil {
		return 0
	}
	return r.SellPricePer1kIDR()
}

func ruleProviderCost(r *billing.PricingRule) int64 {
	if r == nil {
		return 0
	}
	return r.ProviderCostPer1kIDR
}

// enumString extracts a plain string from a JSON-encoded value (e.g. `"ceil"`).
func enumString(raw []byte) string {
	if len(raw) < 2 {
		return ""
	}
	s := string(raw)
	if s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// settingKeys extracts the keys from a settings map for the config_changed
// payload, so clients can reload only what changed.
func settingKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// ─── Catalog writes (P-13) ────────────────────────────────────

type metricUpsertRequest struct {
	Key           string `json:"key" binding:"required"`
	Label         string `json:"label" binding:"required"`
	Unit          string `json:"unit"`
	Description   string `json:"description"`
	IsEnforced    bool   `json:"is_enforced"`
	ResolverKey   string `json:"resolver_key"`
	RatioAdjusted bool   `json:"ratio_adjusted"`
	PerCycle      bool   `json:"per_cycle"`
	GroupKey      string `json:"group_key"`
	SortOrder     int    `json:"sort_order"`
}

// knownResolverKeys is the set of resolver capabilities the code provides. A
// metric may only reference one of these, so an operator cannot configure a
// metric the backend cannot compute.
var knownResolverKeys = map[string]bool{
	"token_period_usage":     true,
	"ai_requests_in_period":  true,
	"active_transactions":    true,
	"vision_scans_in_period": true,
	"voice_requests_per_day": true,
}

// UpsertMetricDefinition handles POST /admin/metric-definitions. It validates
// that an enforced metric names a resolver the backend actually provides.
func (h *Handler) UpsertMetricDefinition(c *gin.Context) {
	var req metricUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	if h.svc.catalog == nil {
		response.InternalError(c, "Katalog konfigurasi tidak tersedia")
		return
	}
	if req.IsEnforced && !knownResolverKeys[req.ResolverKey] {
		response.BadRequest(c, "Metrik yang ditegakkan wajib memilih resolver yang dikenal", "validation")
		return
	}
	if req.GroupKey == "" {
		req.GroupKey = "kuota"
	}
	err := h.svc.catalog.UpsertMetric(c.Request.Context(), MetricDef{
		Key: req.Key, Label: req.Label, Unit: req.Unit, Description: req.Description,
		IsEnforced: req.IsEnforced, ResolverKey: req.ResolverKey,
		RatioAdjusted: req.RatioAdjusted, PerCycle: req.PerCycle,
		GroupKey: req.GroupKey, SortOrder: req.SortOrder,
	})
	if err != nil {
		response.InternalError(c, "Gagal menyimpan metrik")
		return
	}
	h.audit(c, "upsert", "metric_definition", req.Key, nil, map[string]any{"key": req.Key})
	h.publishConfigChanged(c.Request.Context(), "metrics", []string{req.Key})
	response.OK(c, "Metrik disimpan", gin.H{"key": req.Key})
}

// DeleteMetricDefinition handles DELETE /admin/metric-definitions/:key (soft).
func (h *Handler) DeleteMetricDefinition(c *gin.Context) {
	key := c.Param("key")
	if h.svc.catalog == nil {
		response.InternalError(c, "Katalog konfigurasi tidak tersedia")
		return
	}
	if err := h.svc.catalog.DeactivateMetric(c.Request.Context(), key); err != nil {
		response.InternalError(c, "Gagal menonaktifkan metrik")
		return
	}
	h.audit(c, "deactivate", "metric_definition", key, nil, nil)
	h.publishConfigChanged(c.Request.Context(), "metrics", []string{key})
	response.OK(c, "Metrik dinonaktifkan", nil)
}

type featureUpsertRequest struct {
	Key         string `json:"key" binding:"required"`
	Label       string `json:"label" binding:"required"`
	Description string `json:"description"`
	GroupKey    string `json:"group_key"`
	SortOrder   int    `json:"sort_order"`
}

// UpsertFeatureDefinition handles POST /admin/feature-definitions.
func (h *Handler) UpsertFeatureDefinition(c *gin.Context) {
	var req featureUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	if h.svc.catalog == nil {
		response.InternalError(c, "Katalog konfigurasi tidak tersedia")
		return
	}
	if req.GroupKey == "" {
		req.GroupKey = "fitur"
	}
	err := h.svc.catalog.UpsertFeature(c.Request.Context(), FeatureDef{
		Key: req.Key, Label: req.Label, Description: req.Description,
		GroupKey: req.GroupKey, SortOrder: req.SortOrder,
	})
	if err != nil {
		response.InternalError(c, "Gagal menyimpan fitur")
		return
	}
	h.audit(c, "upsert", "feature_definition", req.Key, nil, map[string]any{"key": req.Key})
	h.publishConfigChanged(c.Request.Context(), "features", []string{req.Key})
	response.OK(c, "Fitur disimpan", gin.H{"key": req.Key})
}

// DeleteFeatureDefinition handles DELETE /admin/feature-definitions/:key (soft).
func (h *Handler) DeleteFeatureDefinition(c *gin.Context) {
	key := c.Param("key")
	if h.svc.catalog == nil {
		response.InternalError(c, "Katalog konfigurasi tidak tersedia")
		return
	}
	if err := h.svc.catalog.DeactivateFeature(c.Request.Context(), key); err != nil {
		response.InternalError(c, "Gagal menonaktifkan fitur")
		return
	}
	h.audit(c, "deactivate", "feature_definition", key, nil, nil)
	h.publishConfigChanged(c.Request.Context(), "features", []string{key})
	response.OK(c, "Fitur dinonaktifkan", nil)
}
