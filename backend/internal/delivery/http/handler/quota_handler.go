package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/quota"
	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/response"
)

// QuotaCycleRepo abstracts database operations for plan cycles and user quotas.
type QuotaCycleRepo interface {
	GetPlanCycles(ctx context.Context, planID uuid.UUID) ([]quota.PlanLimitCycle, error)
	UpsertPlanCycles(ctx context.Context, planID uuid.UUID, cycles []quota.PlanLimitCycle) error
	GetWarningThreshold(ctx context.Context, planID uuid.UUID) (int, error)
	GetOrCreateCounter(ctx context.Context, userID uuid.UUID, limitKey, cycleType string, start, end time.Time) (*quota.UsageCounter, error)
	GetCurrentCounters(ctx context.Context, userID uuid.UUID, now time.Time) ([]quota.UsageCounter, error)
	GetSubscriptionWeeklyAnchor(ctx context.Context, userID uuid.UUID) (*time.Time, error)
	GetUserPlanID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// QuotaHandler handles admin plan cycles and user quota snapshot endpoints.
type QuotaHandler struct {
	repo     QuotaCycleRepo
	enforcer *quota.DualCycleEnforcer
	realtime AdminRealtimeEmitter
}

// NewQuotaHandler builds the handler.
func NewQuotaHandler(repo QuotaCycleRepo, enforcer *quota.DualCycleEnforcer, realtime AdminRealtimeEmitter) *QuotaHandler {
	return &QuotaHandler{
		repo:     repo,
		enforcer: enforcer,
		realtime: realtime,
	}
}

// GetPlanCycles returns all cycle configurations for a plan.
// GET /api/v1/admin/plans/:id/cycles
func (h *QuotaHandler) GetPlanCycles(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid plan ID", err.Error())
		return
	}

	cycles, err := h.repo.GetPlanCycles(c.Request.Context(), planID)
	if err != nil {
		response.InternalError(c, "Failed to get plan cycles")
		return
	}
	if cycles == nil {
		cycles = []quota.PlanLimitCycle{}
	}

	response.OK(c, "Plan cycles retrieved successfully", cycles)
}

type upsertCyclesRequest struct {
	Cycles []quota.PlanLimitCycle `json:"cycles" binding:"required"`
}

// UpsertPlanCycles saves or updates plan cycles.
// PUT /api/v1/admin/plans/:id/cycles
func (h *QuotaHandler) UpsertPlanCycles(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid plan ID", err.Error())
		return
	}

	var req upsertCyclesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate items
	var changedKeys []string
	for _, item := range req.Cycles {
		if item.LimitKey == "" {
			response.BadRequest(c, "limit_key is required", "validation_error")
			return
		}
		if item.CycleType != quota.CycleMonthlyOnly && item.CycleType != quota.CycleWeeklyMonthly {
			response.BadRequest(c, "cycle_type must be monthly_only or weekly_monthly", "validation_error")
			return
		}
		if item.CycleType == quota.CycleWeeklyMonthly {
			if item.WeeklyLimit == nil || *item.WeeklyLimit <= 0 {
				response.BadRequest(c, "weekly_limit must be > 0 for weekly_monthly cycle", "validation_error")
				return
			}
			if item.DegradedMode == nil || *item.DegradedMode == "" {
				response.BadRequest(c, "degraded_mode is required for weekly_monthly cycle", "validation_error")
				return
			}
			if *item.DegradedMode == quota.DegradedThrottle {
				if item.ThrottleRateAmount == nil || *item.ThrottleRateAmount <= 0 ||
					item.ThrottleRateWindowSecs == nil || *item.ThrottleRateWindowSecs <= 0 {
					response.BadRequest(c, "throttle rate and window must be > 0", "validation_error")
					return
				}
			}
			if *item.DegradedMode == quota.DegradedModelDowngrade {
				if item.FallbackModel == nil || *item.FallbackModel == "" {
					response.BadRequest(c, "fallback_model is required for model_downgrade mode", "validation_error")
					return
				}
			}
		}
		changedKeys = append(changedKeys, item.LimitKey)
	}

	if err := h.repo.UpsertPlanCycles(c.Request.Context(), planID, req.Cycles); err != nil {
		response.InternalError(c, "Failed to save plan cycles")
		return
	}

	// Emit cycle_config_changed event
	if h.realtime != nil {
		h.realtime.PublishAdmins(c.Request.Context(), contracts.NewEvent(
			contracts.EventCycleConfigChanged,
			time.Now().Unix(),
			contracts.CycleConfigChangedPayload{
				PlanID:      planID.String(),
				ChangedKeys: changedKeys,
			},
		))
	}

	response.OK(c, "Plan cycles updated successfully", nil)
}

// UserQuotaSummary is the response payload for GET /api/v1/me/quota.
type UserQuotaSummary struct {
	WarningThreshold int                           `json:"warning_threshold"`
	WeeklyAnchor     *string                       `json:"weekly_anchor,omitempty"`
	Counters         []quota.UsageCounter          `json:"counters"`
	Cycles           map[string]quota.PlanLimitCycle `json:"cycles"`
}

// GetUserQuotaSnapshot returns current counters and active cycle settings for the authenticated user.
// GET /api/v1/me/quota
func (h *QuotaHandler) GetUserQuotaSnapshot(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		response.Unauthorized(c, "Missing authentication")
		return
	}
	userID := userIDVal.(uuid.UUID)

	now := time.Now().UTC()

	anchor, _ := h.repo.GetSubscriptionWeeklyAnchor(c.Request.Context(), userID)
	var anchorStr *string
	anchorTime := now
	if anchor != nil && !anchor.IsZero() {
		s := anchor.Format(time.RFC3339)
		anchorStr = &s
		anchorTime = *anchor
	}

	thresh, _ := h.repo.GetWarningThreshold(c.Request.Context(), uuid.Nil)

	cyclesMap := make(map[string]quota.PlanLimitCycle)
	if planID, err := h.repo.GetUserPlanID(c.Request.Context(), userID); err == nil && planID != uuid.Nil {
		if list, err := h.repo.GetPlanCycles(c.Request.Context(), planID); err == nil {
			for _, item := range list {
				cyclesMap[item.LimitKey] = item
				if item.CycleType == quota.CycleWeeklyMonthly {
					wStart, wEnd := quota.CalculateWeeklyWindow(anchorTime, now)
					_, _ = h.repo.GetOrCreateCounter(c.Request.Context(), userID, item.LimitKey, quota.CycleTypeWeekly, wStart, wEnd)
				}
			}
		}
	}

	counters, err := h.repo.GetCurrentCounters(c.Request.Context(), userID, now)
	if err != nil {
		response.InternalError(c, "Failed to retrieve quota counters")
		return
	}
	if counters == nil {
		counters = []quota.UsageCounter{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Quota snapshot retrieved successfully",
		"data": UserQuotaSummary{
			WarningThreshold: thresh,
			WeeklyAnchor:     anchorStr,
			Counters:         counters,
			Cycles:           cyclesMap,
		},
	})
}
