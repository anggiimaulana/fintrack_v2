package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/pkg/response"
)

// Handler exposes the Super Admin HTTP surface. Endpoints are grouped by
// domain across the files in this package but share one Handler value.
type Handler struct {
	svc *Service
}

// NewHandler builds the admin handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// overviewResponse is the payload for GET /admin/metrics/overview.
//
// Growth percentages compare the current window against the immediately
// preceding window of the same length; a zero baseline yields 0.
type overviewResponse struct {
	MRRIDR                  int64   `json:"mrr_idr"`
	MRRGrowthPercentage     float64 `json:"mrr_growth_percentage"`
	ActiveUsers30d          int64   `json:"active_users_30d"`
	ActiveUsersGrowthPct    float64 `json:"active_users_growth_percentage"`
	TokensConsumed24h       int64   `json:"tokens_consumed_24h"`
	TokensGrowthPercentage  float64 `json:"tokens_growth_percentage"`
	PaymentSuccessRate      float64 `json:"payment_success_rate"`
	PendingManualPayments   int64   `json:"pending_manual_payments"`
	AIEndpointsHealthy      int64   `json:"ai_endpoints_healthy"`
	AIEndpointsTotal        int64   `json:"ai_endpoints_total"`
	CircuitBreakerOpenCount int64   `json:"circuit_breaker_open_count"`
	CachedAt                string  `json:"cached_at"`
}

// mrrQuery normalises every active subscription to a monthly figure:
// monthly plans contribute their monthly price, yearly plans a twelfth of the
// yearly price (integer floor, K-27).
const mrrQuery = `
	SELECT COALESCE(SUM(
	         CASE WHEN COALESCE(s.metadata->>'billing_cycle', 'monthly') = 'yearly'
	              THEN p.price_yearly_idr / 12
	              ELSE p.price_monthly_idr
	         END), 0)::bigint
	FROM subscriptions s
	JOIN plans p ON p.id = s.plan_id
	WHERE s.status = 'active'`

// activeUsersQuery counts distinct users with billable activity (a recorded
// transaction or a session use) inside the window [$1, $2) counted backwards
// from now.
const activeUsersQuery = `
	SELECT COUNT(DISTINCT u)::bigint FROM (
		SELECT user_id AS u FROM transactions
		 WHERE created_at >= now() - ($1 || ' days')::interval
		   AND created_at <  now() - ($2 || ' days')::interval
		UNION
		SELECT user_id AS u FROM refresh_tokens
		 WHERE last_used_at >= now() - ($1 || ' days')::interval
		   AND last_used_at <  now() - ($2 || ' days')::interval
	) activity`

// Overview handles GET /api/v1/admin/metrics/overview.
//
// The aggregate is cached for 60s; pass ?refresh=true to bypass the cache
// (the admin "Refresh" button).
func (h *Handler) Overview(c *gin.Context) {
	ctx := c.Request.Context()

	const cacheKey = "admin:metrics:overview"
	if h.svc.cache != nil && c.Query("refresh") != "true" {
		if b, ok := h.svc.cache.Get(ctx, cacheKey); ok {
			c.Data(http.StatusOK, "application/json; charset=utf-8", b)
			return
		}
	}

	var out overviewResponse
	var mrr, prevMRR, users, prevUsers, tokens24h, prevTokens24h, pending int64

	// MRR growth baseline: MRR carried by subscriptions that already existed a
	// month ago (the plan catalogue has no price history, so this is the
	// closest available proxy for the previous month's run rate).
	err := h.svc.pool.QueryRow(ctx, mrrQuery).Scan(&mrr)
	if err == nil {
		err = h.svc.pool.QueryRow(ctx, mrrQuery+` AND s.created_at < now() - interval '30 days'`).Scan(&prevMRR)
	}
	if err == nil {
		err = h.svc.pool.QueryRow(ctx, activeUsersQuery, "30", "0").Scan(&users)
	}
	if err == nil {
		err = h.svc.pool.QueryRow(ctx, activeUsersQuery, "60", "30").Scan(&prevUsers)
	}
	if err == nil {
		err = h.svc.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(GREATEST(raw_tokens, ABS(amount))), 0)::bigint FROM token_ledger
			 WHERE entry_type = 'commit' AND created_at >= now() - interval '24 hours'`).Scan(&tokens24h)
	}
	if err == nil {
		err = h.svc.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(GREATEST(raw_tokens, ABS(amount))), 0)::bigint FROM token_ledger
			 WHERE entry_type = 'commit'
			   AND created_at >= now() - interval '48 hours'
			   AND created_at <  now() - interval '24 hours'`).Scan(&prevTokens24h)
	}
	if err == nil {
		err = h.svc.pool.QueryRow(ctx,
			`SELECT COUNT(*)::bigint FROM payments WHERE status = 'pending_review'`).Scan(&pending)
	}
	if err != nil {
		response.InternalError(c, "Gagal menghitung metrik")
		return
	}

	out.MRRIDR = mrr
	out.MRRGrowthPercentage = percentDelta(mrr, prevMRR)
	out.ActiveUsers30d = users
	out.ActiveUsersGrowthPct = percentDelta(users, prevUsers)
	out.TokensConsumed24h = tokens24h
	out.TokensGrowthPercentage = percentDelta(tokens24h, prevTokens24h)
	out.PendingManualPayments = pending

	// Success rate over the last 30 days: settled vs every resolved attempt
	// (deliberately canceled checkouts are excluded).
	var settled, resolved int64
	_ = h.svc.pool.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'settled'), COUNT(*)::bigint
		FROM payments
		WHERE created_at >= now() - interval '30 days'
		  AND status IN ('settled','failed','expired')`).Scan(&settled, &resolved)
	if resolved > 0 {
		out.PaymentSuccessRate = roundTo(float64(settled)/float64(resolved)*100, 1)
	}

	// Endpoint pool health + live circuit-breaker state.
	rows, err := h.svc.pool.Query(ctx, `SELECT id, health FROM ai_endpoints`)
	if err == nil {
		defer rows.Close()
		var breakerOpen int64
		for rows.Next() {
			var id, health string
			if rows.Scan(&id, &health) != nil {
				break
			}
			out.AIEndpointsTotal++
			if health == "healthy" {
				out.AIEndpointsHealthy++
			}
			if h.svc.aigw != nil && h.svc.aigw.Breaker().State(id) == "open" {
				breakerOpen++
			}
		}
		out.CircuitBreakerOpenCount = breakerOpen
	}

	out.CachedAt = h.svc.now().Format(time.RFC3339)
	if h.svc.cache != nil {
		if b, err := json.Marshal(response.Envelope("Metrik berhasil diambil", out)); err == nil {
			h.svc.cache.Set(ctx, cacheKey, b, 60*time.Second)
		}
	}
	response.OK(c, "Metrik berhasil diambil", out)
}

// timeseriesPoint is one day of the dashboard time-series.
type timeseriesPoint struct {
	Date            string `json:"date"`
	RevenueIDR      int64  `json:"revenue_idr"`
	PaymentsSettled int64  `json:"payments_settled"`
	MRRIDR          int64  `json:"mrr_idr"`
	TokensChat      int64  `json:"tokens_chat"`
	TokensVision    int64  `json:"tokens_vision"`
	TokensVoice     int64  `json:"tokens_voice"`
	TokensEmbedding int64  `json:"tokens_embedding"`
}

// Timeseries handles GET /api/v1/admin/metrics/timeseries?period=7d|30d|90d.
// Returns one flat row per day (dense: days without traffic are zero-filled)
// for the revenue area chart and the stacked token-consumption bar chart.
func (h *Handler) Timeseries(c *gin.Context) {
	ctx := c.Request.Context()
	days := 7
	switch c.Query("period") {
	case "30d":
		days = 30
	case "90d":
		days = 90
	case "7d", "":
		days = 7
	}

	idx := map[string]int{}
	out := make([]timeseriesPoint, 0, days)
	rows, err := h.svc.pool.Query(ctx, `
		SELECT to_char(d.day, 'YYYY-MM-DD'),
		       COALESCE(SUM(p.amount_idr), 0)::bigint,
		       COUNT(p.id)::bigint
		FROM generate_series(now()::date - ($1::int - 1), now()::date, '1 day') AS d(day)
		LEFT JOIN payments p
		  ON p.status = 'settled' AND p.created_at::date = d.day
		GROUP BY d.day ORDER BY d.day`, days)
	if err != nil {
		response.InternalError(c, "Gagal memuat timeseries")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var p timeseriesPoint
		if rows.Scan(&p.Date, &p.RevenueIDR, &p.PaymentsSettled) != nil {
			continue
		}
		idx[p.Date] = len(out)
		out = append(out, p)
	}

	trows, err := h.svc.pool.Query(ctx, `
		SELECT to_char(d.day, 'YYYY-MM-DD'), COALESCE(l.modality, ''), COALESCE(SUM(GREATEST(l.raw_tokens, ABS(l.amount))), 0)::bigint
		FROM generate_series(now()::date - ($1::int - 1), now()::date, '1 day') AS d(day)
		LEFT JOIN token_ledger l
		  ON l.created_at::date = d.day AND l.entry_type = 'commit'
		GROUP BY d.day, l.modality ORDER BY d.day`, days)
	if err == nil {
		defer trows.Close()
		for trows.Next() {
			var date, modality string
			var amount int64
			if trows.Scan(&date, &modality, &amount) != nil {
				continue
			}
			i, ok := idx[date]
			if !ok {
				continue
			}
			switch modality {
			case "chat":
				out[i].TokensChat += amount
			case "vision":
				out[i].TokensVision += amount
			case "voice":
				out[i].TokensVoice += amount
			case "embedding":
				out[i].TokensEmbedding += amount
			}
		}
	}

	// MRR projection: the current run rate, plotted as a flat reference line.
	var mrr int64
	_ = h.svc.pool.QueryRow(ctx, mrrQuery).Scan(&mrr)
	for i := range out {
		out[i].MRRIDR = mrr
	}

	response.OK(c, "Timeseries berhasil diambil", out)
}

// gatewayShare is one slice of the payment-method donut chart.
type gatewayShare struct {
	Gateway      string `json:"gateway"`
	SettledCount int64  `json:"settled_count"`
	TotalIDR     int64  `json:"total_idr"`
}

// PaymentMethods handles GET /api/v1/admin/metrics/payment-methods: settled
// payment share per gateway over the last 30 days (donut chart).
func (h *Handler) PaymentMethods(c *gin.Context) {
	rows, err := h.svc.pool.Query(c.Request.Context(), `
		SELECT gateway, COUNT(*)::bigint, COALESCE(SUM(amount_idr), 0)::bigint
		FROM payments
		WHERE status = 'settled' AND created_at >= now() - interval '30 days'
		GROUP BY gateway ORDER BY COUNT(*) DESC`)
	if err != nil {
		response.InternalError(c, "Gagal memuat metode pembayaran")
		return
	}
	defer rows.Close()

	out := []gatewayShare{}
	for rows.Next() {
		var g gatewayShare
		if rows.Scan(&g.Gateway, &g.SettledCount, &g.TotalIDR) == nil {
			out = append(out, g)
		}
	}
	response.OK(c, "Rincian metode pembayaran berhasil dimuat", out)
}

// percentDelta returns the growth of cur vs prev as a percentage (0 when prev == 0).
func percentDelta(cur, prev int64) float64 {
	if prev == 0 {
		return 0
	}
	return roundTo(float64(cur-prev)/float64(prev)*100, 1)
}

// roundTo rounds v to n decimal places.
func roundTo(v float64, n int) float64 {
	f := 1.0
	for i := 0; i < n; i++ {
		f *= 10
	}
	return float64(int64(v*f+sign(v)*0.5)) / f
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}
