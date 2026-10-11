package admin

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/infrastructure/crypto"
	"github.com/fintrack/backend/pkg/response"
)

// ─── AI Gateway operations (P06-016) ──────────────────────────

type endpointDTO struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	BaseURL      string   `json:"base_url"`
	Provider     string   `json:"provider"`
	Priority     int      `json:"priority"`
	Status       string   `json:"status"`
	Health       string   `json:"health"`
	Modalities   []string `json:"modalities"`
	Weight       int      `json:"weight"`
	LastCheckAt  *string  `json:"last_check_at,omitempty"`
	LastError    *string  `json:"last_error,omitempty"`
	HasKey       bool     `json:"has_api_key"`
	BreakerState string   `json:"breaker_state"`
}

// ListGatewayEndpoints handles GET /admin/ai-gateway/endpoints.
func (h *Handler) ListGatewayEndpoints(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := h.svc.pool.Query(ctx, `
		SELECT id, name, base_url, provider, priority, status, health, modalities, weight,
		       last_check_at, last_error, api_key_encrypted IS NOT NULL
		FROM ai_endpoints ORDER BY priority ASC, name ASC`)
	if err != nil {
		response.InternalError(c, "Gagal memuat endpoint AI")
		return
	}
	defer rows.Close()

	out := []endpointDTO{}
	for rows.Next() {
		var e endpointDTO
		var lastCheck *time.Time
		if err := rows.Scan(&e.ID, &e.Name, &e.BaseURL, &e.Provider, &e.Priority, &e.Status,
			&e.Health, &e.Modalities, &e.Weight, &lastCheck, &e.LastError, &e.HasKey); err != nil {
			response.InternalError(c, "Gagal memuat endpoint AI")
			return
		}
		if lastCheck != nil {
			t := lastCheck.Format(time.RFC3339)
			e.LastCheckAt = &t
		}
		if h.svc.aigw != nil {
			e.BreakerState = h.svc.aigw.Breaker().State(e.ID)
		}
		out = append(out, e)
	}
	response.OK(c, "Daftar endpoint AI berhasil dimuat", out)
}

type gatewayEndpointRequest struct {
	Name       string   `json:"name"`
	BaseURL    string   `json:"base_url"`
	Provider   string   `json:"provider"`
	Priority   *int     `json:"priority"`
	Status     *string  `json:"status"`
	Modalities []string `json:"modalities"`
	Weight     *int     `json:"weight"`
	APIKey     *string  `json:"api_key"`
}

// CreateGatewayEndpoint handles POST /admin/ai-gateway/endpoints.
func (h *Handler) CreateGatewayEndpoint(c *gin.Context) {
	var req gatewayEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	if req.Name == "" || req.BaseURL == "" {
		response.BadRequest(c, "name dan base_url wajib diisi", "validation")
		return
	}
	status := "active"
	if req.Status != nil {
		status = *req.Status
	}
	if !validEndpointStatus(status) {
		response.BadRequest(c, "status tidak valid", "validation")
		return
	}
	provider := req.Provider
	if provider == "" {
		provider = "9router"
	}
	priority := derefInt(req.Priority)
	if priority == 0 {
		priority = 1
	}
	weight := derefInt(req.Weight)
	if weight == 0 {
		weight = 100
	}
	modalities := req.Modalities
	if len(modalities) == 0 {
		modalities = []string{"chat", "vision", "voice", "embedding", "reasoning"}
	}

	enc, err := h.encryptKey(req.APIKey)
	if err != nil {
		response.InternalError(c, "Gagal mengenkripsi API key")
		return
	}

	ctx := c.Request.Context()
	var id uuid.UUID
	err = h.svc.pool.QueryRow(ctx, `
		INSERT INTO ai_endpoints (name, base_url, api_key_encrypted, provider, priority, status, modalities, weight)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		req.Name, req.BaseURL, enc, provider, priority, status, modalities, weight).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			response.Conflict(c, "Nama endpoint sudah dipakai")
			return
		}
		response.InternalError(c, "Gagal menyimpan endpoint AI")
		return
	}
	h.invalidateRoutes()
	h.audit(c, "create", "ai_route", id.String(), nil,
		map[string]any{"name": req.Name, "base_url": req.BaseURL, "provider": provider, "status": status})
	response.Created(c, "Endpoint AI dibuat", gin.H{"id": id})
}

// UpdateGatewayEndpoint handles PUT /admin/ai-gateway/endpoints/:id.
func (h *Handler) UpdateGatewayEndpoint(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req gatewayEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	if req.Status != nil && !validEndpointStatus(*req.Status) {
		response.BadRequest(c, "status tidak valid", "validation")
		return
	}
	enc, err := h.encryptKey(req.APIKey)
	if err != nil {
		response.InternalError(c, "Gagal mengenkripsi API key")
		return
	}

	ctx := c.Request.Context()
	tag, err := h.svc.pool.Exec(ctx, `
		UPDATE ai_endpoints SET
			name              = COALESCE(NULLIF($2,''), name),
			base_url          = COALESCE(NULLIF($3,''), base_url),
			provider          = COALESCE(NULLIF($4,''), provider),
			priority          = COALESCE($5, priority),
			status            = COALESCE($6, status),
			weight            = COALESCE($7, weight),
			modalities        = COALESCE($8, modalities),
			api_key_encrypted = COALESCE($9, api_key_encrypted)
		WHERE id = $1`,
		id, req.Name, req.BaseURL, req.Provider, req.Priority, req.Status, req.Weight,
		nullableStrSlice(req.Modalities), enc)
	if err != nil {
		if isUniqueViolation(err) {
			response.Conflict(c, "Nama endpoint sudah dipakai")
			return
		}
		response.InternalError(c, "Gagal memperbarui endpoint AI")
		return
	}
	if tag.RowsAffected() == 0 {
		response.NotFound(c, "Endpoint tidak ditemukan")
		return
	}
	h.invalidateRoutes()
	h.audit(c, "update", "ai_route", id.String(), nil, map[string]any{
		"name": req.Name, "status": req.Status, "priority": req.Priority, "weight": req.Weight,
	})
	response.OK(c, "Endpoint AI diperbarui", nil)
}

// DeleteGatewayEndpoint handles DELETE /admin/ai-gateway/endpoints/:id.
func (h *Handler) DeleteGatewayEndpoint(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	ctx := c.Request.Context()

	var used int64
	if err := h.svc.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ai_routes WHERE endpoint_id = $1 AND is_active`, id).Scan(&used); err != nil {
		response.InternalError(c, "Gagal memeriksa rute")
		return
	}
	if used > 0 {
		response.Error(c, http.StatusConflict,
			"Endpoint masih dipakai rute aktif; nonaktifkan endpoint atau pindahkan rute dulu",
			"ENDPOINT_IN_USE")
		return
	}
	tag, err := h.svc.pool.Exec(ctx, `UPDATE ai_endpoints SET status = 'inactive' WHERE id = $1`, id)
	if err != nil {
		response.InternalError(c, "Gagal menonaktifkan endpoint")
		return
	}
	if tag.RowsAffected() == 0 {
		response.NotFound(c, "Endpoint tidak ditemukan")
		return
	}
	h.invalidateRoutes()
	h.audit(c, "deactivate", "ai_route", id.String(), nil, map[string]any{"status": "inactive"})
	response.OK(c, "Endpoint dinonaktifkan", nil)
}

type routeDTO struct {
	ID         string `json:"id,omitempty"`
	UseCase    string `json:"use_case"`
	EndpointID string `json:"endpoint_id"`
	Model      string `json:"model"`
	// Combo groups routes into a failover set. Routes in the same combo are
	// tried in priority order; the gateway rotates between combos by weight.
	Combo      string `json:"combo"`
	Priority   int    `json:"priority"`
	TimeoutMS  int    `json:"timeout_ms"`
	MaxRetries int    `json:"max_retries"`
	IsActive   bool   `json:"is_active"`
}

// ListRoutes handles GET /admin/ai-gateway/routes.
func (h *Handler) ListRoutes(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := h.svc.pool.Query(ctx, `
		SELECT id, use_case, endpoint_id, model, combo, priority, timeout_ms, max_retries, is_active
		FROM ai_routes ORDER BY use_case, combo, priority`)
	if err != nil {
		response.InternalError(c, "Gagal memuat rute AI")
		return
	}
	defer rows.Close()

	out := []routeDTO{}
	for rows.Next() {
		var r routeDTO
		if err := rows.Scan(&r.ID, &r.UseCase, &r.EndpointID, &r.Model, &r.Combo, &r.Priority,
			&r.TimeoutMS, &r.MaxRetries, &r.IsActive); err != nil {
			response.InternalError(c, "Gagal memuat rute AI")
			return
		}
		out = append(out, r)
	}
	response.OK(c, "Daftar rute AI berhasil dimuat", out)
}

// PutRoutes handles PUT /admin/ai-gateway/routes: replaces the full route set
// atomically so the UI can save the whole use-case matrix in one request.
func (h *Handler) PutRoutes(c *gin.Context) {
	var body []routeDTO
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	validUseCase := map[string]bool{"chat": true, "vision": true, "voice": true, "embedding": true, "reasoning": true}
	seen := make(map[string]bool, len(body))
	endpointIDs := make([]uuid.UUID, 0, len(body))
	for _, r := range body {
		if !validUseCase[r.UseCase] {
			response.BadRequest(c, "use_case tidak valid: "+r.UseCase, "validation")
			return
		}
		eid, err := uuid.Parse(r.EndpointID)
		if err != nil {
			response.BadRequest(c, "endpoint_id tidak valid", "invalid_payload")
			return
		}
		if r.TimeoutMS < 0 || r.MaxRetries < 0 {
			response.BadRequest(c, "timeout_ms/max_retries tidak boleh negatif", "validation")
			return
		}
		combo := normalizeCombo(r.Combo)
		key := r.UseCase + "|" + eid.String() + "|" + r.Model + "|" + combo
		if seen[key] {
			response.BadRequest(c, "rute duplikat: "+r.UseCase+"/"+r.Model+"/"+combo, "duplicate_route")
			return
		}
		seen[key] = true
		endpointIDs = append(endpointIDs, eid)
	}

	ctx := c.Request.Context()
	if h.svc.pool != nil && len(endpointIDs) > 0 {
		var activeCount int
		if err := h.svc.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM ai_endpoints WHERE id = ANY($1) AND status = 'active'`,
			endpointIDs).Scan(&activeCount); err != nil {
			response.InternalError(c, "Gagal memvalidasi endpoint")
			return
		}
		if activeCount != len(uniqueIDs(endpointIDs)) {
			response.BadRequest(c, "endpoint_id tidak ditemukan atau tidak aktif", "invalid_endpoint")
			return
		}
	}

	if h.svc.pool == nil {
		response.InternalError(c, "database pool tidak tersedia")
		return
	}
	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		response.InternalError(c, "Gagal menyimpan rute")
		return
	}
	defer tx.Rollback(ctx)

	var before []routeDTO
	brows, _ := tx.Query(ctx, `SELECT id, use_case, endpoint_id, model, combo, priority, timeout_ms, max_retries, is_active FROM ai_routes ORDER BY use_case, combo, priority`)
	if brows != nil {
		for brows.Next() {
			var r routeDTO
			if brows.Scan(&r.ID, &r.UseCase, &r.EndpointID, &r.Model, &r.Combo, &r.Priority,
				&r.TimeoutMS, &r.MaxRetries, &r.IsActive) == nil {
				before = append(before, r)
			}
		}
		brows.Close()
	}

	if _, err := tx.Exec(ctx, `DELETE FROM ai_routes`); err != nil {
		response.InternalError(c, "Gagal menyimpan rute")
		return
	}
	for i, r := range body {
		priority := r.Priority
		if priority == 0 {
			priority = i + 1
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_routes (use_case, endpoint_id, model, combo, priority, timeout_ms, max_retries, is_active)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			r.UseCase, r.EndpointID, r.Model, normalizeCombo(r.Combo), priority,
			defaultInt(r.TimeoutMS, 30000), defaultInt(r.MaxRetries, 2), r.IsActive); err != nil {
			response.InternalError(c, "Gagal menyimpan rute")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalError(c, "Gagal menyimpan rute")
		return
	}
	h.invalidateRoutes()
	h.audit(c, "update", "ai_route", "routes", map[string]any{"routes": before},
		map[string]any{"routes": body})
	response.OK(c, "Rute AI disimpan", nil)
}

// normalizeCombo maps an empty combo to the default group, so routes created
// before combos existed keep working as one failover set.
func normalizeCombo(combo string) string {
	combo = strings.TrimSpace(combo)
	if combo == "" {
		return "default"
	}
	return combo
}

// uniqueIDs deduplicates UUIDs (a payload may reference the same endpoint in
// several routes).
func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

type resetBreakerRequest struct {
	EndpointID string `json:"endpoint_id"`
}

// ResetCircuitBreaker handles POST /admin/ai-gateway/circuit-breaker/reset.
func (h *Handler) ResetCircuitBreaker(c *gin.Context) {
	if h.svc.aigw == nil {
		response.Error(c, 503, "AI gateway tidak aktif", "GATEWAY_UNAVAILABLE")
		return
	}
	var req resetBreakerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	if _, err := uuid.Parse(req.EndpointID); err != nil {
		response.BadRequest(c, "endpoint_id tidak valid", "validation")
		return
	}
	h.svc.aigw.Breaker().Reset(req.EndpointID)
	h.audit(c, "reset", "ai_route", req.EndpointID, nil, map[string]any{"circuit_breaker": "closed"})
	response.OK(c, "Circuit breaker direset", gin.H{"endpoint_id": req.EndpointID, "state": "closed"})
}

// ─── kill switch wiring ───────────────────────────────────────

// WithKillSwitch installs the emergency-stop store.
func WithKillSwitch(ks *KillSwitchStore) Option { return func(s *Service) { s.killSwitch = ks } }

// GetKillSwitch handles GET /admin/ai-gateway/kill-switch.
func (h *Handler) GetKillSwitch(c *gin.Context) {
	if h.svc.killSwitch == nil {
		out := map[string]bool{}
		for t := range killSwitchTargets() {
			out[t] = false
		}
		response.OK(c, "Kill switch tidak aktif", out)
		return
	}
	response.OK(c, "Status kill switch", h.svc.killSwitch.Matrix())
}

type killSwitchRequest struct {
	Target  string `json:"target"` // global|chat|vision|voice|embedding|reasoning
	Enabled bool   `json:"enabled"`
}

// SetKillSwitch handles POST /admin/ai-gateway/kill-switch.
func (h *Handler) SetKillSwitch(c *gin.Context) {
	if h.svc.killSwitch == nil {
		response.Error(c, 503, "Kill switch tidak tersedia", "KILL_SWITCH_UNAVAILABLE")
		return
	}
	var req killSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Payload tidak valid", "invalid_payload")
		return
	}
	if !validKillTarget(req.Target) {
		response.BadRequest(c, "target tidak valid", "validation")
		return
	}
	ctx := c.Request.Context()
	if err := h.svc.killSwitch.Set(ctx, req.Target, req.Enabled); err != nil {
		response.InternalError(c, "Gagal mengubah kill switch")
		return
	}
	h.audit(c, "update", "settings", KillSwitchKey, nil,
		map[string]any{"target": req.Target, "enabled": req.Enabled})
	response.OK(c, "Kill switch diperbarui", h.svc.killSwitch.Matrix())
}

// ─── helpers ──────────────────────────────────────────────────

func (h *Handler) invalidateRoutes() {
	if h.svc.aigw != nil {
		h.svc.aigw.InvalidateRoutes()
	}
}

func (h *Handler) encryptKey(plain *string) (*string, error) {
	if plain == nil || strings.TrimSpace(*plain) == "" {
		return nil, nil
	}
	key := h.svc.encryptionKey()
	if len(key) == 0 {
		return nil, nil
	}
	enc, err := crypto.Encrypt([]byte(*plain), key)
	if err != nil {
		return nil, err
	}
	return &enc, nil
}

func validEndpointStatus(s string) bool {
	switch s {
	case "active", "degraded", "inactive", "suspended":
		return true
	}
	return false
}

func validKillTarget(t string) bool {
	return killSwitchTargets()[t]
}

func nullableStrSlice(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

func defaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
