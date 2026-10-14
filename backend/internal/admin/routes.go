package admin

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the Super Admin (P06) HTTP surface on an already
// authenticated + super-admin-gated router group (`/api/v1/admin`).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	// Dashboard metrics
	metrics := rg.Group("/metrics")
	{
		metrics.GET("/overview", h.Overview)
		metrics.GET("/timeseries", h.Timeseries)
		metrics.GET("/payment-methods", h.PaymentMethods)
	}

	// Plans & entitlements (entitlements/limits travel inside the plan payload)
	plans := rg.Group("/plans")
	{
		plans.GET("", h.ListPlans)
		plans.POST("", h.CreatePlan)
		plans.PUT("/:id", h.UpdatePlan)
		plans.DELETE("/:id", h.DeactivatePlan)
	}

	// Subscriptions
	subs := rg.Group("/subscriptions")
	{
		subs.GET("", h.ListSubscriptions)
		subs.POST("/:id/extend", h.ExtendSubscription)
		subs.POST("/:id/force-downgrade", h.ForceDowngrade)
		subs.POST("/:id/cancel", h.CancelSubscription)
	}

	// Payments: history, manual review queue, refunds
	pay := rg.Group("/payments")
	{
		pay.GET("", h.ListPayments)
		pay.GET("/review", h.ListPendingPayments)
		pay.POST("/:id/approve", h.ApprovePayment)
		pay.POST("/:id/reject", h.RejectPayment)
		pay.POST("/:id/refund", h.RefundPayment)
	}

	// Token balances & append-only ledger
	tokens := rg.Group("/tokens")
	{
		tokens.GET("/users", h.ListTokenBalances)
		tokens.GET("/ledger", h.ListTokenLedger)
		tokens.POST("/adjust", h.AdjustTokens)
	}

	// AI gateway operations
	ai := rg.Group("/ai-gateway")
	{
		ai.GET("/endpoints", h.ListGatewayEndpoints)
		ai.POST("/endpoints", h.CreateGatewayEndpoint)
		ai.PUT("/endpoints/:id", h.UpdateGatewayEndpoint)
		ai.DELETE("/endpoints/:id", h.DeleteGatewayEndpoint)

		ai.GET("/routes", h.ListRoutes)
		ai.PUT("/routes", h.PutRoutes)

		// Probe the actual model on every active route (catches a "reachable but
		// dead model" config that reachability probing misses).
		ai.GET("/verify-routes", h.VerifyRoutes)

		ai.POST("/circuit-breaker/reset", h.ResetCircuitBreaker)

		ai.GET("/kill-switch", h.GetKillSwitch)
		ai.POST("/kill-switch", h.SetKillSwitch)
	}

	// Pricing rules (F5): dynamic token pricing + markup, fully audited.
	pricing := rg.Group("/pricing-rules")
	{
		pricing.GET("", h.ListPricingRules)
		pricing.GET("/active", h.GetActivePricingRule)
		pricing.POST("", h.CreatePricingRule)
		pricing.PUT("/:id", h.UpdatePricingRule)
		pricing.POST("/:id/activate", h.ActivatePricingRule)
		pricing.DELETE("/:id", h.DeletePricingRule)
	}

	// Entitlement key catalog (F5): powers the plan-editor dropdowns.
	rg.GET("/entitlement-catalog", h.EntitlementCatalog)

	// Quota monitoring (F6): users running low on a plan limit.
	rg.GET("/quota-monitor", h.QuotaMonitor)

	// Audit trail (append-only: read only)
	rg.GET("/audit-logs", h.ListAuditLogs)

	// Global settings & feature flags
	rg.GET("/settings", h.ListSettings)
	rg.PUT("/settings", h.UpdateSettings)
	// Effective runtime config + secret presence (no secret values).
	rg.GET("/settings/config", h.ConfigStatus)
	// Transactional email: read effective status + send a test message.
	rg.GET("/settings/mail", h.MailStatus)
	rg.POST("/settings/mail/test", h.SendTestEmail)
	rg.GET("/settings/storage", h.StorageStatus)
	rg.GET("/feature-flags", h.ListFeatureFlags)
	rg.PUT("/feature-flags", h.UpdateFeatureFlag)
}
