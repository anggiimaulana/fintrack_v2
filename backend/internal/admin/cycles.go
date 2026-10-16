package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/quota"
	"github.com/fintrack/backend/pkg/contracts"
)

// GetPlanCycles handles GET /api/v1/admin/plans/:id/cycles.
func (h *Handler) GetPlanCycles(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_plan_id", "message": "ID paket tidak valid"})
		return
	}

	rows, err := h.svc.pool.Query(c.Request.Context(), `
		SELECT id, plan_id, limit_key, cycle_type, weekly_limit, degraded_mode,
		       throttle_rate_amount, throttle_rate_window_secs, fallback_model,
		       max_degraded_usage, created_at, updated_at
		FROM plan_limit_cycles
		WHERE plan_id = $1
		ORDER BY limit_key ASC
	`, planID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "db_error", "message": "Gagal memuat konfigurasi siklus"})
		return
	}
	defer rows.Close()

	var list []quota.PlanLimitCycle
	for rows.Next() {
		var r quota.PlanLimitCycle
		if err := rows.Scan(
			&r.ID, &r.PlanID, &r.LimitKey, &r.CycleType, &r.WeeklyLimit,
			&r.DegradedMode, &r.ThrottleRateAmount, &r.ThrottleRateWindowSecs,
			&r.FallbackModel, &r.MaxDegradedUsage, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "scan_error", "message": "Gagal membaca konfigurasi siklus"})
			return
		}
		list = append(list, r)
	}
	if list == nil {
		list = []quota.PlanLimitCycle{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Konfigurasi siklus paket berhasil dimuat",
		"data":    list,
	})
}

type upsertPlanCyclesReq struct {
	Cycles []quota.PlanLimitCycle `json:"cycles" binding:"required"`
}

// PutPlanCycles handles PUT /api/v1/admin/plans/:id/cycles.
func (h *Handler) PutPlanCycles(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_plan_id", "message": "ID paket tidak valid"})
		return
	}

	var req upsertPlanCyclesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_body", "message": err.Error()})
		return
	}

	// Validate items
	var changedKeys []string
	for _, item := range req.Cycles {
		if item.LimitKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "validation_error", "message": "limit_key wajib diisi"})
			return
		}
		if item.CycleType == quota.CycleWeeklyMonthly {
			if item.WeeklyLimit == nil || *item.WeeklyLimit <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "validation_error", "message": "weekly_limit harus > 0 untuk siklus mingguan"})
				return
			}
			if item.DegradedMode == nil || *item.DegradedMode == "" {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "validation_error", "message": "degraded_mode wajib dipilih"})
				return
			}
			if *item.DegradedMode == quota.DegradedThrottle {
				if item.ThrottleRateAmount == nil || *item.ThrottleRateAmount <= 0 ||
					item.ThrottleRateWindowSecs == nil || *item.ThrottleRateWindowSecs <= 0 {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "validation_error", "message": "parameter throttle harus > 0"})
					return
				}
			}
			if *item.DegradedMode == quota.DegradedModelDowngrade {
				if item.FallbackModel == nil || *item.FallbackModel == "" {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "validation_error", "message": "fallback_model wajib diisi untuk mode penurunan model"})
					return
				}
			}
		}
		changedKeys = append(changedKeys, item.LimitKey)
	}

	ctx := c.Request.Context()
	tx, err := h.svc.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "tx_error", "message": "Gagal memulai transaksi"})
		return
	}
	defer tx.Rollback(ctx)

	for _, item := range req.Cycles {
		_, err := tx.Exec(ctx, `
			INSERT INTO plan_limit_cycles (
				plan_id, limit_key, cycle_type, weekly_limit, degraded_mode,
				throttle_rate_amount, throttle_rate_window_secs, fallback_model, max_degraded_usage
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (plan_id, limit_key) DO UPDATE SET
				cycle_type = EXCLUDED.cycle_type,
				weekly_limit = EXCLUDED.weekly_limit,
				degraded_mode = EXCLUDED.degraded_mode,
				throttle_rate_amount = EXCLUDED.throttle_rate_amount,
				throttle_rate_window_secs = EXCLUDED.throttle_rate_window_secs,
				fallback_model = EXCLUDED.fallback_model,
				max_degraded_usage = EXCLUDED.max_degraded_usage,
				updated_at = now()
		`, planID, item.LimitKey, item.CycleType, item.WeeklyLimit, item.DegradedMode,
			item.ThrottleRateAmount, item.ThrottleRateWindowSecs, item.FallbackModel, item.MaxDegradedUsage)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "insert_error", "message": "Gagal menyimpan konfigurasi siklus"})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "commit_error", "message": "Gagal menyelesaikan transaksi"})
		return
	}

	h.audit(c, "update", "plan_cycles", planID.String(), nil, map[string]any{"changed_keys": changedKeys})
	// Invalidate entitlement cache for plan
	if h.svc.entInvalidator != nil {
		h.svc.entInvalidator.InvalidatePlan(ctx, planID.String())
	}

	// Emit cycle_config_changed
	if h.svc.emitter != nil {
		h.svc.emitter.PublishAdmins(ctx, contracts.NewEvent(
			contracts.EventCycleConfigChanged,
			time.Now().Unix(),
			contracts.CycleConfigChangedPayload{
				PlanID:      planID.String(),
				ChangedKeys: changedKeys,
			},
		))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Konfigurasi siklus paket berhasil diperbarui",
	})
}
