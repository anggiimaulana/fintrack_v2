package http

import (
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"

	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/usecase"
)

type RouterConfig struct {
	JWTSecret    string
	CORSOrigin   string
	CookieSecure bool
}

// envInt reads an integer env var, falling back to def when unset/invalid.
func envInt(key string, def int64) int64 {
	if v, err := strconv.ParseInt(os.Getenv(key), 10, 64); err == nil && v > 0 {
		return v
	}
	return def
}

func SetupRouter(
	config RouterConfig,
	transactionUC *usecase.TransactionUsecase,
	categoryUC *usecase.CategoryUsecase,
	userUC *usecase.UserUsecase,
	dashboardUC *usecase.DashboardUsecase,
	geminiKeyUC *usecase.GeminiKeyUsecase,
	spkUC *usecase.SPKUsecase,
	aiUC *usecase.AIUsecase,
	providerUC *usecase.ProviderUsecase,
	reportUC *usecase.ReportUsecase,
	authUC *usecase.AuthUsecase,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(config.CORSOrigin))
	r.Use(middleware.RequestLogger())

	middleware.CacheCleanup()

	th := handler.NewTransactionHandler(transactionUC)
	ch := handler.NewCategoryHandler(categoryUC)
	dh := handler.NewDashboardHandler(dashboardUC)
	gkh := handler.NewGeminiKeyHandler(geminiKeyUC)
	sh := handler.NewSPKHandler(spkUC)
	aiH := handler.NewAIHandler(aiUC, transactionUC)
	ah := handler.NewAuthHandler(authUC, config.CookieSecure)
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
		"/api/v1/gemini-keys/:id/test": {Period: 1 * time.Minute, Limit: 10},
	}

	rateLimitMw := middleware.RateLimitMiddlewareByPath(rateLimitMap)

	api := r.Group("/api/v1")
	api.Use(rateLimitMw)
	api.Use(middleware.OriginCheck(config.CORSOrigin))
	{
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
					cached.GET("/gemini-keys", gkh.List)
					cached.GET("/spk/latest", sh.GetLatest)
					cached.GET("/spk/history", sh.GetHistory)
					cached.GET("/reports", rh.Generate)
					cached.GET("/categories/income", ch.ListIncome)
					cached.GET("/categories/expense", ch.ListExpense)
					cached.GET("/categories/expense/:categoryId/subcategories", ch.ListSubcategories)
					cached.GET("/categories/savings", ch.ListSavings)
				}

				userRoutes.GET("/transactions", th.List)
				userRoutes.POST("/transactions", th.Create)
				userRoutes.PUT("/transactions/:id", th.Update)
				userRoutes.DELETE("/transactions/:id", th.Delete)

				userRoutes.POST("/gemini-keys", gkh.Create)
				userRoutes.PATCH("/gemini-keys/:id", gkh.Update)
				userRoutes.DELETE("/gemini-keys/:id", gkh.Delete)
				userRoutes.POST("/gemini-keys/:id/test", gkh.Test)

				ai := userRoutes.Group("/ai")
				{
					ai.POST("/chat", aiH.Chat)
					ai.POST("/sessions", aiH.CreateSession)
					ai.GET("/sessions", aiH.ListSessions)
					ai.PUT("/sessions/:id", aiH.UpdateSession)
					ai.DELETE("/sessions/:id", aiH.DeleteSession)
					ai.GET("/sessions/:id/messages", aiH.ListMessages)
					ai.POST("/receipt", aiH.AnalyzeReceipt)
					ai.POST("/receipt/transaction", aiH.CreateTransactionFromReceipt)
					ai.POST("/voice", aiH.ProcessVoice)
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

				adminRoutes.POST("/providers", providerH.Create)
				adminRoutes.PUT("/providers/:id", providerH.Update)
			}
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "fintrack-api"})
	})

	return r
}
