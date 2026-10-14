package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/pkg/response"
)

// quotaMonitorRow is one user's consumption against a numeric plan limit.
type quotaMonitorRow struct {
	UserID      string  `json:"user_id"`
	Email       string  `json:"email"`
	FullName    string  `json:"full_name"`
	PlanCode    string  `json:"plan_code"`
	PlanName    string  `json:"plan_name"`
	LimitKey    string  `json:"limit_key"`
	Used        int64   `json:"used"`
	Limit       int64   `json:"limit"`
	UsedPercent float64 `json:"used_percent"`
	Remaining   int64   `json:"remaining"`
}

// QuotaMonitor handles GET /admin/quota-monitor?threshold=80.
//
// It lists users whose consumption of a numeric limit has crossed `threshold`
// percent (default 80), so operators can spot accounts about to hit a wall.
// Limits are read from each user's plan; usage comes from the same counters the
// enforcement middleware uses.
func (h *Handler) QuotaMonitor(c *gin.Context) {
	threshold := 80.0
	if v := c.Query("threshold"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 100 {
			threshold = f
		}
	}

	ctx := c.Request.Context()
	rows, err := h.svc.pool.Query(ctx, `
		WITH plan_of_user AS (
			SELECT u.id AS user_id, u.email, u.full_name,
			       COALESCE(p.code, 'free')  AS plan_code,
			       COALESCE(p.name, 'Free')  AS plan_name,
			       COALESCE(p.id, (
			           SELECT id FROM plans WHERE code = 'free' LIMIT 1
			       )) AS plan_id
			FROM users u
			LEFT JOIN subscriptions s ON s.user_id = u.id AND s.status IN ('active','trialing','grace','past_due')
			LEFT JOIN plans p ON p.id = s.plan_id
			WHERE u.is_active = true
		)
		SELECT pu.user_id, pu.email, pu.full_name, pu.plan_code, pu.plan_name,
		       pl.limit_key, pl.limit_value,
		       COALESCE(tp.used_tokens, 0) + COALESCE(tp.reserved_tokens, 0) AS used_tokens,
		       COALESCE((
		           SELECT count(*) FROM token_ledger l
		           WHERE l.user_id = pu.user_id AND l.entry_type = 'reserve'
		             AND l.created_at >= COALESCE(tp.period_start, now() - interval '30 days')
		       ), 0) AS used_requests
		FROM plan_of_user pu
		JOIN plan_limits pl ON pl.plan_id = pu.plan_id
		LEFT JOIN token_periods tp ON tp.user_id = pu.user_id AND tp.period_end >= now()
		WHERE pl.limit_key IN ('ai_tokens_per_cycle','ai_requests_per_cycle')
		  AND pl.limit_value > 0
		ORDER BY pu.user_id`)
	if err != nil {
		response.InternalError(c, "Gagal memuat pemantauan kuota")
		return
	}
	defer rows.Close()

	out := []quotaMonitorRow{}
	for rows.Next() {
		var r quotaMonitorRow
		var usedTokens, usedRequests int64
		if err := rows.Scan(&r.UserID, &r.Email, &r.FullName, &r.PlanCode, &r.PlanName,
			&r.LimitKey, &r.Limit, &usedTokens, &usedRequests); err != nil {
			response.InternalError(c, "Gagal memuat pemantauan kuota")
			return
		}
		if r.LimitKey == "ai_tokens_per_cycle" {
			r.Used = usedTokens
		} else {
			r.Used = usedRequests
		}
		if r.Limit <= 0 {
			continue
		}
		r.UsedPercent = float64(r.Used) / float64(r.Limit) * 100
		if r.UsedPercent < threshold {
			continue
		}
		r.Remaining = r.Limit - r.Used
		if r.Remaining < 0 {
			r.Remaining = 0
		}
		out = append(out, r)
	}
	response.OK(c, "Pemantauan kuota", out)
}
