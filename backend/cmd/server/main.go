package main

import (
	"context"
	"encoding/hex"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/admin"
	"github.com/fintrack/backend/internal/aigateway/breaker"
	"github.com/fintrack/backend/internal/aigateway/gateway"
	"github.com/fintrack/backend/internal/audit"
	"github.com/fintrack/backend/internal/billing"
	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/internal/featureflag"
	infraDB "github.com/fintrack/backend/internal/infrastructure/db"
	"github.com/fintrack/backend/internal/infrastructure/embedding"
	"github.com/fintrack/backend/internal/infrastructure/metrics"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/internal/realtime"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/logger"
)

// shutdownTimeout is the graceful-drain budget (default 30s), overridable via
// SHUTDOWN_TIMEOUT in seconds.
func shutdownTimeout() time.Duration {
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 30 * time.Second
}

func main() {
	_ = godotenv.Load()

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger.Init(logLevel)
	logger.Info("Starting FinTrack API server...")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET is required")
	}

	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:3000,http://localhost:3001,http://127.0.0.1:3000"
	}

	cookieSecure := os.Getenv("COOKIE_SECURE") == "true"

	encryptionKeyHex := os.Getenv("ENCRYPTION_KEY")
	if encryptionKeyHex == "" {
		logger.Fatal("ENCRYPTION_KEY (32 bytes hex) is required")
	}
	encryptionKey, err := hex.DecodeString(encryptionKeyHex)
	if err != nil {
		logger.Fatal("ENCRYPTION_KEY must be a valid hex string (64 hex chars = 32 bytes)")
	}
	if len(encryptionKey) != 32 {
		logger.Fatal("ENCRYPTION_KEY must decode to exactly 32 bytes (64 hex characters)")
	}

	bgCtx := context.Background()

	// Redis (cache berversi + pub/sub). Degrades gracefully when unavailable.
	redisCfg := infraRedis.ConfigFromEnv(os.Getenv, func(key string, def int) int {
		if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
			return v
		}
		return def
	})
	redisClient, redisErr := infraRedis.New(bgCtx, redisCfg)
	if redisErr != nil {
		logger.Warn("Redis unavailable — running without cache", zap.Error(redisErr))
	}
	defer redisClient.Close()

	// Install the versioned cache engine (Redis when available, else memory).
	middleware.ConfigureCache(redisClient)

	// Observability: metrics registry (four golden signals + business counters).
	metricsRegistry := metrics.New()

	// Listen for cross-instance cache-invalidation events.
	stopCacheSubscriber := middleware.StartCacheInvalidationSubscriber(bgCtx)

	poolCfg := infraDB.PoolConfigFromEnv(os.Getenv, func(key string, def int) int {
		if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
			return v
		}
		return def
	})
	pool, err := infraDB.NewPool(bgCtx, databaseURL, poolCfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// Audit trail (append-only). Started once the pool is available.
	auditSvc := audit.NewService(audit.NewPostgresRepository(pool), audit.DefaultConfig())
	defer auditSvc.Close()
	_ = auditSvc // consumed by the Super Admin audit viewer (P06-017).

	// Repositories
	txRepo := postgres.NewTransactionRepo(pool)
	incomeCatRepo := postgres.NewIncomeCategoryRepo(pool)
	expenseCatRepo := postgres.NewExpenseCategoryRepo(pool)
	savingsCatRepo := postgres.NewSavingsCategoryRepo(pool)
	subcatRepo := postgres.NewExpenseSubcategoryRepo(pool)
	userRepo := postgres.NewUserRepo(pool)
	geminiKeyRepo := postgres.NewGeminiKeyRepo(pool)
	spkRepo := postgres.NewSPKRepo(pool)
	chatSessionRepo := postgres.NewChatSessionRepo(pool)
	chatMessageRepo := postgres.NewChatMessageRepo(pool)
	providerRepo := postgres.NewProviderRepo(pool)
	refreshRepo := postgres.NewRefreshTokenRepo(pool)
	embedRepo := postgres.NewEmbeddingRepo(pool)
	txManager := postgres.NewTxManager(pool)
	aiGatewayRepo := postgres.NewAIGatewayRepo(pool)

	// Emergency kill switch (P06-016): persisted in `settings`, fanned out over
	// Pub/Sub so every instance stops routing a use case within ~100ms.
	killSwitch := admin.NewKillSwitchStore(pool, infraRedis.NewPubSub(redisClient))
	if err := killSwitch.Load(bgCtx); err != nil {
		logger.Warn("AI kill switch state not loaded (using defaults)", zap.Error(err))
	}
	stopKillSwitchWatch := killSwitch.Watch(bgCtx)
	defer stopKillSwitchWatch()

	// Centralized AI gateway (P03). Attached to the AI usecase when enabled.
	// Realtime SSE (P07 slice B): hub for local streams, Redis pub/sub for
	// cross-instance fan-out, ring-buffer for Last-Event-ID resume.
	realtimeHub := realtime.NewHub()
	realtimePubSub := infraRedis.NewPubSub(redisClient)
	realtimeBuffer := realtime.NewBuffer(redisClient)
	realtimePublisher := realtime.NewPublisher(realtimePubSub, realtimeHub, realtimeBuffer)
	realtimeHandler := realtime.NewHandler(realtimeHub, realtimeBuffer)
	stopRealtimeSub := realtime.NewSubscriber(realtimePubSub, realtimeHub).Start(bgCtx)
	defer stopRealtimeSub()
	stopSweeper := realtimeHub.StartIdleSweeper(bgCtx, 2*time.Minute, 1*time.Minute)
	defer stopSweeper()
	defer realtimeHub.CloseAll()

	var aiGateway *gateway.Service
	aiGateway = gateway.New(aiGatewayRepo, gateway.Config{
		EncryptionKey: encryptionKey,
		KillSwitch:    func(uc domain.UseCase) bool { return killSwitch.Enabled(string(uc)) },
		Breaker: breaker.Config{
			OnStateChange: func(endpointID string, from, to breaker.State) {
				if to != breaker.StateOpen {
					return
				}
				realtimePublisher.PublishAdmins(bgCtx, contracts.NewEvent(
					contracts.EventAIEndpointDown, time.Now().Unix(),
					contracts.AIEndpointDownPayload{
						EndpointID:   endpointID,
						FailureCount: aiGateway.Breaker().Failures(endpointID),
						Status:       string(to),
					}))
			},
		},
	})
	healthChecker := gateway.NewHealthChecker(aiGatewayRepo, gateway.HealthConfig{Interval: 60 * time.Second, Timeout: 5 * time.Second})
	healthChecker.Start(bgCtx)
	defer healthChecker.Stop()

	// One cache engine shared process-wide: version bumps (role demotion,
	// entitlement invalidation) must be visible to every consumer, which the
	// in-memory backend cannot guarantee across separate instances.
	cacheEngine := infraRedis.NewCacheEngine(redisClient)

	// Feature flag gating the centralized gateway (default OFF for safe rollout).
	flagSvc := featureflag.NewService(featureflag.NewPostgresRepository(pool), cacheEngine)
	// The centralized gateway is an architectural switch, not a per-user rollout,
	// so it is read with IsEnabledForPlatform (IsEnabled(…, nil) returns false for
	// any partial rollout — which is how this gateway silently stayed off).
	//
	// Safety guard: switching the platform onto the gateway with no routes
	// configured would send every AI request into a dead end and break AI for all
	// users. So the flag alone is not sufficient — at least one active chat route
	// must exist. With none configured, the gateway stays off and BYOK keeps
	// serving traffic.
	gatewayEnabled := flagSvc.IsEnabledForPlatform(bgCtx, "feature_ai_gateway_centralized")
	if gatewayEnabled {
		// A route alone is not sufficient: the gateway path has no BYOK fallback,
		// so pointing the platform at an endpoint whose health check is failing
		// would break AI for every user. Both a route AND a healthy endpoint are
		// required. "unknown" health does not count — an unprobed endpoint is not
		// evidence that it can serve traffic.
		healthy, err := aiGatewayRepo.HasHealthyEndpoint(bgCtx, domain.UseCaseChat)
		if err != nil {
			logger.Warn("AI gateway: endpoint health check failed; keeping gateway disabled", zap.Error(err))
			healthy = false
		}
		if !healthy {
			hasRoutes, rerr := aiGatewayRepo.HasActiveRoutes(bgCtx, domain.UseCaseChat)
			if rerr == nil && hasRoutes {
				logger.Warn("AI gateway: flag is ON and a chat route exists, but no endpoint is healthy; " +
					"gateway stays disabled (BYOK fallback in use)")
			} else {
				logger.Warn("AI gateway: flag is ON but no active chat route is configured; " +
					"gateway stays disabled (BYOK fallback in use)")
			}
		}
		gatewayEnabled = healthy
	}
	if gatewayEnabled {
		logger.Info("AI gateway centralized: ENABLED")
	} else {
		logger.Info("AI gateway centralized: disabled")
	}

	// Embedder (RAG). Prefer the gateway embedder when the gateway is enabled,
	// otherwise fall back to the direct Gemini embedder (graceful degradation).
	var embedder embedding.Embedder
	if gatewayEnabled {
		embedder = embedding.NewGatewayEmbedder(aiGateway)
	} else {
		direct := embedding.NewGeminiEmbedder(os.Getenv("GEMINI_EMBEDDING_API_KEY"))
		if !direct.Enabled() {
			logger.Warn("GEMINI_EMBEDDING_API_KEY not set — RAG embeddings disabled")
		}
		embedder = direct
	}
	// The emergency stop covers embeddings on every transport: the gateway
	// embedder consults it internally, the direct one is wrapped here.
	embedder = embedding.NewKillSwitchEmbedder(embedder, func() bool {
		return killSwitch.Enabled(string(domain.UseCaseEmbedding))
	})

	// Category-name resolver used to build embedding text.
	catName := func(ctx context.Context, tx *domain.Transaction) string {
		if tx.IncomeCategoryID != nil {
			if c, err := incomeCatRepo.GetByID(ctx, *tx.IncomeCategoryID); err == nil && c != nil {
				return c.Name
			}
		}
		if tx.ExpenseCategoryID != nil {
			if c, err := expenseCatRepo.GetByID(ctx, *tx.ExpenseCategoryID); err == nil && c != nil {
				return c.Name
			}
		}
		if tx.SavingsCategoryID != nil {
			if c, err := savingsCatRepo.GetByID(ctx, *tx.SavingsCategoryID); err == nil && c != nil {
				return c.Name
			}
		}
		return ""
	}

	// Usecases
	spkUC := usecase.NewSPKUsecase(txRepo, spkRepo, txManager)
	spkQueue := usecase.NewSPKQueue(4, func(ctx context.Context, userID uuid.UUID) {
		if _, err := spkUC.Calculate(ctx, userID); err != nil {
			logger.Warn("SPK calculation failed", zap.Error(err))
		}
	})
	transactionUC := usecase.NewTransactionUsecase(txRepo, spkQueue, embedder, embedRepo, catName)
	categoryUC := usecase.NewCategoryUsecase(incomeCatRepo, expenseCatRepo, savingsCatRepo, subcatRepo)
	userUC := usecase.NewUserUsecase(userRepo)
	dashboardUC := usecase.NewDashboardUsecase(txRepo, spkUC)
	geminiKeyUC := usecase.NewGeminiKeyUsecase(geminiKeyRepo, encryptionKey)
	aiUC := usecase.NewAIUsecase(chatSessionRepo, chatMessageRepo, geminiKeyRepo, txRepo, incomeCatRepo, expenseCatRepo, savingsCatRepo, userRepo, encryptionKey, embedder, embedRepo)
	if gatewayEnabled {
		aiUC.WithGateway(aiGateway)
	}
	// The emergency stop applies regardless of which transport serves a model
	// call, so it is wired outside the gateway feature flag.
	aiUC.WithKillSwitch(killSwitch.Enabled)
	// Subscription & entitlement engine (P04).
	billingRepo := postgres.NewBillingRepo(pool)
	entSource := postgres.NewEntitlementSource(billingRepo)
	entResolver := entitlement.NewResolver(entSource, cacheEngine)
	entHandler := handler.NewEntitlementHandler(entResolver)
	entBroadcaster := entitlement.NewBroadcaster(infraRedis.NewPubSub(redisClient), entResolver)
	stopEntSubscriber := entBroadcaster.StartSubscriber(bgCtx)
	defer stopEntSubscriber()

	// Payments & billing (P05).
	paymentRepo := postgres.NewPaymentRepo(pool)
	settingsRepo := postgres.NewSettingsRepo(pool)
	payRegistry := payment.NewRegistry()
	payRegistry.Register(payment.NewMidtransGateway(payment.MidtransConfig{
		ServerKey:    os.Getenv("MIDTRANS_SERVER_KEY"),
		ClientKey:    os.Getenv("MIDTRANS_CLIENT_KEY"),
		IsProduction: os.Getenv("MIDTRANS_IS_PRODUCTION") == "true",
	}))
	payRegistry.Register(payment.NewManualGateway(
		os.Getenv("MANUAL_BANK_NAME"), os.Getenv("MANUAL_ACCOUNT_NUMBER"), os.Getenv("MANUAL_ACCOUNT_HOLDER")))
	payTax := payment.NewTaxCalculator(settingsRepo)
	payInvoice := payment.NewInvoiceService(paymentRepo, settingsRepo, payTax)
	subActivator := usecase.NewSubscriptionActivator(billingRepo, billingRepo, entResolver)
	payWebhook := payment.NewWebhookProcessor(paymentRepo, payRegistry, subActivator, txManager)
	payManual := payment.NewManualService(paymentRepo, subActivator, txManager, nil)
	payAdmin := payment.NewAdminService(payManual)
	payCheckout := payment.NewCheckoutService(paymentRepo, settingsRepo, payRegistry, payInvoice, nil)
	payHandler := handler.NewPaymentHandler(payWebhook).
		WithManual(payManual).
		WithAdmin(payAdmin).
		WithCheckout(payCheckout, billingRepo, userRepo)
	// Realtime producers (P07 slice B).
	payWebhook.SetEmitter(realtimePublisher)
	payManual.SetEmitter(realtimePublisher)

	// Dunning & grace workers.
	dunningWorker := payment.NewDunningWorker(paymentRepo, nil, time.Hour)
	graceWorker := payment.NewGraceWorker(paymentRepo, settingsRepo, nil, 30*time.Minute)
	dunningWorker.Start(bgCtx)
	graceWorker.Start(bgCtx)
	defer dunningWorker.Stop()
	defer graceWorker.Stop()

	providerUC := usecase.NewProviderUsecase(providerRepo)
	reportUC := usecase.NewReportUsecase(txRepo)
	authUC := usecase.NewAuthUsecase(userRepo, refreshRepo, []byte(jwtSecret))

	// Entitlement enforcement (P07 slice A): the middleware existed since P04 but
	// was never constructed, so plan checks never ran. Wire it now, with usage
	// resolvers backed by the transaction and token-period repositories.
	entMiddleware := middleware.NewEntitlementMiddleware(entResolver)
	entUsage := &middleware.UsageProviders{
		CountTransactions: txRepo.CountActiveByUser,
		CurrentTokens:     middleware.TokenPeriodUsage(billingRepo),
		CountVisionScans: func(ctx context.Context, userID uuid.UUID) (int64, error) {
			return billingRepo.CountVisionScansInPeriod(ctx, userID, time.Now().UTC().Unix())
		},
	}

	quotaResolver := billing.QuotaResolverFunc(func(ctx context.Context, userID uuid.UUID, at time.Time) (time.Time, time.Time, int64, error) {
		res, err := entResolver.Resolve(ctx, userID)
		if err != nil {
			return time.Time{}, time.Time{}, 0, err
		}
		quota := res.Limit("ai_tokens_per_cycle")
		if quota < 0 {
			quota = 1000000000 // unlimited
		}
		sub, _ := billingRepo.GetByUserID(ctx, userID)
		if sub != nil && sub.CurrentPeriodEnd != nil {
			return sub.CurrentPeriodStart, *sub.CurrentPeriodEnd, quota, nil
		}
		start := at.Truncate(24 * time.Hour)
		return start, start.AddDate(0, 0, 30), quota, nil
	})
	ledgerService := billing.NewLedgerService(billingRepo, txManager, quotaResolver, nil)

	routerConfig := deliveryHTTP.RouterConfig{
		JWTSecret:    jwtSecret,
		CORSOrigin:   corsOrigin,
		CookieSecure: cookieSecure,
		Metrics:      metricsRegistry,
		Entitlement:  entMiddleware,
		Usage:        entUsage,
		Pool:         pool,
		Redis:        redisClient,
		Ledger:       ledgerService,
	}

	// Super Admin portal (P06): one service + handler shared by every route.
	adminSvc := admin.NewService(pool, auditSvc,
		admin.WithManualService(payManual),
		admin.WithGateway(aiGateway),
		admin.WithKillSwitch(killSwitch),
		admin.WithFeatureFlags(flagSvc),
		admin.WithEncryptionKey(encryptionKey),
		admin.WithCache(cacheEngine),
		admin.WithEntitlementInvalidator(entBroadcaster),
		admin.WithRealtimeEmitter(realtimePublisher),
	)
	superAdminH := admin.NewHandler(adminSvc)
	adminRoleResolver := admin.NewRoleResolver(pool, cacheEngine)

	r := deliveryHTTP.SetupRouter(
		routerConfig,
		transactionUC,
		categoryUC,
		userUC,
		dashboardUC,
		geminiKeyUC,
		spkUC,
		aiUC,
		providerUC,
		reportUC,
		authUC,
		entHandler,
		realtimeHandler,
		payHandler,
		superAdminH,
		adminRoleResolver,
	)

	// Start the SPK worker pool.
	spkQueue.Start(bgCtx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Server listening on :" + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	spkQueue.Stop()
	stopCacheSubscriber()

	// Drain window: long enough for in-flight requests (including SSE streams,
	// which the shutdown closes) to finish. Override with SHUTDOWN_TIMEOUT
	// (seconds) when a platform needs a different budget; keep it in sync with
	// the orchestrator's termination grace period.
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout())
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}
