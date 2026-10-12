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
	// Ledger, when non-nil, enables two-phase token reservations on AI routes.
	Ledger *billing.LedgerService
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
	_ *usecase.GeminiKeyUsecase, // BYOK deprecated (TASK-P07-013)
	spkUC *usecase.SPKUsecase,
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
	aiH := handler.NewAIHandler(aiUC, transactionUC)
	if config.Ledger != nil {
		aiH.WithLedger(config.Ledger)
	}
	ah := handler.NewAuthHandler(authUC, config.CookieSecure)
	if config.Pool != nil {
		ah.WithAnonymize(user.NewAnonymizeService(config.Pool))
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
			protected.DELETE("/me", ah.DeleteAccount)
			protected.POST("/user/delete-account", ah.DeleteAccount)
			protected.GET("/user/export-data", ah.ExportData)
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
					// reports_export: Pro/Business only.
					cached.GET("/reports", gated(rh.Generate, featureGate(config, "reports_export"))...)
					cached.GET("/categories/income", ch.ListIncome)
					cached.GET("/categories/expense", ch.ListExpense)
					cached.GET("/categories/expense/:categoryId/subcategories", ch.ListSubcategories)
					cached.GET("/categories/savings", ch.ListSavings)
				}

				userRoutes.GET("/transactions", th.List)
				// max_transactions: Free 100, Pro 5000, Business unlimited.
				userRoutes.POST("/transactions", gated(th.Create, limitGate(config, "max_transactions"))...)
				userRoutes.PUT("/transactions/:id", th.Update)
				userRoutes.DELETE("/transactions/:id", th.Delete)
				if paymentH != nil {
					userRoutes.POST("/payments/checkout", paymentH.Checkout)
				}
				ai := userRoutes.Group("/ai")
				{
					// ai_chat is in every plan; the token quota is the real gate.
					ai.POST("/chat", gated(aiH.Chat,
						featureGate(config, "ai_chat"),
						limitGate(config, "ai_tokens_per_cycle"))...)
					ai.POST("/sessions", aiH.CreateSession)
					ai.GET("/sessions", aiH.ListSessions)
					ai.PUT("/sessions/:id", aiH.UpdateSession)
					ai.DELETE("/sessions/:id", aiH.DeleteSession)
					ai.GET("/sessions/:id/messages", aiH.ListMessages)
					// OCR: Pro/Business only, bounded by ocr_scans_per_cycle.
					ai.POST("/receipt", gated(aiH.AnalyzeReceipt,
						featureGate(config, "ocr_receipt"),
						limitGate(config, "ocr_scans_per_cycle"))...)
					ai.POST("/receipt/transaction", gated(aiH.CreateTransactionFromReceipt,
						featureGate(config, "ocr_receipt"))...)
					// Voice input: Pro/Business only.
					ai.POST("/voice", gated(aiH.ProcessVoice, featureGate(config, "voice_input"))...)
				}

				userRoutes.POST("/spk/calculate", sh.Calculate)
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
			}
		}
	}

	// Health probes. `/health/live` performs no I/O; `/health/ready` checks the
	// real dependencies. `/health` is kept as an alias of live for compatibility.
	healthH := NewHealthHandler(config.Pool, config.Redis)
	r.GET("/health", healthH.Live)
	r.GET("/health/live", healthH.Live)
	r.GET("/health/ready", healthH.Ready)

	if config.Metrics != nil {
		r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(config.Metrics.Registry(), promhttp.HandlerOpts{})))
	}

	return r
}
