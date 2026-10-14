package http

import (
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ulule/limiter/v3"

	"github.com/fintrack/backend/internal/admin"
	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/infrastructure/metrics"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/internal/realtime"
	"github.com/fintrack/backend/internal/storage"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/internal/user"
	"github.com/fintrack/backend/pkg/logger"
	"github.com/fintrack/backend/pkg/response"
	"go.uber.org/zap"
)

type RouterConfig struct {
	JWTSecret    string
	CORSOrigin   string
	CookieSecure bool
	// Metrics, when non-nil, enables the /metrics endpoint and request
	// instrumentation.
	Metrics *metrics.Registry
	// Entitlement, when non-nil, enables plan enforcement (feature + limit
	// gates) on the user-facing routes. Nil disables gating entirely, which is
	// what tests that predate P07 rely on.
	Entitlement *middleware.EntitlementMiddleware
	// Usage supplies the numeric usage for limit checks. Required when
	// Entitlement is set.
	Usage *middleware.UsageProviders
	// Pool and Redis back the readiness probe (/health/ready). Either may be nil
	// in tests; the probe then reports the dependency as down/degraded.
	Pool  *pgxpool.Pool
	Redis *infraRedis.Client
	// Storage, when non-nil, enables avatar upload/serving on the auth handler.
	Storage *storage.Provider
	// Ledger, when non-nil, enables two-phase token reservations on AI routes.
	Ledger *billing.LedgerService
	// Estimates, when non-nil, supplies per-modality reserve amounts (P3).
	Estimates *billing.EstimateStore
	// Notifications, when non-nil, exposes the user's notification inbox and
	// preferences. Nil omits those routes (tests that predate them).
	Notifications *handler.NotificationHandler
	// Audit records administrative mutations (payment accounts, etc.). Nil
	// disables auditing, which tests that do not exercise the trail rely on.
	Audit handler.AuditWriter
}

// envInt reads an integer env var, falling back to def when unset/invalid.
func envInt(key string, def int64) int64 {
	if v, err := strconv.ParseInt(os.Getenv(key), 10, 64); err == nil && v > 0 {
		return v
	}
	return def
}

// featureGate returns the handlers enforcing a plan feature, or nil when
// entitlement enforcement is not configured (e.g. in tests that predate P07).
func featureGate(cfg RouterConfig, featureKey string) []gin.HandlerFunc {
	if cfg.Entitlement == nil {
		return nil
	}
	return []gin.HandlerFunc{cfg.Entitlement.RequireFeature(featureKey)}
}

// limitGate returns the handlers enforcing a numeric plan limit.
func limitGate(cfg RouterConfig, limitKey string) []gin.HandlerFunc {
	if cfg.Entitlement == nil {
		return nil
	}
	var usage middleware.UsageResolver
	if cfg.Usage != nil {
		usage = cfg.Usage.UsageFor(limitKey)
	}
	return []gin.HandlerFunc{cfg.Entitlement.WithinLimit(limitKey, usage)}
}

// gated returns the middleware chain (dropping nil groups) followed by the
// final route handler. Go only permits `...` on the last argument, so the
// handler is appended here rather than spread at each call site.
func gated(handler gin.HandlerFunc, groups ...[]gin.HandlerFunc) []gin.HandlerFunc {
	out := make([]gin.HandlerFunc, 0, len(groups)+1)
	for _, g := range groups {
		out = append(out, g...)
	}
	return append(out, handler)
}

func SetupRouter(
	config RouterConfig,
	transactionUC *usecase.TransactionUsecase,
	categoryUC *usecase.CategoryUsecase,
	userUC *usecase.UserUsecase,
	dashboardUC *usecase.DashboardUsecase,
	spkUC *usecase.SPKUsecase,
	spkMetricUC *usecase.SPKMetricUsecase,
	budgetUC *usecase.BudgetUsecase,
	paymentAccountUC *usecase.PaymentAccountUsecase,
	aiUC *usecase.AIUsecase,
	providerUC *usecase.ProviderUsecase,
	reportUC *usecase.ReportUsecase,
	authUC *usecase.AuthUsecase,
	entitlementH *handler.EntitlementHandler,
	realtimeH *realtime.Handler,
	paymentH *handler.PaymentHandler,
	superAdminH *admin.Handler,
	roleResolver middleware.RoleResolver,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("server panic recovered", zap.Any("panic", recovered))
		response.InternalError(c, "Terjadi kesalahan internal pada server. Silakan coba beberapa saat lagi.")
		c.Abort()
	}))
	if config.Metrics != nil {
		r.Use(config.Metrics.Middleware())
	}
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(config.CORSOrigin))
	r.Use(middleware.RequestLogger())

	middleware.CacheCleanup()

	th := handler.NewTransactionHandler(transactionUC)
	ch := handler.NewCategoryHandler(categoryUC)
	dh := handler.NewDashboardHandler(dashboardUC)

	sh := handler.NewSPKHandler(spkUC)
	smh := handler.NewSPKMetricHandler(spkMetricUC)
	bh := handler.NewBudgetHandler(budgetUC)
	pah := handler.NewPaymentAccountHandler(paymentAccountUC).WithAudit(config.Audit)
	aiH := handler.NewAIHandler(aiUC, transactionUC)
	if config.Ledger != nil {
		aiH.WithLedger(config.Ledger)
	}
	if config.Estimates != nil {
		aiH.WithEstimates(config.Estimates)
	}
	ah := handler.NewAuthHandler(authUC, config.CookieSecure)
	if config.Pool != nil {
		ah.WithAnonymize(user.NewAnonymizeService(config.Pool))
	}
	if config.Storage != nil {
		ah.WithStorage(config.Storage)
	}
	adminH := handler.NewAdminHandler(userUC)
	providerH := handler.NewProviderHandler(providerUC)
	rh := handler.NewReportHandler(reportUC)

	// Rate limits. Login/register limits are overridable via env so load
	// testing (k6) can exercise the auth flow without tripping the guard.
	loginLimit := envInt("RATE_LIMIT_LOGIN", 5)
	registerLimit := envInt("RATE_LIMIT_REGISTER", 3)

	rateLimitMap := map[string]limiter.Rate{
		"/api/v1/auth/login":           {Period: 1 * time.Minute, Limit: loginLimit},
		"/api/v1/auth/register":        {Period: 1 * time.Minute, Limit: registerLimit},
		"/api/v1/auth/refresh":         {Period: 1 * time.Minute, Limit: 10},
		"/api/v1/spk/calculate":        {Period: 1 * time.Minute, Limit: 30},
		// Billing mutations are expensive (they issue invoices and call the
		// gateway), so they get their own tight per-user budget to stop a user
		// (or a stuck retry loop) from hammering them.
		"/api/v1/payments/checkout":     {Period: 1 * time.Minute, Limit: 5},
		"/api/v1/invoices/:id/cancel":   {Period: 1 * time.Minute, Limit: 10},
		"/api/v1/payments/:id/proof":    {Period: 1 * time.Minute, Limit: 10},
	}

	rateLimitMw := middleware.RateLimitMiddlewareByPath(rateLimitMap)

	api := r.Group("/api/v1")
	api.Use(rateLimitMw)
	api.Use(middleware.OriginCheck(config.CORSOrigin))
	{
		// Payment webhook is public (no JWT); signature is verified by the gateway.
		if paymentH != nil {
			api.POST("/payments/webhook/midtrans", paymentH.MidtransWebhook)
		}

		auth := api.Group("/auth")
		{
			auth.POST("/login", ah.Login)
			auth.POST("/register", ah.Register)
			auth.POST("/logout", ah.Logout)
			auth.POST("/refresh", ah.Refresh)
		}

		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(config.JWTSecret))
		{
			protected.GET("/me", ah.GetProfile)
			protected.PUT("/me", ah.UpdateProfile)
			protected.PUT("/auth/password", ah.ChangePassword)
			protected.POST("/me/avatar", ah.UploadAvatar)
			protected.DELETE("/me", ah.DeleteAccount)
			protected.POST("/user/delete-account", ah.DeleteAccount)
			protected.GET("/user/export-data", ah.ExportData)
			// Notification inbox + preferences. Preferences gate email delivery;
			// the in-app inbox is always available.
			if config.Notifications != nil {
				nh := config.Notifications
				protected.GET("/me/notification-preferences", nh.GetPreferences)
				protected.PUT("/me/notification-preferences", nh.UpdatePreferences)
				protected.GET("/me/notifications", nh.List)
				// Static segments must be registered before the ":id" wildcard so
				// gin does not route "stats"/"read"/"delete" into :id.
				protected.GET("/me/notifications/stats", nh.Stats)
				protected.POST("/me/notifications/read", nh.MarkReadBulk)
				protected.POST("/me/notifications/read-all", nh.MarkAllRead)
				protected.POST("/me/notifications/delete", nh.DeleteBulk)
				protected.GET("/me/notifications/:id", nh.GetByID)
				protected.POST("/me/notifications/:id/read", nh.MarkRead)
				protected.DELETE("/me/notifications/:id", nh.DeleteOne)
				protected.DELETE("/me/notifications", nh.Clear)
				protected.POST("/me/notifications/test-email", nh.SendTestEmail)
			}
			// Realtime SSE stream. Registered outside the cached group: a
			// long-lived stream must never be wrapped in the 30s response cache.
			if realtimeH != nil {
				protected.GET("/realtime/stream", realtimeH.Stream)
			}
			if entitlementH != nil {
				protected.GET("/me/entitlements", entitlementH.Me)
			}

			userRoutes := protected.Group("")
			userRoutes.Use(middleware.RequireUser())
			{
				cached := userRoutes.Group("")
				cached.Use(middleware.CacheMiddleware(30 * time.Second))
				{
					cached.GET("/dashboard/overview", dh.GetOverview)
					cached.GET("/dashboard/charts", dh.GetChartData)
					cached.GET("/dashboard/charts/categories", dh.GetCategoryChart)
					cached.GET("/transactions/stats", th.GetStats)
					cached.GET("/transactions/:id", th.GetByID)

					cached.GET("/spk/latest", sh.GetLatest)
					cached.GET("/spk/history", sh.GetHistory)
					cached.GET("/spk/metrics", smh.GetMyModel)
					// advanced_budgeting: Pro/Business only.
					cached.GET("/budgets", gated(bh.List, featureGate(config, "advanced_budgeting"))...)
					cached.GET("/budgets/status", gated(bh.Status, featureGate(config, "advanced_budgeting"))...)
					// reports_export: Pro/Business only.
					cached.GET("/reports", gated(rh.Generate, featureGate(config, "reports_export"))...)
					cached.GET("/reports/analytics", gated(rh.Analytics, featureGate(config, "reports_export"))...)
					cached.GET("/categories/income", ch.ListIncome)
					cached.GET("/categories/expense", ch.ListExpense)
					cached.GET("/categories/expense/:categoryId/subcategories", ch.ListSubcategories)
					cached.GET("/categories/savings", ch.ListSavings)
				}

				userRoutes.GET("/transactions", th.List)
				// Report export reuses the `reports_export` gate: a user who may
				// not view reports may not download them either. Registered
				// outside the cached group because it streams a binary file (the
				// response cache only stores JSON).
				userRoutes.GET("/reports/export", gated(rh.Export, featureGate(config, "reports_export"))...)
				// max_transactions: Free 100, Pro 5000, Business unlimited.
				userRoutes.POST("/transactions", gated(th.Create, limitGate(config, "max_transactions"))...)
				userRoutes.PUT("/transactions/:id", th.Update)
				userRoutes.DELETE("/transactions/:id", th.Delete)
				if paymentH != nil {
					userRoutes.POST("/payments/checkout", paymentH.Checkout)
					// Self-service billing (P0/P1): catalog, history, invoices,
					// manual proof upload, and plan changes.
					userRoutes.GET("/plans", paymentH.ListPublicPlans)
					userRoutes.GET("/payments", paymentH.ListMyPayments)
					userRoutes.GET("/payments/:id", paymentH.GetMyPayment)
					userRoutes.GET("/payments/:id/instructions", paymentH.GetMyPaymentInstructions)
					userRoutes.POST("/payments/:id/proof", paymentH.UploadMyProof)
					userRoutes.GET("/payment-proofs/:id", paymentH.ServeProof)
					userRoutes.GET("/invoices", paymentH.ListMyInvoices)
					// A user may cancel their own unpaid invoice to free the
					// one-open-invoice slot for a new checkout.
					userRoutes.POST("/invoices/:id/cancel", paymentH.CancelMyInvoice)

					userRoutes.GET("/subscriptions/me", paymentH.MySubscription)
					userRoutes.POST("/subscriptions/me/downgrade", paymentH.ScheduleDowngrade)
					userRoutes.DELETE("/subscriptions/me/downgrade", paymentH.CancelDowngrade)
					userRoutes.POST("/subscriptions/me/cancel", paymentH.CancelSubscription)
				}
				// Manual payment destinations a user may pay to. Available to every
				// authenticated user (they need it to complete a manual payment).
				userRoutes.GET("/payment-accounts", pah.ListActive)
				ai := userRoutes.Group("/ai")
				{
					// ai_chat is in every plan; the token quota is the real gate.
					// ai_requests_per_cycle bounds the number of AI calls per cycle
					// across chat, OCR, and voice.
					ai.POST("/chat", gated(aiH.Chat,
						featureGate(config, "ai_chat"),
						limitGate(config, "ai_requests_per_cycle"),
						limitGate(config, "ai_tokens_per_cycle"))...)
					ai.POST("/sessions", aiH.CreateSession)
					ai.GET("/sessions", aiH.ListSessions)
					ai.PUT("/sessions/:id", aiH.UpdateSession)
					ai.DELETE("/sessions/:id", aiH.DeleteSession)
					ai.GET("/sessions/:id/messages", aiH.ListMessages)
					// OCR: Pro/Business only, bounded by ocr_scans_per_cycle AND the
					// shared token quota — a scan consumes vision tokens, so it must
					// respect ai_tokens_per_cycle exactly like chat does.
					ai.POST("/receipt", gated(aiH.AnalyzeReceipt,
						featureGate(config, "ocr_receipt"),
						limitGate(config, "ai_requests_per_cycle"),
						limitGate(config, "ocr_scans_per_cycle"),
						limitGate(config, "ai_tokens_per_cycle"))...)
					ai.POST("/receipt/transaction", gated(aiH.CreateTransactionFromReceipt,
						featureGate(config, "ocr_receipt"))...)
					// Voice input: Pro/Business only. Voice consumes audio tokens,
					// so it is gated by the shared token quota too (P1).
					ai.POST("/voice", gated(aiH.ProcessVoice,
						featureGate(config, "voice_input"),
						limitGate(config, "ai_requests_per_cycle"),
						limitGate(config, "ai_tokens_per_cycle"))...)
				}

				userRoutes.POST("/spk/calculate", sh.Calculate)
				// SPK metric override: a user may replace the super-admin default
				// with their own scoring model. DELETE reverts to the default.
				userRoutes.PUT("/spk/metrics", smh.PutMyModel)
				userRoutes.DELETE("/spk/metrics", smh.DeleteMyModel)
				// Budget writes share the advanced_budgeting gate.
				userRoutes.POST("/budgets", gated(bh.Upsert, featureGate(config, "advanced_budgeting"))...)
				userRoutes.DELETE("/budgets/:id", gated(bh.Delete, featureGate(config, "advanced_budgeting"))...)
				userRoutes.GET("/providers", providerH.List)
			}

			adminRoutes := protected.Group("/admin")
			adminRoutes.Use(middleware.RequireAdmin())
			{
				adminCached := adminRoutes.Group("")
				adminCached.Use(middleware.CacheMiddleware(30 * time.Second))
				{
					adminCached.GET("/users", adminH.ListUsers)
					adminCached.GET("/stats", adminH.GetStats)
					adminCached.GET("/subcategories", ch.ListSubcategoriesByQuery)
				}

				adminRoutes.PUT("/users/:id/status", adminH.SetUserStatus)

				adminRoutes.POST("/categories/income", ch.CreateIncome)
				adminRoutes.PUT("/categories/income/:id", ch.UpdateIncome)
				adminRoutes.DELETE("/categories/income/:id", ch.DeleteIncome)

				adminRoutes.POST("/categories/expense", ch.CreateExpense)
				adminRoutes.PUT("/categories/expense/:id", ch.UpdateExpense)
				adminRoutes.DELETE("/categories/expense/:id", ch.DeleteExpense)

				adminRoutes.POST("/subcategories", ch.CreateSubcategory)
				adminRoutes.PUT("/subcategories/:id", ch.UpdateSubcategory)
				adminRoutes.DELETE("/subcategories/:id", ch.DeleteSubcategory)

				adminRoutes.POST("/categories/savings", ch.CreateSavings)
				adminRoutes.PUT("/categories/savings/:id", ch.UpdateSavings)
				adminRoutes.DELETE("/categories/savings/:id", ch.DeleteSavings)

				if paymentH != nil {
					adminRoutes.POST("/payments/manual/:id/approve", paymentH.ApproveManual)
					adminRoutes.POST("/payments/manual/:id/reject", paymentH.RejectManual)
					adminRoutes.GET("/payments/manual/pending", paymentH.ListPendingManual)
				}

				adminRoutes.POST("/providers", providerH.Create)
				adminRoutes.PUT("/providers/:id", providerH.Update)
			}

			// Super Admin portal (P06). Gated on the live `super_admin` role,
			// re-resolved per request so a demotion revokes access instantly.
			if superAdminH != nil {
				sa := protected.Group("/admin")
				sa.Use(middleware.RequireSuperAdmin(roleResolver))
				admin.RegisterRoutes(sa, superAdminH)
				// SPK default metric model: super-admin owned, inherited by every
				// user who has not saved a personal override.
				sa.GET("/spk-metrics", smh.GetDefault)
				sa.PUT("/spk-metrics", smh.PutDefault)
				// Manual payment destinations (bank/e-wallet/QRIS) CRUD.
				sa.GET("/payment-accounts", pah.ListAll)
				sa.POST("/payment-accounts", pah.Create)
				sa.PUT("/payment-accounts/:id", pah.Update)
				sa.DELETE("/payment-accounts/:id", pah.Delete)
			}
		}
	}

	// Health probes. `/health/live` performs no I/O; `/health/ready` checks the
	// real dependencies. `/health` is kept as an alias of live for compatibility.
	healthH := NewHealthHandler(config.Pool, config.Redis)
	r.GET("/health", healthH.Live)
	r.GET("/health/live", healthH.Live)
	r.GET("/health/ready", healthH.Ready)

	// Public avatar images: an <img> tag cannot send the auth cookie, and the
	// filename embeds an unguessable user UUID, so this is served unauthenticated
	// (the handler still verifies the profile points at the requested file).
	r.GET("/api/v1/avatars/:file", ah.ServeAvatar)

	if config.Metrics != nil {
		r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(config.Metrics.Registry(), promhttp.HandlerOpts{})))
	}

	return r
}
